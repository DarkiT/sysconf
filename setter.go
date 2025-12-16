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
	// 创建快照用于回滚
	snap := c.createSnapshot()

	start := time.Now()
	defer func() {
		recordSetOperation(time.Since(start))
	}()

	if key == "" {
		c.logger.Errorf("Attempted to set config with empty key")
		recordErrorOperation()
		return ErrInvalidKey
	}

	// 统一持锁，避免并发写导致的状态丢失
	c.mu.Lock()

	// 复制当前数据，准备生成候选快照
	currentData := c.loadData()
	newData := make(map[string]any, len(currentData)+1)

	// 移除当前键以及同前缀的旧值，确保写入后数据一致
	prefix := key + "."
	for k, v := range currentData {
		if k == key || strings.HasPrefix(k, prefix) {
			continue
		}
		newData[k] = v
	}

	// 合并新值（自动展开嵌套结构）
	c.mergeValueIntoData(newData, key, value)

	// 拷贝验证器切片，避免锁内重复加锁
	validators := make([]ConfigValidator, len(c.validators))
	copy(validators, c.validators)

	// 字段级验证基于候选快照执行，避免无效写入后再回滚
	if err := c.validateSingleFieldWithData(key, value, validators, newData); err != nil {
		c.logger.Errorf("Validation failed for key %s: %v", key, err)
		recordErrorOperation()
		c.mu.Unlock()
		c.restoreSnapshot(snap)
		return err
	}

	// 验证通过后再原子提交数据与 viper
	c.storeData(newData)
	c.viper.Set(key, value)
	c.mu.Unlock()

	c.invalidateCache()

	// 如果配置文件名称不存在则不保存文件
	if c.name == "" {
		c.logger.Debugf("Config file name not set, skipping write")
		return nil
	}

	// 标记待写入并按统一锁序写回配置文件
	if err := c.flushPendingWritesWithPending(true); err != nil {
		c.restoreSnapshot(snap)
		return fmt.Errorf("write failed and rolled back: %w", err)
	}

	return nil
}

// flushPendingWritesWithPending 以统一锁顺序（cacheBuildMu -> mu.RLock -> writeMu）刷新待写入配置。
// markPending 表示在写入锁内应当标记有待写入（用于 Set 调用路径）。
func (c *Config) flushPendingWritesWithPending(markPending bool) error {
	// 按顺序获取锁，防止与 snapshotAllSettings 产生循环等待
	c.cacheBuildMu.Lock()
	c.mu.Lock()
	c.writeMu.Lock()
	defer func() {
		c.writeMu.Unlock()
		c.mu.Unlock()
		c.cacheBuildMu.Unlock()
	}()

	if markPending {
		c.pendingWrites = true
	}

	if !c.pendingWrites {
		c.logger.Debugf("No pending changes, skipping write operation")
		return nil
	}

	c.logger.Infof("Writing config file")

	// 在持锁后获取加密路径所需的配置快照，保证一致性并避免重入
	var settingsSnapshot map[string]any
	if c.cryptoOptions.Enabled {
		settingsSnapshot = deepCloneMap(c.viper.AllSettings())
	}

	// 如果启用了加密，使用自定义的写入方法（使用预先获取的快照）
	if c.cryptoOptions.Enabled {
		if err := c.writeConfigFileWithData(settingsSnapshot); err != nil {
			c.logger.Errorf("Failed to write encrypted config file: %v", err)
			c.pendingWrites = false
			return err
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
					c.pendingWrites = false
					return err
				} else {
					c.logger.Infof("Config file created successfully")
				}
			} else {
				c.logger.Errorf("Failed to write config file: %v", err)
				c.pendingWrites = false
				return err
			}
		} else {
			c.logger.Infof("Config file written successfully")
		}
	}

	c.pendingWrites = false
	return nil
}

// validateSingleFieldWithData 基于候选数据验证单个字段，避免先写后回滚
func (c *Config) validateSingleFieldWithData(
	key string,
	value any,
	validators []ConfigValidator,
	currentData map[string]any,
) error {
	// 没有验证器时使用默认验证器做基础类型校验
	if len(validators) == 0 {
		defaultValidator := validation.NewDefaultValidator()
		testConfig := map[string]any{key: value}
		return defaultValidator.Validate(testConfig)
	}

	// 构建验证上下文：当前字段 + 同级字段
	validationContext := make(map[string]any)
	validationContext[key] = value

	keyParts := strings.Split(key, ".")
	if len(keyParts) > 1 {
		prefix := keyParts[0] + "."
		contextGroup := make(map[string]any)
		contextGroup[keyParts[len(keyParts)-1]] = value

		for k, v := range currentData {
			if strings.HasPrefix(k, prefix) && k != key {
				subKey := strings.TrimPrefix(k, prefix)
				if !strings.Contains(subKey, ".") {
					contextGroup[subKey] = v
				}
			}
		}

		validationContext[keyParts[0]] = contextGroup
	}

	// 执行验证
	for _, validator := range validators {
		if !c.validatorSupportsField(validator, key) {
			continue
		}

		if structValidator, ok := validator.(*validation.StructuredValidator); ok {
			if err := c.validateSingleFieldWithStructValidator(structValidator, key, value); err != nil {
				c.logger.Errorf("Field validation failed for key %s with validator %s: %v", key, validator.GetName(), err)
				return fmt.Errorf("field validation failed (%s): %w", validator.GetName(), err)
			}
			continue
		}

		if err := validator.Validate(validationContext); err != nil {
			c.logger.Errorf("Field validation failed for key %s with validator %s: %v", key, validator.GetName(), err)
			return fmt.Errorf("field validation failed (%s): %w", validator.GetName(), err)
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

// validateAllConfigsWithData 使用传入数据执行全量验证，避免重复持锁。
// SetEnvPrefix 更新环境变量选项
func (c *Config) SetEnvPrefix(prefix string) error {
	c.mu.Lock()
	c.envOptions.Prefix = prefix
	c.envOptions.Enabled = prefix != "" // 如果有前缀就启用环境变量
	c.mu.Unlock()

	// 重新初始化（不在锁内调用以避免死锁）
	return c.reinitialize()
}
