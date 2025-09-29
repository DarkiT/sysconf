package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ValidationRule 验证规则
type ValidationRule struct {
	// Type 验证类型
	Type string

	// Value 验证值
	Value string

	// Message 错误消息
	Message string
}

// Validate 验证值
func Validate(value interface{}, rule ValidationRule) error {
	var err error
	switch rule.Type {
	case "required":
		if isZero(reflect.ValueOf(value)) {
			return fmt.Errorf("%s", rule.Message)
		}
	case "min":
		if err = validateMin(value, rule.Value); err != nil {
			return fmt.Errorf("%s", rule.Message)
		}
	case "max":
		if err = validateMax(value, rule.Value); err != nil {
			return fmt.Errorf("%s", rule.Message)
		}
	case "range":
		if err = validateRangeValue(value, rule.Value); err != nil {
			return fmt.Errorf("%s", rule.Message)
		}
	case "length":
		if err = validateLengthValue(value, rule.Value); err != nil {
			return fmt.Errorf("%s", rule.Message)
		}
	case "pattern":
		if err = validatePattern(value, rule.Value); err != nil {
			return fmt.Errorf("%s", rule.Message)
		}
	case "enum":
		if err = validateEnumValue(value, rule.Value); err != nil {
			return fmt.Errorf("%s", rule.Message)
		}
	default:
		// 使用 rules.go 中的验证器处理其他类型
		ruleStr := rule.Type
		if rule.Value != "" {
			ruleStr = rule.Type + ":" + rule.Value
		}
		if valid, errMsg := ValidateValue(value, ruleStr); !valid {
			if rule.Message != "" {
				return fmt.Errorf("%s", rule.Message)
			}
			return fmt.Errorf("%s", errMsg)
		}
	}
	return nil
}

// isZero 检查值是否为零值
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

