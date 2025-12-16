package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/darkit/sysconf/internal/cache"
	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cast"
)

// IsZero 检查反射值是否为零值
func IsZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

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
	case reflect.Struct:
		// 对于结构体，检查是否所有字段都是零值
		for i := 0; i < v.NumField(); i++ {
			if !IsZero(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return false
}

// ParseSlice 解析字符串为切片类型
func ParseSlice(s string, t reflect.Type) (reflect.Value, error) {
	// 输入验证
	if t.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("target type must be a slice, got %s", t.Kind())
	}

	// 预处理输入字符串
	s = strings.TrimSpace(s)
	if s == "" {
		return reflect.MakeSlice(t, 0, 0), nil
	}

	// 尝试解析 JSON 数组
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var slice []any
		if err := json.Unmarshal([]byte(s), &slice); err == nil {
			return ConvertJSONArrayToSlice(slice, t)
		}
		// JSON解析失败时继续尝试逗号分隔格式
	}

	// 处理逗号分隔的字符串
	parts := strings.Split(s, ",")
	slice := reflect.MakeSlice(t, 0, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			// 跳过空字符串
			continue
		}

		val := reflect.New(t.Elem()).Elem()
		if err := SetSliceElementValue(val, part, t.Elem()); err != nil {
			return reflect.Value{}, fmt.Errorf("parse slice element %d ('%s'): %w", i, part, err)
		}

		// 扩展切片并设置值
		slice = reflect.Append(slice, val)
	}

	return slice, nil
}

// SetSliceElementValue 设置切片元素的值，带类型安全检查
func SetSliceElementValue(val reflect.Value, part string, elemType reflect.Type) error {
	// 防御性编程：检查val是否可以设置
	if !val.CanSet() {
		return fmt.Errorf("cannot set value for type %s", elemType.Kind())
	}

	switch elemType.Kind() {
	case reflect.String:
		val.SetString(part)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := cast.ToInt64E(part)
		if err != nil {
			return fmt.Errorf("convert to int: %w", err)
		}
		if val.OverflowInt(n) {
			return fmt.Errorf("value %d overflows %s", n, elemType.Kind())
		}
		val.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := cast.ToUint64E(part)
		if err != nil {
			return fmt.Errorf("convert to uint: %w", err)
		}
		if val.OverflowUint(n) {
			return fmt.Errorf("value %d overflows %s", n, elemType.Kind())
		}
		val.SetUint(n)
		return nil
	case reflect.Float32, reflect.Float64:
		n, err := cast.ToFloat64E(part)
		if err != nil {
			return fmt.Errorf("convert to float: %w", err)
		}
		if val.OverflowFloat(n) {
			return fmt.Errorf("value %f overflows %s", n, elemType.Kind())
		}
		val.SetFloat(n)
		return nil
	case reflect.Bool:
		b, err := cast.ToBoolE(part)
		if err != nil {
			return fmt.Errorf("convert to bool: %w", err)
		}
		val.SetBool(b)
		return nil
	default:
		return fmt.Errorf("unsupported slice element type: %s", elemType.Kind())
	}
}

// StringToSliceHookFunc 创建mapstructure的字符串到切片转换hook
func StringToSliceHookFunc() mapstructure.DecodeHookFunc {
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

		// 降级为逗号分隔，过滤空字符串
		parts := strings.Split(str, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result, nil
	}
}

