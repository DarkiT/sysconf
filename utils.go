package sysconf

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cast"
)

var (
	// 为名称转换添加缓存以提高性能
	camelToSnakeCache = make(map[string]string)
	snakeToCamelCache = make(map[string]string)
	cacheMutex        sync.RWMutex
)

// 内部方法和辅助函数
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func parseSlice(s string, t reflect.Type) (reflect.Value, error) {
	// 预处理输入字符串
	s = strings.TrimSpace(s)
	if s == "" {
		return reflect.MakeSlice(t, 0, 0), nil
	}

	// 尝试解析 JSON 数组
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var slice []any
		if err := json.Unmarshal([]byte(s), &slice); err == nil {
			return convertJSONArrayToSlice(slice, t)
		}
	}

	// 处理逗号分隔的字符串
	parts := strings.Split(s, ",")
	slice := reflect.MakeSlice(t, len(parts), len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		val := reflect.New(t.Elem()).Elem()

		var err error
		switch t.Elem().Kind() {
		case reflect.String:
			val.SetString(part)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var n int64
			n, err = cast.ToInt64E(part)
			if err == nil {
				val.SetInt(n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var n uint64
			n, err = cast.ToUint64E(part)
			if err == nil {
				val.SetUint(n)
			}
		case reflect.Float32, reflect.Float64:
			var n float64
			n, err = cast.ToFloat64E(part)
			if err == nil {
				val.SetFloat(n)
			}
		case reflect.Bool:
			var b bool
			b, err = cast.ToBoolE(part)
			if err == nil {
				val.SetBool(b)
			}
		default:
			return reflect.Value{}, fmt.Errorf("unsupported slice element type: %s", t.Elem().Kind())
		}

		if err != nil {
			return reflect.Value{}, fmt.Errorf("parse slice element %d: %w", i, err)
		}
		slice.Index(i).Set(val)
	}

	return slice, nil
}

func stringToSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.Slice {
			return data, nil
		}

		str, ok := data.(string)
		if !ok {
			return data, nil
		}

		if str == "" {
			return []string{}, nil
		}

		// 尝试解析 JSON
		var slice []any
		if err := json.Unmarshal([]byte(str), &slice); err == nil {
			return slice, nil
		}

		// 降级为逗号分隔
		return strings.Split(str, ","), nil
	}
}

func stringToMapHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.Map {
			return data, nil
		}

		str, ok := data.(string)
		if !ok {
			return data, nil
		}

		if str == "" {
			return make(map[string]any), nil
		}

		var m map[string]any
		if err := json.Unmarshal([]byte(str), &m); err != nil {
			return nil, fmt.Errorf("invalid map format: %s", str)
		}
		return m, nil
	}
}

