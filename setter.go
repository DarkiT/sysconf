package sysconf

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/darkit/sysconf/validation"
)

// Set 设置配置值
func (c *Config) Set(key string, value any) error {
	start := time.Now()
	defer func() {
		recordSetOperation(time.Since(start))
	}()

	if key == "" {
		c.logger.Errorf("Attempted to set config with empty key")
		recordErrorOperation()
		return ErrInvalidKey
	}

	// 使用新的原子设置方法（先设置以便验证器能看到完整配置）
	if err := c.setRaw(key, value); err != nil {
		c.logger.Errorf("Failed to set config value: %v", err)
		recordErrorOperation()
		return err
	}

	// 同步设置到viper（用于文件写入）
	c.mu.Lock()
	c.viper.Set(key, value)
	c.mu.Unlock()

	// 立即进行字段级验证（只验证当前设置的字段）
	if err := c.validateSingleField(key, value); err != nil {
		// 验证失败，回滚设置
		c.logger.Errorf("Validation failed for key %s: %v", key, err)
		// 回滚原子存储
		currentData := c.loadData()
		newData := make(map[string]any)
		for k, v := range currentData {
			if k != key {
				newData[k] = v
			}
		}
		c.storeData(newData)
		// 回滚viper
		c.mu.Lock()
		c.viper.Set(key, nil)
		c.mu.Unlock()
		recordErrorOperation()
		return err
	}

	// 使缓存失效，以便下次读取时获取最新值（保持兼容性）
	c.invalidateCache()

	// 如果配置文件名称不存在则不保存文件
	if c.name == "" {
		c.logger.Debugf("Config file name not set, skipping write")
		return nil
	}
	// 用独立的互斥锁处理写入操作
	c.writeMu.Lock()

	// 标记有待写入的更改
	c.pendingWrites = true

	// 如果定时器已存在，重置它
	if c.writeTimer != nil {
		if c.writeTimer.Stop() {
			c.logger.Debugf("Reset existing config write timer")
		}
		c.writeTimer = nil
	}

	delay := c.writeDelay
	if delay > 0 {
		c.writeTimer = time.AfterFunc(delay, c.flushPendingWritesAsync)
		c.writeMu.Unlock()
	} else {
		c.writeMu.Unlock()
		go c.flushPendingWritesAsync()
	}

	return nil
}

// flushPendingWritesAsync 负责异步刷新待写入的配置。
func (c *Config) flushPendingWritesAsync() {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.flushPendingWritesLocked()
}

// flushPendingWritesLocked 假设调用前已经持有 writeMu。
func (c *Config) flushPendingWritesLocked() {
	if !c.pendingWrites {
		c.logger.Debugf("No pending changes, skipping write operation")
		return
	}

	// 在写入前再次验证所有配置的有效性
	if err := c.validateAllConfigs(); err != nil {
		c.logger.Errorf("Configuration validation failed before write: %v", err)
		c.pendingWrites = false
		return
	}

	c.logger.Infof("Writing config file")

	// 如果启用了加密，使用自定义的写入方法
	if c.cryptoOptions.Enabled {
		if err := c.writeConfigFile(); err != nil {
			c.logger.Errorf("Failed to write encrypted config file: %v", err)
		} else {
			c.logger.Infof("Encrypted config file written successfully")
		}
	} else {
		// 没有启用加密时，使用viper的标准写入方法
		if err := c.viper.WriteConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if errors.As(err, &configFileNotFoundError) {
				configFile := filepath.Join(c.path, c.name+"."+c.mode)
				c.logger.Infof("Config file does not exist, creating new file: %s", configFile)
				if err := c.viper.WriteConfigAs(configFile); err != nil {
					c.logger.Errorf("Failed to create config file: %v", err)
				} else {
					c.logger.Infof("Config file created successfully")
				}
			} else {
				c.logger.Errorf("Failed to write config file: %v", err)
			}
		} else {
			c.logger.Infof("Config file written successfully")
		}
	}

	c.pendingWrites = false
}

