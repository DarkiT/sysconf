package sysconf

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/darkit/sysconf/internal/utils"
	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// matchNameCache 缓存字段名匹配结果，避免重复计算
var matchNameCache sync.Map // map[string]bool, key format: "mapKey|fieldName"

// cachedMatchName 带缓存的字段名匹配函数
func cachedMatchName(mapKey, fieldName string) bool {
	cacheKey := mapKey + "|" + fieldName
	if cached, ok := matchNameCache.Load(cacheKey); ok {
		return cached.(bool)
	}

	result := matchNameUncached(mapKey, fieldName)
	matchNameCache.Store(cacheKey, result)
	return result
}

// matchNameUncached 不带缓存的字段名匹配逻辑
func matchNameUncached(mapKey, fieldName string) bool {
	// 1) 精确匹配
	if mapKey == fieldName {
		return true
	}

	// 2) 忽略大小写匹配
	if strings.EqualFold(mapKey, fieldName) {
		return true
	}

	// 3) 驼峰 ↔ 蛇形（大小写无关）匹配
	snakeField := camelToSnake(fieldName)
	if mapKey == snakeField || strings.EqualFold(mapKey, snakeField) {
		return true
	}

	camelMap := snakeToCamel(mapKey)
	return camelMap == fieldName || strings.EqualFold(camelMap, fieldName)
}

// Unmarshal 将配置解析到结构体
// key 为空时解析整个配置，否则解析指定的配置段
// 支持 default 标签设置默认值
// 支持 required 标签验证必填字段
// 支持大驼峰命名风格的结构体字段自动映射到下划线风格的配置键名
func (c *Config) Unmarshal(obj any, key ...string) error {
	isStructPtr, err := validateUnmarshalTarget(obj)
	if err != nil {
		return err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 如果是结构体指针，则设置默认值
	if isStructPtr {
		c.logger.Debugf("Setting default values")
		if err := setDefaultValues(obj); err != nil {
			c.logger.Errorf("Failed to set default values: %v", err)
			return fmt.Errorf("set defaults: %w", err)
		}
	}

	// 创建解码器配置
	c.logger.Debugf("Creating decoder config")
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(time.RFC3339),
			stringToSliceHookFunc(),
			stringToMapHookFunc(),
		),
		Result:           obj,
		ZeroFields:       false,
		WeaklyTypedInput: true,
		TagName:          strings.Join([]string{"config", "sysconf", strings.Join(viper.SupportedExts, ", ")}, ","),
		SquashTagOption:  "inline",
		// 启用字段名到键名的自动转换，支持驼峰命名到下划线命名的转换
		MatchName: cachedMatchName,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		c.logger.Errorf("Failed to create decoder: %v", err)
		return fmt.Errorf("create decoder: %w", err)
	}

	// 获取配置数据
	var decodeInput any
	if len(key) > 0 && key[0] != "" {
		configKey := strings.Join(key, ".")
		c.logger.Debugf("Getting sub-config: %s", configKey)
		if val, exists := c.getRaw(configKey); exists {
			decodeInput = val
		}
	} else {
		c.logger.Debugf("Getting all config settings")
		decodeInput = c.snapshotAllSettings()
	}

	// 如果没有配置数据，保持默认值
	if isEmptyUnmarshalInput(decodeInput) {
		c.logger.Warnf("No config data found, using default values")
		return nil
	}

	// 解码配置
	c.logger.Debugf("Decoding config")
	if err := decoder.Decode(decodeInput); err != nil {
		c.logger.Errorf("Failed to decode config: %v", err)
		errStr := err.Error()
		// 区分不同类型的解码错误
		if strings.Contains(errStr, "cannot parse") || strings.Contains(errStr, "cannot convert") {
			return fmt.Errorf("type conversion failed: %w", err)
		}
		if strings.Contains(errStr, "has no field") || strings.Contains(errStr, "unknown field") {
			return fmt.Errorf("unknown field: %w", err)
		}
		if strings.Contains(errStr, "required") {
			return fmt.Errorf("required validation failed: %w", err)
		}
		return fmt.Errorf("decode failed: %w", err)
	}

	// 如果是结构体指针，则验证必填字段
	if isStructPtr {
		c.logger.Debugf("Validating required fields")
		if err := utils.ValidateStruct(obj); err != nil {
			c.logger.Errorf("Field validation failed: %v", err)
			return fmt.Errorf("validate: %w", err)
		}
	}

	c.logger.Infof("Config parsed successfully")
	return nil
}

func isEmptyUnmarshalInput(input any) bool {
	if input == nil {
		return true
	}

	switch v := input.(type) {
	case map[string]any:
		return len(v) == 0
	case []any:
		return len(v) == 0
	case []string:
		return len(v) == 0
	}

	return false
}

func validateUnmarshalTarget(obj any) (bool, error) {
	if obj == nil {
		return false, fmt.Errorf("unmarshal target cannot be nil")
	}

	targetType := reflect.TypeOf(obj)
	if targetType.Kind() != reflect.Ptr {
		return false, fmt.Errorf("unmarshal target must be a pointer")
	}

	targetValue := reflect.ValueOf(obj)
	if targetValue.IsNil() {
		return false, fmt.Errorf("unmarshal target pointer cannot be nil")
	}

	return targetType.Elem().Kind() == reflect.Struct, nil
}

// setDefaultValues 设置默认值
func setDefaultValues(obj any) error {
	return utils.SetDefaultValues(obj)
}

func stringToSliceHookFunc() mapstructure.DecodeHookFunc {
	return utils.StringToSliceHookFunc()
}

func stringToMapHookFunc() mapstructure.DecodeHookFunc {
	return utils.StringToMapHookFunc()
}

func camelToSnake(s string) string {
	return utils.CamelToSnake(s)
}

func snakeToCamel(s string) string {
	return utils.SnakeToCamel(s)
}
