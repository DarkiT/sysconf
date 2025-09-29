package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// Validator 统一的验证器接口
type Validator interface {
	// Validate 验证配置数据
	Validate(config map[string]any) error
	// GetName 获取验证器名称
	GetName() string
}

// ValidatorFunc 函数式验证器
type ValidatorFunc func(config map[string]any) error

// Validate 实现 Validator 接口
func (f ValidatorFunc) Validate(config map[string]any) error {
	return f(config)
}

// GetName 实现 Validator 接口
func (f ValidatorFunc) GetName() string {
	return "函数式验证器"
}

// DefaultValidator 默认验证器，包含基础的配置验证规则
type DefaultValidator struct {
	name string
}

// NewDefaultValidator 创建默认验证器
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		name: "默认系统验证器",
	}
}

// Validate 执行默认验证规则
func (d *DefaultValidator) Validate(config map[string]any) error {
	return d.validateRecursive("", config)
}

// GetName 获取验证器名称
func (d *DefaultValidator) GetName() string {
	return d.name
}

// validateRecursive 递归验证配置
func (d *DefaultValidator) validateRecursive(prefix string, config map[string]any) error {
	for key, value := range config {
		// 构建完整的键路径
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		// 如果值是嵌套的map，递归检查
		if nestedMap, ok := value.(map[string]any); ok {
			if err := d.validateRecursive(fullKey, nestedMap); err != nil {
				return err
			}
			continue
		}

		// 执行各种验证规则
		if err := d.validatePort(fullKey, value); err != nil {
			return err
		}
		if err := d.validateTimeout(fullKey, value); err != nil {
			return err
		}
		if err := d.validateURL(fullKey, value); err != nil {
			return err
		}
		if err := d.validateEmail(fullKey, value); err != nil {
			return err
		}
		if err := d.validateHost(fullKey, value); err != nil {
			return err
		}
	}

	return nil
}

// validatePort 验证端口号
func (d *DefaultValidator) validatePort(key string, value any) error {
	if strings.HasSuffix(key, ".port") || strings.HasSuffix(key, "_port") || key == "port" {
		if port, err := cast.ToIntE(value); err == nil {
			if port < 1 || port > 65535 {
				return fmt.Errorf("port '%s' value %d is invalid (must be in range 1-65535)", key, port)
			}
		} else {
			return fmt.Errorf("port '%s' value must be an integer", key)
		}
	}
	return nil
}

// validateTimeout 验证超时设置
func (d *DefaultValidator) validateTimeout(key string, value any) error {
	if strings.Contains(key, "timeout") {
		if timeoutStr, ok := value.(string); ok {
			if _, err := time.ParseDuration(timeoutStr); err != nil {
				return fmt.Errorf("timeout setting '%s' value '%s' is invalid format (should be '30s', '5m' etc)", key, timeoutStr)
			}
		}
	}
	return nil
}

// validateURL 验证URL格式
func (d *DefaultValidator) validateURL(key string, value any) error {
	if strings.Contains(key, "url") || strings.Contains(key, "endpoint") {
		if urlStr, ok := value.(string); ok && urlStr != "" {
			if !strings.HasPrefix(urlStr, "http://") &&
				!strings.HasPrefix(urlStr, "https://") &&
				!strings.HasPrefix(urlStr, "tcp://") &&
				!strings.HasPrefix(urlStr, "ws://") &&
				!strings.HasPrefix(urlStr, "wss://") {
				return fmt.Errorf("URL '%s' value '%s' is invalid format (must start with http://, https://, tcp://, ws://, wss://)", key, urlStr)
			}
		}
	}
	return nil
}

// validateEmail 验证邮箱格式
func (d *DefaultValidator) validateEmail(key string, value any) error {
	if strings.Contains(key, "email") || strings.Contains(key, "mail") {
		if emailStr, ok := value.(string); ok && emailStr != "" {
			if !strings.Contains(emailStr, "@") || !strings.Contains(emailStr, ".") {
				return fmt.Errorf("email '%s' value '%s' is invalid format", key, emailStr)
			}
		}
	}
	return nil
}

// validateHost 验证主机名格式
func (d *DefaultValidator) validateHost(key string, value any) error {
	if strings.HasSuffix(key, ".host") || key == "host" {
		if hostStr, ok := value.(string); ok && hostStr != "" {
			// 简单的主机名验证：不能包含特殊字符
			if strings.ContainsAny(hostStr, " \t\n\r\"'<>&") {
				return fmt.Errorf("hostname '%s' value '%s' is invalid format", key, hostStr)
			}
		}
	}
	return nil
}

// StructuredValidator 基于规则的验证器（重命名避免冲突）
type StructuredValidator struct {
	name     string
	rules    map[string][]ValidationRule // 结构化规则
	strRules map[string][]string         // 字符串规则
}

// NewRuleValidator 创建基于规则的验证器（保持接口兼容性）
func NewRuleValidator(name string) *StructuredValidator {
	return &StructuredValidator{
		name:     name,
		rules:    make(map[string][]ValidationRule),
		strRules: make(map[string][]string),
	}
}