// validateSingleField 验证单个配置字段的有效性（字段级验证）
func (c *Config) validateSingleField(key string, value any) error {
	// 如果没有验证器，跳过验证
	c.mu.RLock()
	validators := make([]ConfigValidator, len(c.validators))
	copy(validators, c.validators)
	c.mu.RUnlock()

	if len(validators) == 0 {
		// 使用默认验证器进行基础类型验证
		defaultValidator := validation.NewDefaultValidator()
		testConfig := map[string]any{key: value}
		return defaultValidator.Validate(testConfig)
	}

	// 为字段级验证创建最小配置上下文
	// 包含当前字段和必要的上下文信息
	fieldConfig := make(map[string]any)
	fieldConfig[key] = value

	// 获取当前配置数据，为验证器提供上下文
	currentData := c.loadData()

	// 构建验证上下文：包含当前设置的字段和已有的相关字段
	validationContext := make(map[string]any)
	validationContext[key] = value

	// 添加同一配置组的其他字段作为上下文
	keyParts := strings.Split(key, ".")
	if len(keyParts) > 1 {
		prefix := keyParts[0] + "."
		contextGroup := make(map[string]any)
		contextGroup[keyParts[len(keyParts)-1]] = value

		// 添加同组其他已存在的字段
		for k, v := range currentData {
			if strings.HasPrefix(k, prefix) && k != key {
				subKey := strings.TrimPrefix(k, prefix)
				if !strings.Contains(subKey, ".") { // 只添加同级字段
					contextGroup[subKey] = v
				}
			}
		}

		validationContext[keyParts[0]] = contextGroup
	}

	// 对每个验证器执行字段级验证
	for _, validator := range validators {
		// 检查验证器是否支持该字段
		if !c.validatorSupportsField(validator, key) {
			continue // 跳过不相关的验证器
		}

		// 对于StructuredValidator，执行单字段验证
		if structValidator, ok := validator.(*validation.StructuredValidator); ok {
			if err := c.validateSingleFieldWithStructValidator(structValidator, key, value); err != nil {
				c.logger.Errorf("Field validation failed for key %s with validator %s: %v", key, validator.GetName(), err)
				return fmt.Errorf("field validation failed (%s): %w", validator.GetName(), err)
			}
		} else {
			// 对于其他验证器类型，使用原有逻辑
			if err := validator.Validate(validationContext); err != nil {
				// 字段级验证失败，但错误信息应当更友好
				c.logger.Errorf("Field validation failed for key %s with validator %s: %v", key, validator.GetName(), err)
				return fmt.Errorf("field validation failed (%s): %w", validator.GetName(), err)
			}
		}
	}

	c.logger.Debugf("Field validation passed for key %s (%d validators checked)", key, len(validators))
	return nil
}

// validatorSupportsField 检查验证器是否支持特定字段
func (c *Config) validatorSupportsField(validator ConfigValidator, key string) bool {
	keyParts := strings.Split(key, ".")
	if len(keyParts) == 0 {
		return false
	}

	fieldGroup := keyParts[0]

	// 优先使用 StructuredValidator 的动态字段检查（避免硬编码）
	if structValidator, ok := validator.(*validation.StructuredValidator); ok {
		supportedFields := structValidator.GetSupportedFields()
		for _, supportedField := range supportedFields {
			if supportedField == fieldGroup {
				return true
			}
		}
		return false
	}

	// 对于其他验证器类型，检查是否有 HasRuleForField 方法
	if ruleValidator, ok := validator.(interface{ HasRuleForField(string) bool }); ok {
		return ruleValidator.HasRuleForField(key)
	}

	// 默认验证器支持所有字段
	if strings.Contains(strings.ToLower(validator.GetName()), "default") {
		return true
	}

	// 保守策略：未知验证器默认不支持
	c.logger.Debugf("Unknown validator type %s, skipping field %s", validator.GetName(), key)
	return false
}

// validateSingleFieldWithStructValidator 使用StructuredValidator验证单个字段
func (c *Config) validateSingleFieldWithStructValidator(validator *validation.StructuredValidator, key string, value any) error {
	// 直接验证单个字段，跳过required检查其他字段的逻辑
	rules := validator.GetRulesForField(key)
	stringRules := validator.GetStringRulesForField(key)

	// 验证结构化规则
	for _, rule := range rules {
		// 对于单字段验证，跳过required检查（因为我们正在设置这个字段）
		if rule.Type == "required" {
			continue
		}

		if err := validation.Validate(value, rule); err != nil {
			return fmt.Errorf("field '%s': %w", key, err)
		}
	}

	// 验证字符串规则
	for _, ruleStr := range stringRules {
		// 对于单字段验证，跳过required检查
		if strings.HasPrefix(ruleStr, "required") {
			continue
		}

		if valid, errMsg := validation.ValidateValue(value, ruleStr); !valid {
			return fmt.Errorf("field '%s': %s", key, errMsg)
		}
	}

	return nil
}

// validateAllConfigs 验证所有配置的有效性
func (c *Config) validateAllConfigs() error {
	// 获取viper的嵌套结构数据供验证器使用
	c.mu.RLock()
	allSettings := c.viper.AllSettings()
	c.mu.RUnlock()

	// 如果viper数据为空，尝试从原子存储重构嵌套结构
	if len(allSettings) == 0 {
		flatData := c.loadData()
		allSettings = c.reconstructNestedStructure(flatData)
	}

	// 执行所有已注册的验证器
	for i, validator := range c.validators {
		if err := validator.Validate(allSettings); err != nil {
			c.logger.Errorf("Config validation failed at validator %d: %v", i+1, err)
			return fmt.Errorf("config validation failed (validator %d): %w", i+1, err)
		}
	}

	// 如果没有自定义验证器，执行默认的内置验证
	if len(c.validators) == 0 {
		defaultValidator := validation.NewDefaultValidator()
		if err := defaultValidator.Validate(allSettings); err != nil {
			c.logger.Errorf("Default config validation failed: %v", err)
			return fmt.Errorf("default validation failed: %w", err)
		}
	}

	c.logger.Debugf("All config validations passed (%d validators)", len(c.validators))
	return nil
}

// SetEnvPrefix 更新环境变量选项
func (c *Config) SetEnvPrefix(prefix string) error {
	c.mu.Lock()
	c.envOptions.Prefix = prefix
	c.envOptions.Enabled = prefix != "" // 如果有前缀就启用环境变量
	c.mu.Unlock()

	// 重新初始化（不在锁内调用以避免死锁）
	return c.reinitialize()
}
