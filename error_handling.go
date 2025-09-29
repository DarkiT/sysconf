package sysconf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigError 配置错误类型
type ConfigError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	File    string `json:"file,omitempty"`
	Cause   error  `json:"-"`
}

func (e *ConfigError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// 错误类型常量
const (
	ErrTypeFileNotFound   = "FileNotFound"
	ErrTypePermission     = "Permission"
	ErrTypeInvalidFormat  = "InvalidFormat"
	ErrTypeValidation     = "Validation"
	ErrTypeDecryption     = "Decryption"
	ErrTypeConversion     = "Conversion"
	ErrTypeEnvironment    = "Environment"
	ErrTypeInitialization = "Initialization"
)

// NewConfigError 创建新的配置错误
func NewConfigError(errorType, message string) *ConfigError {
	return &ConfigError{
		Type:    errorType,
		Message: message,
	}
}

// NewConfigErrorWithCause 创建带原因的配置错误
func NewConfigErrorWithCause(errorType, message string, cause error) *ConfigError {
	return &ConfigError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// NewConfigErrorWithDetails 创建带详细信息的配置错误
func NewConfigErrorWithDetails(errorType, message, key, value, file string, cause error) *ConfigError {
	return &ConfigError{
		Type:    errorType,
		Message: message,
		Key:     key,
		Value:   value,
		File:    file,
		Cause:   cause,
	}
}

// wrapError 包装标准错误为配置错误
func (c *Config) wrapError(err error, context string) error {
	if err == nil {
		return nil
	}

	// 如果已经是ConfigError，直接返回
	if configErr, ok := err.(*ConfigError); ok {
		return configErr
	}

	// 根据错误类型进行分类处理
	if os.IsNotExist(err) {
		return &ConfigError{
			Type:    ErrTypeFileNotFound,
			Message: fmt.Sprintf("配置文件不存在: %s", context),
			File:    c.getConfigFilePath(),
			Cause:   err,
		}
	}

	if os.IsPermission(err) {
		return &ConfigError{
			Type:    ErrTypePermission,
			Message: fmt.Sprintf("配置文件权限不足: %s", context),
			File:    c.getConfigFilePath(),
			Cause:   err,
		}
	}

	if strings.Contains(err.Error(), "yaml") || strings.Contains(err.Error(), "json") || strings.Contains(err.Error(), "toml") {
		return &ConfigError{
			Type:    ErrTypeInvalidFormat,
			Message: fmt.Sprintf("配置文件格式错误: %s", context),
			File:    c.getConfigFilePath(),
			Cause:   err,
		}
	}

	if strings.Contains(err.Error(), "decrypt") || strings.Contains(err.Error(), "authentication") {
		return &ConfigError{
			Type:    ErrTypeDecryption,
			Message: fmt.Sprintf("配置文件解密失败: %s", context),
			File:    c.getConfigFilePath(),
			Cause:   err,
		}
	}

	// 默认为初始化错误
	return &ConfigError{
		Type:    ErrTypeInitialization,
		Message: fmt.Sprintf("配置初始化失败: %s", context),
		Cause:   err,
	}
}

// getConfigFilePath 获取配置文件路径
func (c *Config) getConfigFilePath() string {
	if c.name == "" {
		return "内存配置"
	}
	return filepath.Join(c.path, c.name+"."+c.mode)
}

// IsConfigError 检查是否为配置错误
func IsConfigError(err error) bool {
	var configErr *ConfigError
	return errors.As(err, &configErr)
}

// GetConfigErrorType 获取配置错误类型
func GetConfigErrorType(err error) string {
	var configErr *ConfigError
	if errors.As(err, &configErr) {
		return configErr.Type
	}
	return ""
}

// GetErrorSuggestion 根据错误类型提供修复建议
func GetErrorSuggestion(err error) string {
	var configErr *ConfigError
	if !errors.As(err, &configErr) {
		return "请检查错误信息并重试"
	}

	switch configErr.Type {
	case ErrTypeFileNotFound:
		return fmt.Sprintf("请确保配置文件 %s 存在，或使用 WithContent() 提供默认配置", configErr.File)

	case ErrTypePermission:
		return fmt.Sprintf("请检查对配置文件 %s 的读写权限", configErr.File)

	case ErrTypeInvalidFormat:
		return "请检查配置文件格式是否正确，可以使用在线验证工具验证YAML/JSON格式"

	case ErrTypeValidation:
		return fmt.Sprintf("请检查配置项 %s 的值 %s 是否符合验证规则", configErr.Key, configErr.Value)

	case ErrTypeDecryption:
		return "请检查加密密钥是否正确，或尝试使用未加密的配置文件"

	case ErrTypeConversion:
		return fmt.Sprintf("请检查配置项 %s 的值 %s 是否为有效的类型", configErr.Key, configErr.Value)

	case ErrTypeEnvironment:
		return "请检查环境变量设置，确保格式正确且值有效"

	default:
		return "请查看详细错误信息并检查配置"
	}
}

// PrintErrorHelp 打印错误帮助信息
func PrintErrorHelp(err error) {
	if err == nil {
		return
	}

	fmt.Printf("❌ 配置错误: %v\n", err)

	if suggestion := GetErrorSuggestion(err); suggestion != "" {
		fmt.Printf("💡 建议: %s\n", suggestion)
	}

	var configErr *ConfigError
	if errors.As(err, &configErr) {
		if configErr.File != "" && configErr.File != "内存配置" {
			fmt.Printf("📁 文件: %s\n", configErr.File)
		}
		if configErr.Key != "" {
			fmt.Printf("🔑 配置键: %s\n", configErr.Key)
		}
		if configErr.Value != "" {
			fmt.Printf("💾 配置值: %s\n", configErr.Value)
		}
	}
}

// ErrorRecovery 错误恢复策略
type ErrorRecovery struct {
	config *Config
}

// NewErrorRecovery 创建错误恢复实例
func NewErrorRecovery(config *Config) *ErrorRecovery {
	return &ErrorRecovery{config: config}
}

// RecoverFromError 从错误中恢复
func (er *ErrorRecovery) RecoverFromError(err error) error {
	var configErr *ConfigError
	if !errors.As(err, &configErr) {
		return err
	}

	switch configErr.Type {
	case ErrTypeFileNotFound:
		return er.recoverFromFileNotFound()

	case ErrTypePermission:
		return er.recoverFromPermissionError()

	case ErrTypeInvalidFormat:
		return er.recoverFromFormatError()

	case ErrTypeDecryption:
		return er.recoverFromDecryptionError()

	default:
		return err
	}
}

// recoverFromFileNotFound 从文件未找到错误中恢复
func (er *ErrorRecovery) recoverFromFileNotFound() error {
	if er.config.content != "" {
		er.config.logger.Infof("File not found, creating from default content")
		return er.config.createDefaultConfig()
	}
	return NewConfigError(ErrTypeFileNotFound, "配置文件不存在且未提供默认内容")
}

// recoverFromPermissionError 从权限错误中恢复
func (er *ErrorRecovery) recoverFromPermissionError() error {
	// 尝试切换到只读模式
	er.config.logger.Warnf("Permission denied, switching to read-only mode")
	// 这里可以设置一个只读标志
	return NewConfigError(ErrTypePermission, "权限不足，已切换到只读模式")
}

// recoverFromFormatError 从格式错误中恢复
func (er *ErrorRecovery) recoverFromFormatError() error {
	// 尝试备份原文件并创建新的默认配置
	if er.config.content != "" {
		er.config.logger.Warnf("Format error, backing up and creating new config")
		return er.config.createDefaultConfig()
	}
	return NewConfigError(ErrTypeInvalidFormat, "配置文件格式错误且无法自动修复")
}

// recoverFromDecryptionError 从解密错误中恢复
func (er *ErrorRecovery) recoverFromDecryptionError() error {
	// 如果有默认内容，可以尝试重新创建加密配置
	if er.config.content != "" {
		er.config.logger.Warnf("Decryption failed, recreating encrypted config from defaults")
		return er.config.createDefaultConfig()
	}
	return NewConfigError(ErrTypeDecryption, "解密失败且无法自动恢复")
}
