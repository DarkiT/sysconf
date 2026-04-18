package sysconf

import (
	"fmt"
	"strings"
	"time"

	"github.com/darkit/sysconf/validation"
)

// Set 设置配置值
func (c *Config) Set(key string, value any) error {
	if c.closed.Load() {
		return ErrAlreadyClosed
	}

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
	if c.closed.Load() {
		c.mu.Unlock()
		return ErrAlreadyClosed
	}

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

	// 根据写入延迟策略触发写盘
	if err := c.scheduleWrite(); err != nil {
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

	if markPending {
		c.pendingWrites = true
	}

	if !c.pendingWrites {
		c.logger.Debugf("No pending changes, skipping write operation")
		c.writeMu.Unlock()
		c.mu.Unlock()
		c.cacheBuildMu.Unlock()
		return nil
	}

	c.logger.Infof("Writing config file")

	// 在持锁后获取配置快照，确保一致性
	settingsSnapshot := c.snapshotAllSettings()
	// 标记已消费当前待写入状态，允许新的写入在锁外排队
	c.pendingWrites = false

	// 释放读写锁，避免写盘阻塞写路径
	c.mu.Unlock()
	c.cacheBuildMu.Unlock()

	if err := c.writeConfigFileWithData(settingsSnapshot); err != nil {
		c.logger.Errorf("Failed to write config file: %v", err)
		c.writeMu.Unlock()
		return err
	}
	// 写入完成后再释放写入锁，保证写入顺序
	c.writeMu.Unlock()
	c.logger.Infof("Config file written successfully")
	return nil
}

// scheduleWrite 根据 writeDelay 决定立即写盘或延迟合并写盘。
func (c *Config) scheduleWrite() error {
	return c.scheduleDebouncedWrite()
}

func (c *Config) scheduleDebouncedWrite() error {
	if c.writeDelay <= 0 {
		return c.flushPendingWritesWithPending(true)
	}

	// 标记待写入并重置定时器
	c.cacheBuildMu.Lock()
	c.mu.Lock()
	c.pendingWrites = true
	if c.writeTimer == nil {
		c.writeTimer = time.AfterFunc(c.writeDelay, func() {
			if err := c.flushPendingWritesWithPending(false); err != nil {
				c.logger.Errorf("Failed to flush pending writes: %v", err)
			}
		})
	} else {
		c.writeTimer.Stop()
		c.writeTimer.Reset(c.writeDelay)
	}
	c.mu.Unlock()
	c.cacheBuildMu.Unlock()
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

// SetMultiple 批量设置多个配置值
// 相比多次调用 Set，此方法减少了验证和文件写入开销
//
// 参数:
//   - values: 键值对映射，键为配置路径，值为配置值
//
// 返回值:
//   - error: 如果任何键值对验证失败，返回错误并回滚所有更改
func (c *Config) SetMultiple(values map[string]any) error {
	if c.closed.Load() {
		return ErrAlreadyClosed
	}

	if len(values) == 0 {
		return nil
	}

	// 创建快照用于回滚
	snap := c.createSnapshot()

	start := time.Now()
	defer func() {
		recordSetOperation(time.Since(start))
	}()

	// 验证所有键
	for key := range values {
		if key == "" {
			c.logger.Errorf("Attempted to set config with empty key in batch operation")
			recordErrorOperation()
			return ErrInvalidKey
		}
	}

	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return ErrAlreadyClosed
	}

	// 复制当前数据
	currentData := c.loadData()
	newData := make(map[string]any, len(currentData)+len(values))

	// 收集所有需要移除的键前缀
	prefixes := make([]string, 0, len(values))
	for key := range values {
		prefixes = append(prefixes, key+".")
	}

	// 复制不受影响的数据
	for k, v := range currentData {
		shouldSkip := false
		for key := range values {
			if k == key {
				shouldSkip = true
				break
			}
		}
		if !shouldSkip {
			for _, prefix := range prefixes {
				if strings.HasPrefix(k, prefix) {
					shouldSkip = true
					break
				}
			}
		}
		if !shouldSkip {
			newData[k] = v
		}
	}

	// 合并所有新值
	for key, value := range values {
		c.mergeValueIntoData(newData, key, value)
	}

	// 拷贝验证器切片
	validators := make([]ConfigValidator, len(c.validators))
	copy(validators, c.validators)

	// 验证所有字段
	for key, value := range values {
		if err := c.validateSingleFieldWithData(key, value, validators, newData); err != nil {
			c.logger.Errorf("Validation failed for key %s in batch operation: %v", key, err)
			recordErrorOperation()
			c.mu.Unlock()
			c.restoreSnapshot(snap)
			return fmt.Errorf("batch set failed at key '%s': %w", key, err)
		}
	}

	// 验证通过后原子提交
	c.storeData(newData)
	for key, value := range values {
		c.viper.Set(key, value)
	}
	c.mu.Unlock()

	c.invalidateCache()

	// 如果配置文件名称不存在则不保存文件
	if c.name == "" {
		c.logger.Debugf("Config file name not set, skipping write")
		return nil
	}

	// 根据写入延迟策略触发写盘
	if err := c.scheduleWrite(); err != nil {
		c.restoreSnapshot(snap)
		return fmt.Errorf("batch write failed and rolled back: %w", err)
	}

	c.logger.Infof("Batch set completed: %d keys updated", len(values))
	return nil
}

// SetEnvPrefix 更新环境变量选项
func (c *Config) SetEnvPrefix(prefix string) error {
	if c.closed.Load() {
		return ErrAlreadyClosed
	}

	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return ErrAlreadyClosed
	}
	c.envOptions.Prefix = prefix
	c.envOptions.Enabled = prefix != "" // 如果有前缀就启用环境变量
	c.mu.Unlock()

	// 重新初始化（不在锁内调用以避免死锁）
	return c.reinitialize()
}
