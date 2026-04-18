package validation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RuleValidator 验证规则函数类型
type RuleValidator func(value any, params string) (bool, string)

// validatorsMu 保护 validators map 的读写锁
var validatorsMu sync.RWMutex

// 预定义的验证规则映射
var validators = map[string]RuleValidator{
	"required":    validateRequired,
	"string":      validateString,
	"number":      validateNumber,
	"email":       validateEmail,
	"url":         validateURL,
	"range":       validateRange,
	"length":      validateLength,
	"regex":       validateRegex,
	"enum":        validateEnum,
	"ipv4":        validateIPv4,
	"ipv6":        validateIPv6,
	"port":        validatePort,
	"hostname":    validateHostname,
	"alphanum":    validateAlphaNum,
	"uuid":        validateUUID,
	"json":        validateJSON,
	"base64":      validateBase64,
	"datetime":    validateDateTime,
	"timezone":    validateTimezone,
	"creditcard":  validateCreditCard,
	"phonenumber": validatePhoneNumber,
}

// RegisterValidator 注册自定义验证规则
func RegisterValidator(name string, validator RuleValidator) {
	validatorsMu.Lock()
	defer validatorsMu.Unlock()
	validators[name] = validator
}

// ValidateValue 验证值是否符合规则
func ValidateValue(value any, rule string) (bool, string) {
	parts := strings.SplitN(rule, ":", 2)
	ruleName := parts[0]
	params := ""
	if len(parts) > 1 {
		params = parts[1]
	}

	validatorsMu.RLock()
	validator, ok := validators[ruleName]
	validatorsMu.RUnlock()

	if !ok {
		return false, fmt.Sprintf("unknown validation rule: %s", ruleName)
	}

	return validator(value, params)
}

// validateRequired 验证必填字段
func validateRequired(value any, _ string) (bool, string) {
	if value == nil {
		return false, "field cannot be empty"
	}
	if str, ok := value.(string); ok && str == "" {
		return false, "field cannot be empty"
	}
	return true, ""
}

// validateString 验证字符串类型
func validateString(value any, _ string) (bool, string) {
	_, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	return true, ""
}

// validateNumber 验证数字类型
func validateNumber(value any, _ string) (bool, string) {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true, ""
	case string:
		// 尝试解析字符串为数字
		if v != "" {
			if _, err := strconv.ParseFloat(v, 64); err == nil {
				return true, ""
			}
		}
		return false, "field must be number type"
	}
	return false, "field must be number type"
}