func convertJSONArrayToSlice(arr []any, t reflect.Type) (reflect.Value, error) {
	slice := reflect.MakeSlice(t, len(arr), len(arr))

	for i, item := range arr {
		val := reflect.New(t.Elem()).Elem()

		switch t.Elem().Kind() {
		case reflect.String:
			s, err := cast.ToStringE(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetString(s)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := cast.ToInt64E(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := cast.ToUint64E(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetUint(n)
		case reflect.Float32, reflect.Float64:
			n, err := cast.ToFloat64E(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetFloat(n)
		case reflect.Bool:
			b, err := cast.ToBoolE(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetBool(b)
		default:
			return reflect.Value{}, fmt.Errorf("unsupported slice element type: %s", t.Elem().Kind())
		}

		slice.Index(i).Set(val)
	}

	return slice, nil
}

func setDefaultValues(obj any) error {
	if obj == nil {
		return errors.New("nil pointer")
	}

	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr {
		return errors.New("not a pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return errors.New("not a struct")
	}

	return setDefaultValuesRecursive(val)
}

func setDefaultValuesRecursive(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		tag := typ.Field(i).Tag.Get("default")

		if field.Kind() == reflect.Struct {
			if err := setDefaultValuesRecursive(field); err != nil {
				return err
			}
			continue
		}

		if tag != "" && isZero(field) {
			if err := setFieldValue(field, tag); err != nil {
				return fmt.Errorf("set field %s: %w", typ.Field(i).Name, err)
			}
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 特殊处理 time.Duration 类型
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// 先尝试解析为 duration 字符串（例如 "1h30m"）
			if d, err := time.ParseDuration(value); err == nil {
				field.SetInt(int64(d))
				return nil
			}

			// 尝试解析为数字
			v, err := cast.ToInt64E(value)
			if err != nil {
				return fmt.Errorf("invalid duration value: %s", value)
			}

			// 如果是整数，假设它是秒数而不是纳秒
			field.SetInt(v * int64(time.Second))
			return nil
		}

		// 普通整数字段
		v, err := cast.ToInt64E(value)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		field.SetInt(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := cast.ToFloat64E(value)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(v)
		return nil
	case reflect.Bool:
		v, err := cast.ToBoolE(value)
		if err != nil {
			return fmt.Errorf("invalid bool value: %s", value)
		}
		field.SetBool(v)
		return nil
	case reflect.Slice:
		v, err := parseSlice(value, field.Type())
		if err != nil {
			return err
		}
		field.Set(v)
		return nil
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
}

func validateStruct(obj any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 检查当前字段是否必填
		required := fieldType.Tag.Get("required")
		if required == "true" {
			if isZero(field) {
				return fmt.Errorf("field %s is required", fieldType.Name)
			}
		}

		// 递归检查嵌套结构体
		if field.Kind() == reflect.Struct {
			// 为嵌套结构体创建一个新的指针
			nestedPtr := reflect.New(field.Type())
			// 设置指针指向当前字段
			nestedPtr.Elem().Set(field)
			// 递归验证嵌套结构体
			if err := validateStruct(nestedPtr.Interface()); err != nil {
				return fmt.Errorf("nested field %s: %w", fieldType.Name, err)
			}
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			// 递归验证指针指向的结构体
			if err := validateStruct(field.Interface()); err != nil {
				return fmt.Errorf("pointer field %s: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// 驼峰命名转下划线命名
// 例如：CamelCase -> camel_case, AccessID -> access_id
func camelToSnake(s string) string {
	// 检查缓存
	cacheMutex.RLock()
	if v, ok := camelToSnakeCache[s]; ok {
		cacheMutex.RUnlock()
		return v
	}
	cacheMutex.RUnlock()

	// 估算结果字符串长度为原长度的1.5倍
	var result strings.Builder
	result.Grow(int(float64(len(s)) * 1.5))

	for i, c := range s {
		// 如果是大写字母
		if c >= 'A' && c <= 'Z' {
			// 添加下划线的条件:
			// 1. 不是第一个字符
			// 2. 前一个字符不是下划线
			// 3. 前一个字符不是大写字母，或者是大写字母且下一个字符是小写字母
			//    （处理类似AccessID -> access_id的情况，ID作为一个单词）
			if i > 0 && (s[i-1] < 'A' || s[i-1] > 'Z' || (i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z')) {
				result.WriteByte('_')
			}
			// 转换为小写
			result.WriteByte(byte(c - 'A' + 'a'))
		} else {
			result.WriteByte(byte(c))
		}
	}

	resultStr := result.String()

	// 更新缓存
	cacheMutex.Lock()
	camelToSnakeCache[s] = resultStr
	cacheMutex.Unlock()

	return resultStr
}

// 下划线命名转驼峰命名
// 例如：snake_case -> snakeCase, access_id -> accessId
func snakeToCamel(s string) string {
	// 检查缓存
	cacheMutex.RLock()
	if v, ok := snakeToCamelCache[s]; ok {
		cacheMutex.RUnlock()
		return v
	}
	cacheMutex.RUnlock()

	// 处理特殊情况
	if s == "" {
		return ""
	}

	// 处理全下划线的情况
	isAllUnderscores := true
	for i := 0; i < len(s); i++ {
		if s[i] != '_' {
			isAllUnderscores = false
			break
		}
	}
	if isAllUnderscores {
		return ""
	}

	// 估算结果字符串长度
	var result strings.Builder
	result.Grow(len(s))

	// 移除前后下划线
	s = strings.Trim(s, "_")
	if s == "" {
		return ""
	}

	// 分割字符串，处理连续下划线
	parts := strings.Split(s, "_")
	// 过滤空部分（处理连续下划线）
	filteredParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filteredParts = append(filteredParts, part)
		}
	}

	if len(filteredParts) == 0 {
		return ""
	}

	// 第一部分保持小写
	result.WriteString(strings.ToLower(filteredParts[0]))

	// 后续部分首字母大写
	for i := 1; i < len(filteredParts); i++ {
		part := filteredParts[i]
		if part == "" {
			continue
		}
		if len(part) > 0 {
			// 首字母大写
			result.WriteString(strings.ToUpper(part[:1]))
			// 其余部分小写
			if len(part) > 1 {
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}

	resultStr := result.String()

	// 更新缓存
	cacheMutex.Lock()
	snakeToCamelCache[s] = resultStr
	cacheMutex.Unlock()

	return resultStr
}

// 检查字符串是否全部为小写字母
func isAllLower(s string) bool {
	if s == "" {
		return true
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		// 只检查字母字符是否为小写
		// a-z 是小写字母，A-Z 是大写字母
		if c >= 'A' && c <= 'Z' {
			// 如果有大写字母，则不是全小写
			return false
		}
		// 其他字符（数字、下划线、特殊符号等）被忽略
	}
	return true
}