// Validate 执行规则验证
func (r *StructuredValidator) Validate(config map[string]any) error {
	// 验证结构化规则
	for key, rules := range r.rules {
		value, exists := getNestedValue(config, key)

		for _, rule := range rules {
			if !exists && rule.Type != "required" {
				continue // 非必填字段且不存在，跳过验证
			}

			// 使用全局 Validate 函数验证规则
			if err := Validate(value, rule); err != nil {
				return fmt.Errorf("validator '%s' - field '%s': %w", r.name, key, err)
			}
		}
	}

	// 验证字符串规则
	for key, rules := range r.strRules {
		value, exists := getNestedValue(config, key)

		for _, ruleStr := range rules {
			if !exists && !strings.HasPrefix(ruleStr, "required") {
				continue
			}

			// 使用 rules.go 中的 ValidateValue 验证字符串规则
			if valid, errMsg := ValidateValue(value, ruleStr); !valid {
				return fmt.Errorf("validator '%s' - field '%s': %s", r.name, key, errMsg)
			}
		}
	}

	return nil
}

// GetName 获取验证器名称
func (r *StructuredValidator) GetName() string {
	return r.name
}

// AddRule 添加单个结构化规则
func (r *StructuredValidator) AddRule(key string, rule ValidationRule) *StructuredValidator {
	r.rules[key] = append(r.rules[key], rule)
	return r
}

// AddRules 添加多个结构化规则
func (r *StructuredValidator) AddRules(key string, rules ...ValidationRule) *StructuredValidator {
	r.rules[key] = append(r.rules[key], rules...)
	return r
}

// AddStringRule 添加单个字符串规则
func (r *StructuredValidator) AddStringRule(key string, rule string) *StructuredValidator {
	r.strRules[key] = append(r.strRules[key], rule)
	return r
}

// AddStringRules 添加多个字符串规则
func (r *StructuredValidator) AddStringRules(key string, rules ...string) *StructuredValidator {
	r.strRules[key] = append(r.strRules[key], rules...)
	return r
}

// GetRulesForField 获取特定字段的结构化规则
func (r *StructuredValidator) GetRulesForField(key string) []ValidationRule {
	if rules, exists := r.rules[key]; exists {
		return rules
	}
	return nil
}

// GetStringRulesForField 获取特定字段的字符串规则
func (r *StructuredValidator) GetStringRulesForField(key string) []string {
	if rules, exists := r.strRules[key]; exists {
		return rules
	}
	return nil
}

// HasRuleForField 检查是否有特定字段的规则
func (r *StructuredValidator) HasRuleForField(key string) bool {
	_, hasStructRule := r.rules[key]
	_, hasStringRule := r.strRules[key]
	return hasStructRule || hasStringRule
}

// GetSupportedFields 获取验证器支持的所有字段前缀
func (r *StructuredValidator) GetSupportedFields() []string {
	fieldPrefixes := make(map[string]bool)

	// 从结构化规则中提取字段前缀
	for key := range r.rules {
		if prefix := extractFieldPrefix(key); prefix != "" {
			fieldPrefixes[prefix] = true
		}
	}

	// 从字符串规则中提取字段前缀
	for key := range r.strRules {
		if prefix := extractFieldPrefix(key); prefix != "" {
			fieldPrefixes[prefix] = true
		}
	}

	// 转换为切片
	prefixes := make([]string, 0, len(fieldPrefixes))
	for prefix := range fieldPrefixes {
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

// extractFieldPrefix 从配置键中提取字段前缀
func extractFieldPrefix(key string) string {
	parts := strings.Split(key, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// CompositeValidator 复合验证器，组合多个验证器
type CompositeValidator struct {
	name       string
	validators []Validator
}

// NewCompositeValidator 创建复合验证器
func NewCompositeValidator(name string, validators ...Validator) *CompositeValidator {
	return &CompositeValidator{
		name:       name,
		validators: validators,
	}
}

// Validate 执行所有子验证器的验证
func (c *CompositeValidator) Validate(config map[string]any) error {
	for i, validator := range c.validators {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("composite validator '%s' validator %d (%s) validation failed: %w",
				c.name, i+1, validator.GetName(), err)
		}
	}
	return nil
}

// GetName 获取验证器名称
func (c *CompositeValidator) GetName() string {
	return c.name
}

// AddValidator 添加验证器
func (c *CompositeValidator) AddValidator(validator Validator) *CompositeValidator {
	c.validators = append(c.validators, validator)
	return c
}

// GetValidators 获取所有子验证器
func (c *CompositeValidator) GetValidators() []Validator {
	return c.validators
}

// getNestedValue 获取嵌套配置值
func getNestedValue(config map[string]any, key string) (any, bool) {
	keys := strings.Split(key, ".")
	current := config

	for i, k := range keys {
		if i == len(keys)-1 {
			// 最后一个键，返回值
			value, exists := current[k]
			return value, exists
		}

		// 中间键，继续向下查找
		if next, ok := current[k].(map[string]any); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
}