// emailRegex 预编译的邮箱验证正则表达式
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9._%+-]*[a-zA-Z0-9])?@[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?\.[a-zA-Z]{2,}$`)

// validateEmail 验证电子邮件地址
func validateEmail(value any, _ string) (bool, string) {
	email, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	// 检查连续点号
	if strings.Contains(email, "..") {
		return false, "invalid email address"
	}
	if !emailRegex.MatchString(email) {
		return false, "invalid email address"
	}
	return true, ""
}

// validateURL 验证 URL
func validateURL(value any, _ string) (bool, string) {
	urlStr, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	_, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return false, "invalid URL"
	}
	return true, ""
}

// validateRange 验证数值范围
func validateRange(value any, params string) (bool, string) {
	var num float64
	switch v := value.(type) {
	case int:
		num = float64(v)
	case int8:
		num = float64(v)
	case int16:
		num = float64(v)
	case int32:
		num = float64(v)
	case int64:
		num = float64(v)
	case uint:
		num = float64(v)
	case uint8:
		num = float64(v)
	case uint16:
		num = float64(v)
	case uint32:
		num = float64(v)
	case uint64:
		num = float64(v)
	case float32:
		num = float64(v)
	case float64:
		num = v
	default:
		return false, "field must be number type"
	}

	parts := strings.Split(params, ",")
	if len(parts) != 2 {
		return false, "invalid range parameters"
	}

	min, err1 := strconv.ParseFloat(parts[0], 64)
	max, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil {
		return false, "invalid range parameters"
	}

	if num < min || num > max {
		return false, fmt.Sprintf("value must be between %v and %v", min, max)
	}
	return true, ""
}

// validateLength 验证字符串长度
func validateLength(value any, params string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}

	parts := strings.Split(params, ",")
	if len(parts) != 2 {
		return false, "invalid length parameters"
	}

	min, err1 := strconv.Atoi(parts[0])
	max, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false, "invalid length parameters"
	}

	length := len(str)
	if length < min || length > max {
		return false, fmt.Sprintf("string length must be between %d and %d", min, max)
	}
	return true, ""
}

// validateRegex 验证正则表达式
func validateRegex(value any, pattern string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, "invalid regular expression"
	}

	if !re.MatchString(str) {
		return false, "string does not match regular expression"
	}
	return true, ""
}

// validateEnum 验证枚举值
func validateEnum(value any, params string) (bool, string) {
	str := fmt.Sprintf("%v", value)
	validValues := strings.Split(params, ",")
	for _, v := range validValues {
		if str == v {
			return true, ""
		}
	}
	return false, fmt.Sprintf("value must be one of: %s", params)
}

// validateIPv4 验证 IPv4 地址
func validateIPv4(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	re := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !re.MatchString(str) {
		return false, "invalid IPv4 address"
	}
	// 验证每个段的值是否在 0-255 之间
	parts := strings.Split(str, ".")
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return false, "invalid IPv4 address"
		}
	}
	return true, ""
}

// validateIPv6 验证 IPv6 地址
// 支持完整形式和压缩形式（如 ::1, fe80::1, 2001:db8::1）
func validateIPv6(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	ip := net.ParseIP(str)
	if ip == nil {
		return false, "invalid IPv6 address"
	}
	// 确保是 IPv6 而非 IPv4
	if ip.To4() != nil {
		return false, "invalid IPv6 address"
	}
	return true, ""
}

// validatePort 验证端口号
func validatePort(value any, _ string) (bool, string) {
	var port int
	switch v := value.(type) {
	case int:
		port = v
	case float64:
		port = int(v)
	case string:
		var err error
		port, err = strconv.Atoi(v)
		if err != nil {
			return false, "port number must be numeric"
		}
	default:
		return false, "port number must be numeric"
	}
	if port < 1 || port > 65535 {
		return false, "port number must be between 1-65535"
	}
	return true, ""
}

// hostnameRegex 预编译的主机名验证正则表达式
var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// validateHostname 验证主机名
func validateHostname(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	// 限制长度防止 ReDoS 攻击
	if len(str) > 253 {
		return false, "hostname too long (max 253 characters)"
	}
	if !hostnameRegex.MatchString(str) {
		return false, "invalid hostname"
	}
	return true, ""
}

// validateAlphaNum 验证字母数字
func validateAlphaNum(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !re.MatchString(str) {
		return false, "field can only contain letters and numbers"
	}
	return true, ""
}

// validateUUID 验证 UUID
func validateUUID(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	re := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !re.MatchString(str) {
		return false, "invalid UUID format"
	}
	return true, ""
}

// validateJSON 验证 JSON 字符串
func validateJSON(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	var js json.RawMessage
	if err := json.Unmarshal([]byte(str), &js); err != nil {
		return false, "invalid JSON format"
	}
	return true, ""
}

// validateBase64 验证 Base64 编码
func validateBase64(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	// 使用标准库验证，确保长度和填充正确
	_, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return false, "invalid Base64 encoding"
	}
	return true, ""
}

// validateDateTime 验证日期时间格式
func validateDateTime(value any, format string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	_, err := time.Parse(format, str)
	if err != nil {
		return false, "invalid datetime format"
	}
	return true, ""
}

// validateTimezone 验证时区
func validateTimezone(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	_, err := time.LoadLocation(str)
	if err != nil {
		return false, "invalid timezone"
	}
	return true, ""
}

// validateCreditCard 验证信用卡号
func validateCreditCard(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	// 移除空格和破折号
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ReplaceAll(str, "-", "")

	// 验证长度
	if len(str) < 13 || len(str) > 19 {
		return false, "invalid credit card number"
	}

	// Luhn 算法验证
	sum := 0
	isDouble := false
	for i := len(str) - 1; i >= 0; i-- {
		digit := int(str[i] - '0')
		if digit < 0 || digit > 9 {
			return false, "invalid credit card number"
		}
		if isDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		isDouble = !isDouble
	}
	if sum%10 != 0 {
		return false, "invalid credit card number"
	}
	return true, ""
}

// validatePhoneNumber 验证电话号码
func validatePhoneNumber(value any, _ string) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "field must be string type"
	}
	// 支持以下格式：
	// +86 123 4567 8901
	// +86-123-4567-8901
	// +86.123.4567.8901
	// 86 123 4567 8901
	// 123 4567 8901
	// 12345678901
	re := regexp.MustCompile(`^(\+?\d{1,3}[-. ]?)?\d{3}[-. ]?\d{4}[-. ]?\d{4}$`)
	if !re.MatchString(str) {
		return false, "invalid phone number format"
	}
	return true, ""
}