// validateMin 验证最小值
func validateMin(value interface{}, min string) error {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		minVal, err := strconv.ParseInt(min, 10, 64)
		if err != nil {
			return err
		}
		if v.Int() < minVal {
			return fmt.Errorf("value must be greater than or equal to %d", minVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		minVal, err := strconv.ParseUint(min, 10, 64)
		if err != nil {
			return err
		}
		if v.Uint() < minVal {
			return fmt.Errorf("value must be greater than or equal to %d", minVal)
		}
	case reflect.Float32, reflect.Float64:
		minVal, err := strconv.ParseFloat(min, 64)
		if err != nil {
			return err
		}
		if v.Float() < minVal {
			return fmt.Errorf("value must be greater than or equal to %f", minVal)
		}
	case reflect.String:
		minLen, err := strconv.Atoi(min)
		if err != nil {
			return err
		}
		if v.Len() < minLen {
			return fmt.Errorf("length must be greater than or equal to %d", minLen)
		}
	default:
		return fmt.Errorf("unsupported type: %v", v.Kind())
	}
	return nil
}

// validateMax 验证最大值
func validateMax(value interface{}, max string) error {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		maxVal, err := strconv.ParseInt(max, 10, 64)
		if err != nil {
			return err
		}
		if v.Int() > maxVal {
			return fmt.Errorf("value must be less than or equal to %d", maxVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		maxVal, err := strconv.ParseUint(max, 10, 64)
		if err != nil {
			return err
		}
		if v.Uint() > maxVal {
			return fmt.Errorf("value must be less than or equal to %d", maxVal)
		}
	case reflect.Float32, reflect.Float64:
		maxVal, err := strconv.ParseFloat(max, 64)
		if err != nil {
			return err
		}
		if v.Float() > maxVal {
			return fmt.Errorf("value must be less than or equal to %f", maxVal)
		}
	case reflect.String:
		maxLen, err := strconv.Atoi(max)
		if err != nil {
			return err
		}
		if v.Len() > maxLen {
			return fmt.Errorf("length must be less than or equal to %d", maxLen)
		}
	default:
		return fmt.Errorf("unsupported type: %v", v.Kind())
	}
	return nil
}

// validateRangeValue 验证范围
func validateRangeValue(value interface{}, rangeStr string) error {
	parts := strings.Split(rangeStr, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid range format: %s", rangeStr)
	}

	if err := validateMin(value, parts[0]); err != nil {
		return err
	}
	if err := validateMax(value, parts[1]); err != nil {
		return err
	}
	return nil
}

// validateLengthValue 验证长度
func validateLengthValue(value interface{}, length string) error {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		l, err := strconv.Atoi(length)
		if err != nil {
			return err
		}
		if v.Len() != l {
			return fmt.Errorf("length must be equal to %d", l)
		}
	default:
		return fmt.Errorf("unsupported type: %v", v.Kind())
	}
	return nil
}

// validatePattern 验证正则表达式
func validatePattern(value interface{}, pattern string) error {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.String {
		return fmt.Errorf("pattern validation can only be used on string types")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regular expression: %s", pattern)
	}

	if !re.MatchString(v.String()) {
		return fmt.Errorf("does not match pattern: %s", pattern)
	}
	return nil
}

// validateEnumValue 验证枚举值
func validateEnumValue(value interface{}, enumStr string) error {
	v := reflect.ValueOf(value)
	enumValues := strings.Split(enumStr, ",")

	switch v.Kind() {
	case reflect.String:
		strValue := v.String()
		for _, enum := range enumValues {
			if strValue == strings.TrimSpace(enum) {
				return nil
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue := v.Int()
		for _, enum := range enumValues {
			enumInt, err := strconv.ParseInt(strings.TrimSpace(enum), 10, 64)
			if err != nil {
				continue
			}
			if intValue == enumInt {
				return nil
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue := v.Uint()
		for _, enum := range enumValues {
			enumUint, err := strconv.ParseUint(strings.TrimSpace(enum), 10, 64)
			if err != nil {
				continue
			}
			if uintValue == enumUint {
				return nil
			}
		}
	case reflect.Float32, reflect.Float64:
		floatValue := v.Float()
		for _, enum := range enumValues {
			enumFloat, err := strconv.ParseFloat(strings.TrimSpace(enum), 64)
			if err != nil {
				continue
			}
			if floatValue == enumFloat {
				return nil
			}
		}
	default:
		return fmt.Errorf("unsupported type: %v", v.Kind())
	}

	return fmt.Errorf("value must be one of: %s", enumStr)
}

// NewRule 创建验证规则
func NewRule(typ string, value string, message string) ValidationRule {
	return ValidationRule{
		Type:    typ,
		Value:   value,
		Message: message,
	}
}

// Required 创建必填验证规则
func Required(message string) ValidationRule {
	return NewRule("required", "", message)
}

// Min 创建最小值验证规则
func Min(min string, message string) ValidationRule {
	return NewRule("min", min, message)
}

// Max 创建最大值验证规则
func Max(max string, message string) ValidationRule {
	return NewRule("max", max, message)
}

// Range 创建范围验证规则
func Range(min, max string, message string) ValidationRule {
	return NewRule("range", fmt.Sprintf("%s,%s", min, max), message)
}

// Length 创建长度验证规则
func Length(length string, message string) ValidationRule {
	return NewRule("length", length, message)
}

// Pattern 创建正则表达式验证规则
func Pattern(pattern string, message string) ValidationRule {
	return NewRule("pattern", pattern, message)
}

// Enum 创建枚举验证规则
func Enum(values string, message string) ValidationRule {
	return NewRule("enum", values, message)
}

// ValidateStruct 验证结构体
func ValidateStruct(v interface{}, rules map[string][]ValidationRule) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("can only validate struct types")
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if fieldRules, exists := rules[fieldType.Name]; exists {
			for _, rule := range fieldRules {
				if rule.Type == "required" && isZero(field) {
					return fmt.Errorf("field %s is required", fieldType.Name)
				}

				if !isZero(field) {
					if err := Validate(field.Interface(), rule); err != nil {
						return fmt.Errorf("field %s validation failed: %s", fieldType.Name, err)
					}
				}
			}
		}
	}

	return nil
}
