package sysconf

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// 有效配置优先级（高 → 低）：
// 显式 Set > 命令行 flag > 环境变量 > 文件/内联内容 > getter/struct 默认值。
// Get* / GetAs / Unmarshal / AllSettings / Load 必须走同一条路径。

// ChangeHandler 热重载回调：prev/next 为有效配置快照。
// 重载或校验失败时 err != nil，此时不应把 next 当作已生效配置。
type ChangeHandler func(prev, next map[string]any, err error)

func (c *Config) loadOverrides() map[string]any {
	if c == nil {
		return nil
	}
	if v := c.overrides.Load(); v != nil {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

func (c *Config) loadFlags() map[string]any {
	if c == nil {
		return nil
	}
	if v := c.flagSnapshot.Load(); v != nil {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

func lookupMapValue(m map[string]any, key string) (any, bool) {
	if m == nil || key == "" {
		return nil, false
	}
	if v, ok := m[key]; ok {
		return v, true
	}
	lower := strings.ToLower(key)
	if v, ok := m[lower]; ok {
		return v, true
	}
	if isEnvStyleKey(key) {
		if dotted := strings.ReplaceAll(lower, "_", "."); dotted != lower {
			if v, ok := m[dotted]; ok {
				return v, true
			}
		}
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func isEnvStyleKey(key string) bool {
	if key == "" || !strings.Contains(key, "_") || strings.Contains(key, ".") {
		return false
	}
	for _, r := range key {
		if r >= 'a' && r <= 'z' {
			return false
		}
	}
	return true
}

func (c *Config) lookupOverride(key string) (any, bool) {
	return lookupMapValue(c.loadOverrides(), key)
}

func (c *Config) lookupFlagValue(key string) (any, bool) {
	if len(c.pflags) == 0 {
		return nil, false
	}
	return lookupMapValue(c.collectFlags(), key)
}

func (c *Config) commitOverrideLocked(key string, value any) {
	current := c.loadOverrides()
	next := make(map[string]any, len(current)+1)
	prefix := key + "."
	for k, v := range current {
		if k == key || strings.HasPrefix(k, prefix) {
			continue
		}
		next[k] = v
	}
	c.mergeValueIntoData(next, key, value)
	c.overrides.Store(next)
}

func (c *Config) mergePersistentFlat() map[string]any {
	flat := deepCloneMap(c.loadData())
	if flat == nil {
		flat = make(map[string]any)
	}
	for k, v := range c.loadOverrides() {
		flat[k] = deepCloneValue(v)
	}
	return flat
}

func (c *Config) mergeEffectiveFlat() map[string]any {
	flat := c.mergePersistentFlat()
	if c.envEnabled.Load() {
		for k := range flat {
			if val, ok := c.lookupEnvValue(k); ok {
				flat[k] = val
			}
		}
		for _, key := range c.envOnlyConfigKeys() {
			if _, exists := flat[key]; exists {
				continue
			}
			if val, ok := c.lookupEnvValue(key); ok {
				flat[key] = val
			}
		}
	}
	for k, v := range c.collectFlags() {
		flat[k] = deepCloneValue(v)
	}
	for k, v := range c.loadOverrides() {
		flat[k] = deepCloneValue(v)
	}
	return flat
}

func (c *Config) snapshotPersistentSettings() map[string]any {
	if c == nil {
		return nil
	}
	return c.reconstructNestedStructure(c.mergePersistentFlat())
}

func (c *Config) snapshotFlagsLocked() {
	out := c.collectFlags()
	if out == nil {
		out = map[string]any{}
	}
	c.flagSnapshot.Store(out)
	c.flagEnabled.Store(len(c.pflags) > 0)
}

func (c *Config) collectFlags() map[string]any {
	if c == nil || len(c.pflags) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any)
	for _, flagSet := range c.pflags {
		if flagSet == nil {
			continue
		}
		flagSet.VisitAll(func(f *pflag.Flag) {
			if f == nil {
				return
			}
			if c.pflagOptions.OnlyChanged && !f.Changed {
				return
			}
			if c.pflagOptions.Validate != nil {
				if err := c.pflagOptions.Validate(f); err != nil {
					c.logger.Errorf("Invalid flag %s: %v", f.Name, err)
					return
				}
			}
			key := f.Name
			if c.pflagOptions.KeyMapper != nil {
				key = c.pflagOptions.KeyMapper(f)
			}
			if key == "" {
				return
			}
			out[key] = nativeFlagValue(f)
		})
	}
	return out
}

func nativeFlagValue(f *pflag.Flag) any {
	if f == nil || f.Value == nil {
		return nil
	}
	raw := f.Value.String()
	switch f.Value.Type() {
	case "bool":
		v, err := strconv.ParseBool(raw)
		if err == nil {
			return v
		}
	case "int", "int8", "int16", "int32", "int64", "count":
		v, err := strconv.ParseInt(raw, 0, 64)
		if err == nil {
			return int(v)
		}
	case "uint", "uint8", "uint16", "uint32", "uint64":
		v, err := strconv.ParseUint(raw, 0, 64)
		if err == nil {
			return uint(v)
		}
	case "float32", "float64":
		v, err := strconv.ParseFloat(raw, 64)
		if err == nil {
			return v
		}
	case "duration":
		v, err := time.ParseDuration(raw)
		if err == nil {
			return v
		}
	case "stringSlice":
		if sv, ok := f.Value.(interface{ GetSlice() []string }); ok {
			return append([]string(nil), sv.GetSlice()...)
		}
	}
	return raw
}

func (c *Config) validateOnInit() error {
	if c.skipInitValidation {
		return nil
	}
	return c.validateMerged()
}

func (c *Config) validateMerged() error {
	if len(c.validators) == 0 {
		return nil
	}
	settings := c.snapshotAllSettings()
	for _, validator := range c.validators {
		if validator == nil {
			continue
		}
		if err := validator.Validate(settings); err != nil {
			return fmt.Errorf("%w (%s): %v", ErrValidation, validator.GetName(), err)
		}
	}
	return nil
}

var (
	timeTimeType = reflect.TypeFor[time.Time]()
	durationType = reflect.TypeFor[time.Duration]()
)

func collectConfigKeys(t reflect.Type, prefix string) []string {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}

	keys := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		name := configFieldName(field)
		if name == "-" {
			continue
		}
		full := name
		if prefix != "" {
			full = prefix + "." + name
		}
		ft := field.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && ft != timeTimeType && !ft.ConvertibleTo(durationType) {
			keys = append(keys, collectConfigKeys(ft, full)...)
			continue
		}
		keys = append(keys, full)
	}
	return keys
}

func configFieldName(field reflect.StructField) string {
	for _, tag := range []string{"config", "sysconf", "yaml", "json", "toml", "mapstructure"} {
		if v, ok := field.Tag.Lookup(tag); ok {
			name := strings.Split(v, ",")[0]
			if name != "" {
				return name
			}
		}
	}
	return camelToSnake(field.Name)
}

func (c *Config) injectStructLayerValues(input any, obj any, prefix string) any {
	nested, ok := input.(map[string]any)
	if !ok {
		if input != nil {
			return input
		}
		nested = make(map[string]any)
	} else {
		nested = deepCloneMap(nested)
	}

	relPrefix := ""
	if prefix != "" {
		relPrefix = prefix + "."
	}
	for _, key := range collectConfigKeys(reflect.TypeOf(obj), "") {
		full := key
		if relPrefix != "" {
			full = relPrefix + key
		}
		val, exists := c.getRaw(full)
		if !exists {
			continue
		}
		c.setNestedValue(nested, key, sanitizeValue(val))
	}
	return nested
}