// StringToMapHookFunc 创建mapstructure的字符串到map转换hook
func StringToMapHookFunc() mapstructure.DecodeHookFunc {
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

// ConvertJSONArrayToSlice 将JSON数组转换为指定类型的切片
func ConvertJSONArrayToSlice(arr []any, t reflect.Type) (reflect.Value, error) {
	slice := reflect.MakeSlice(t, 0, len(arr))

	for i, item := range arr {
		val := reflect.New(t.Elem()).Elem()

		if err := SetJSONElementValue(val, item, t.Elem(), i); err != nil {
			return reflect.Value{}, err
		}

		slice = reflect.Append(slice, val)
	}

	return slice, nil
}

// SetJSONElementValue 设置JSON数组元素的值，改进错误处理
func SetJSONElementValue(val reflect.Value, item any, elemType reflect.Type, index int) error {
	switch elemType.Kind() {
	case reflect.String:
		s, err := cast.ToStringE(item)
		if err != nil {
			return fmt.Errorf("element %d: convert to string: %w", index, err)
		}
		val.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := cast.ToInt64E(item)
		if err != nil {
			return fmt.Errorf("element %d: convert to int: %w", index, err)
		}
		if val.OverflowInt(n) {
			return fmt.Errorf("element %d: value %d overflows %s", index, n, elemType.Kind())
		}
		val.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := cast.ToUint64E(item)
		if err != nil {
			return fmt.Errorf("element %d: convert to uint: %w", index, err)
		}
		if val.OverflowUint(n) {
			return fmt.Errorf("element %d: value %d overflows %s", index, n, elemType.Kind())
		}
		val.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := cast.ToFloat64E(item)
		if err != nil {
			return fmt.Errorf("element %d: convert to float: %w", index, err)
		}
		if val.OverflowFloat(n) {
			return fmt.Errorf("element %d: value %f overflows %s", index, n, elemType.Kind())
		}
		val.SetFloat(n)
	case reflect.Bool:
		b, err := cast.ToBoolE(item)
		if err != nil {
			return fmt.Errorf("element %d: convert to bool: %w", index, err)
		}
		val.SetBool(b)
	default:
		return fmt.Errorf("element %d: unsupported slice element type: %s", index, elemType.Kind())
	}
	return nil
}

// SetDefaultValues 为结构体设置默认值
func SetDefaultValues(obj any) error {
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

	return SetDefaultValuesRecursive(val)
}

// SetDefaultValuesRecursive 递归设置默认值
func SetDefaultValuesRecursive(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		tag := typ.Field(i).Tag.Get("default")

		if field.Kind() == reflect.Struct {
			if err := SetDefaultValuesRecursive(field); err != nil {
				return err
			}
			continue
		}

		if tag != "" && IsZero(field) {
			if err := SetFieldValue(field, tag); err != nil {
				return fmt.Errorf("set field %s: %w", typ.Field(i).Name, err)
			}
		}
	}

	return nil
}

// SetFieldValue 设置字段值
func SetFieldValue(field reflect.Value, value string) error {
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
		v, err := ParseSlice(value, field.Type())
		if err != nil {
			return err
		}
		field.Set(v)
		return nil
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
}

// ValidateStruct 验证结构体字段
func ValidateStruct(obj any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 检查当前字段是否必填（支持 required:"true" / required:"required" / validate:"required"）
		if requiredTag := fieldType.Tag.Get("required"); requiredTag == "true" || requiredTag == "required" {
			if IsZero(field) {
				return fmt.Errorf("field %s is required", fieldType.Name)
			}
		}
		if validateTag := fieldType.Tag.Get("validate"); strings.Contains(validateTag, "required") {
			if IsZero(field) {
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
			if err := ValidateStruct(nestedPtr.Interface()); err != nil {
				return fmt.Errorf("nested field %s: %w", fieldType.Name, err)
			}
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			// 递归验证指针指向的结构体
			if err := ValidateStruct(field.Interface()); err != nil {
				return fmt.Errorf("pointer field %s: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// CamelToSnake 驼峰命名转下划线命名
// 例如：CamelCase -> camel_case, AccessID -> access_id
func CamelToSnake(s string) string {
	// 输入验证
	if s == "" {
		return ""
	}

	// 检查缓存
	if v, ok := cache.GlobalManager.GetCamelToSnake(s); ok {
		return v
	}

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
	cache.GlobalManager.SetCamelToSnake(s, resultStr)

	return resultStr
}

// SnakeToCamel 下划线命名转驼峰命名
// 例如：snake_case -> snakeCase, access_id -> accessId
func SnakeToCamel(s string) string {
	// 输入验证
	if s == "" {
		return ""
	}

	// 检查缓存
	if v, ok := cache.GlobalManager.GetSnakeToCamel(s); ok {
		return v
	}

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
	cache.GlobalManager.SetSnakeToCamel(s, resultStr)

	return resultStr
}

// IsAllLower 检查字符串是否全部为小写字母
func IsAllLower(s string) bool {
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
