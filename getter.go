package sysconf

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// ParamParser 参数解析器
type ParamParser struct{}

// ParseKeyAndDefault 解析配置键和默认值
// 返回: (key, defaultValue, hasDefault)
func (p *ParamParser) ParseKeyAndDefault(parts []string) (string, string, bool) {
	if len(parts) == 0 {
		return "", "", false
	}

	// 如果只有一个参数，检查是否包含点号
	if len(parts) == 1 {
		return parts[0], "", false
	}

	// 对于多个参数，区分两种情况：
	// 1. 点号格式：cfg.GetString("database.host", "default")
	// 2. 多参数格式：cfg.GetString("database", "host", "default")

	firstPart := parts[0]
	if strings.Contains(firstPart, ".") {
		// 点号格式：第一个参数是完整键，最后一个参数是默认值
		key := firstPart
		defaultVal := parts[len(parts)-1]
		return key, defaultVal, true
	} else {
		// 多参数格式：除最后一个参数外的所有参数组成键
		keyParts := parts[:len(parts)-1]
		key := strings.Join(keyParts, ".")
		defaultVal := parts[len(parts)-1]
		return key, defaultVal, true
	}
}

// Get 获取配置值
//
// 参数:
//   - key: 配置键名
//   - def: 可选的默认值，当配置不存在时返回
//
// 返回值:
//   - 配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) Get(key string, def ...any) any {
	start := time.Now()
	defer func() {
		// 记录性能指标
		duration := time.Since(start)
		cacheHit := true // 新架构中总是从原子存储获取，本质上是缓存
		recordGetOperation(duration, cacheHit)
	}()

	if key == "" {
		if len(def) > 0 {
			return def[0]
		}
		return nil
	}

	// 使用新的无锁原子读取
	if val, exists := c.getRaw(key); exists {
		c.logger.Debugf("Get config value: %s = %v", key, val)
		return val
	}

	// 不存在则返回默认值
	if len(def) > 0 {
		return def[0]
	}

	c.logger.Debugf("Config key not found: %s", key)
	return nil
}

// GetBool 获取布尔值配置
//
// 支持两种调用方式：
//   - GetBool("database.host", "true") - 点号分隔的键名
//   - GetBool("database", "host", "true") - 多参数形式
//
// 参数:
//   - parts: 可变参数，最后一个参数可选作为默认值
//
// 返回值:
//   - 布尔类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetBool(parts ...string) bool {
	if len(parts) == 0 {
		return false
	}

	parser := &ParamParser{}
	key, defaultVal, hasDefault := parser.ParseKeyAndDefault(parts)

	if val, exists := c.getRaw(key); exists {
		// 快速路径：直接类型断言
		if b, ok := val.(bool); ok {
			return b
		}
		// 回退到 cast 转换
		if result, err := cast.ToBoolE(val); err == nil {
			return result
		}
	}

	if hasDefault {
		if val, err := strconv.ParseBool(defaultVal); err == nil {
			return val
		}
		c.logger.Errorf("Invalid default bool value '%s' for key '%s', using false", defaultVal, key)
	}
	return false
}

// GetFloat 获取浮点数配置
//
// 支持两种调用方式：
//   - GetFloat("metrics.value", "0.95") - 点号分隔的键名
//   - GetFloat("metrics", "value", "0.95") - 多参数形式
//
// 参数:
//   - parts: 可变参数，最后一个参数可选作为默认值
//
// 返回值:
//   - 浮点类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetFloat(parts ...string) float64 {
	if len(parts) == 0 {
		return 0
	}

	parser := &ParamParser{}
	key, defaultVal, hasDefault := parser.ParseKeyAndDefault(parts)

	if val, exists := c.getRaw(key); exists {
		// 快速路径：直接类型断言
		if f, ok := val.(float64); ok {
			return f
		}
		if f, ok := val.(float32); ok {
			return float64(f)
		}
		if i, ok := val.(int); ok {
			return float64(i)
		}
		// 回退到 cast 转换
		if result, err := cast.ToFloat64E(val); err == nil {
			return result
		}
	}

	if hasDefault {
		if val, err := strconv.ParseFloat(defaultVal, 64); err == nil {
			return val
		}
		c.logger.Errorf("Invalid default float value '%s' for key '%s', using 0", defaultVal, key)
	}
	return 0.0
}

// GetInt 获取整数配置
//
// 支持两种调用方式：
//   - GetInt("database.port", "5432") - 点号分隔的键名
//   - GetInt("database", "port", "5432") - 多参数形式
//
// 参数:
//   - parts: 可变参数，最后一个参数可选作为默认值
//
// 返回值:
//   - 整数类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetInt(parts ...string) int {
	if len(parts) == 0 {
		return 0
	}

	parser := &ParamParser{}
	key, defaultVal, hasDefault := parser.ParseKeyAndDefault(parts)

	if val, exists := c.getRaw(key); exists {
		// 快速路径：直接类型断言
		if i, ok := val.(int); ok {
			return i
		}
		if i, ok := val.(int64); ok {
			return int(i)
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
		// 回退到 cast 转换
		if result, err := cast.ToIntE(val); err == nil {
			return result
		}
	}

	if hasDefault {
		if val, err := strconv.Atoi(defaultVal); err == nil {
			return val
		}
		c.logger.Errorf("Invalid default int value '%s' for key '%s', using 0", defaultVal, key)
	}
	return 0
}

// GetString 获取字符串配置
//
// 支持两种调用方式：
//   - GetString("database.host", "localhost") - 点号分隔的键名
//   - GetString("database", "host", "localhost") - 多参数形式
//
// 参数:
//   - parts: 可变参数，最后一个参数可选作为默认值
//
// 返回值:
//   - 字符串类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetString(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	parser := &ParamParser{}
	key, defaultVal, hasDefault := parser.ParseKeyAndDefault(parts)

	if val, exists := c.getRaw(key); exists {
		// 快速路径：直接类型断言
		if s, ok := val.(string); ok {
			return s
		}
		// 回退到 cast 转换
		if result, err := cast.ToStringE(val); err == nil {
			return result
		}
	}

	if hasDefault {
		return defaultVal
	}
	return ""
}

// GetStringSlice 获取字符串切片配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 字符串切片类型的配置值
func (c *Config) GetStringSlice(key string) []string {
	if key == "" {
		return []string{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 使用新的原子存储系统
	val, exists := c.getRaw(key)
	if !exists {
		return []string{}
	}

	result, err := cast.ToStringSliceE(val)
	if err != nil {
		return []string{}
	}
	if result == nil {
		return []string{}
	}
	return result
}

// GetBoolSlice 获取布尔值切片配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 布尔值切片类型的配置值
func (c *Config) GetBoolSlice(key string) []bool {
	if key == "" {
		return []bool{}
	}

	// 使用新的原子存储系统获取原始值
	val, exists := c.getRaw(key)
	if !exists {
		val = nil
	}
	if val == nil {
		return []bool{}
	}

	switch v := val.(type) {
	case []bool:
		return v
	case []interface{}:
		result := make([]bool, 0, len(v))
		for _, item := range v {
			if b, ok := item.(bool); ok {
				result = append(result, b)
			} else if s, ok := item.(string); ok {
				if b, err := strconv.ParseBool(s); err == nil {
					result = append(result, b)
				}
			}
		}
		return result
	case []string:
		result := make([]bool, 0, len(v))
		for _, s := range v {
			if b, err := strconv.ParseBool(s); err == nil {
				result = append(result, b)
			}
		}
		return result
	default:
		return []bool{}
	}
}

// GetIntSlice 获取整数切片配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 整数切片类型的配置值
func (c *Config) GetIntSlice(key string) []int {
	if key == "" {
		return []int{}
	}

	// 使用新的原子存储系统
	val, exists := c.getRaw(key)
	if !exists {
		return []int{}
	}

	result, err := cast.ToIntSliceE(val)
	if err != nil {
		return []int{}
	}
	if result == nil {
		return []int{}
	}
	return result
}

// GetFloatSlice 获取浮点数切片配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 浮点数切片类型的配置值
func (c *Config) GetFloatSlice(key string) []float64 {
	if key == "" {
		return []float64{}
	}

	// 使用新的原子存储系统获取原始值
	val, exists := c.getRaw(key)
	if !exists {
		val = nil
	}
	c.logger.Debugf("GetFloatSlice[%s] - 原始值: %v (类型: %T)", key, val, val)
	if val == nil {
		c.logger.Debugf("GetFloatSlice[%s] - 值为nil，返回空切片", key)
		return []float64{}
	}

	// 直接类型判断和转换，避免使用有问题的cast.ToSliceE()
	switch v := val.(type) {
	case []float64:
		// 已经是float64切片，直接返回
		c.logger.Debugf("GetFloatSlice[%s] - 直接返回[]float64: %v", key, v)
		return v

	case []interface{}:
		// interface{}切片，逐个转换
		result := make([]float64, 0, len(v))
		for i, item := range v {
			if f, err := cast.ToFloat64E(item); err == nil {
				c.logger.Debugf("GetFloatSlice[%s] - 元素[%d] %v -> %f", key, i, item, f)
				result = append(result, f)
			} else {
				c.logger.Debugf("GetFloatSlice[%s] - 元素[%d] %v 转换失败: %v", key, i, item, err)
			}
		}
		c.logger.Debugf("GetFloatSlice[%s] - []interface{}转换结果: %v (长度: %d)", key, result, len(result))
		return result

	case []string:
		// 字符串切片，转换为float64
		result := make([]float64, 0, len(v))
		for i, s := range v {
			if f, err := cast.ToFloat64E(s); err == nil {
				c.logger.Debugf("GetFloatSlice[%s] - 字符串[%d] %s -> %f", key, i, s, f)
				result = append(result, f)
			} else {
				c.logger.Debugf("GetFloatSlice[%s] - 字符串[%d] %s 转换失败: %v", key, i, s, err)
			}
		}
		c.logger.Debugf("GetFloatSlice[%s] - []string转换结果: %v (长度: %d)", key, result, len(result))
		return result

	case []int:
		// 整数切片，转换为float64
		result := make([]float64, 0, len(v))
		for i, n := range v {
			f := float64(n)
			c.logger.Debugf("GetFloatSlice[%s] - 整数[%d] %d -> %f", key, i, n, f)
			result = append(result, f)
		}
		c.logger.Debugf("GetFloatSlice[%s] - []int转换结果: %v (长度: %d)", key, result, len(result))
		return result

	case []float32:
		// float32切片，转换为float64
		result := make([]float64, 0, len(v))
		for i, f32 := range v {
			f64 := float64(f32)
			c.logger.Debugf("GetFloatSlice[%s] - float32[%d] %f -> %f", key, i, f32, f64)
			result = append(result, f64)
		}
		c.logger.Debugf("GetFloatSlice[%s] - []float32转换结果: %v (长度: %d)", key, result, len(result))
		return result

	default:
		// 尝试作为单个值转换
		if f, err := cast.ToFloat64E(val); err == nil {
			c.logger.Debugf("GetFloatSlice[%s] - 单个值转换: %v -> [%f]", key, val, f)
			return []float64{f}
		}
		c.logger.Debugf("GetFloatSlice[%s] - 无法转换类型 %T，返回空切片", key, val)
		return []float64{}
	}
}

// GetStringMap 获取字符串映射配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 字符串映射类型的配置值，映射值为任意类型
func (c *Config) GetStringMap(key string) map[string]any {
	if key == "" {
		return make(map[string]any)
	}

	// 使用新的原子存储系统
	val, exists := c.getRaw(key)
	if exists {
		// 如果直接存在，尝试转换
		if result, err := cast.ToStringMapE(val); err == nil && result != nil {
			return result
		}
	}

	// 如果不存在或转换失败，尝试从扁平化数据重构
	data := c.loadData()
	if reconstructed, found := c.reconstructNestedValue(data, key); found {
		if result, err := cast.ToStringMapE(reconstructed); err == nil && result != nil {
			return result
		}
	}

	return make(map[string]any)
}

// GetStringMapString 获取字符串-字符串映射配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 字符串到字符串的映射类型配置值
func (c *Config) GetStringMapString(key string) map[string]string {
	if key == "" {
		return make(map[string]string)
	}

	// 使用新的原子存储系统
	val, exists := c.getRaw(key)
	if exists {
		// 如果直接存在，尝试转换
		if result, err := cast.ToStringMapStringE(val); err == nil && result != nil {
			return result
		}
	}

	// 如果不存在或转换失败，尝试从扁平化数据重构
	data := c.loadData()
	if reconstructed, found := c.reconstructNestedValue(data, key); found {
		if result, err := cast.ToStringMapStringE(reconstructed); err == nil && result != nil {
			return result
		}
	}

	return make(map[string]string)
}

// GetTime 获取时间配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 时间类型的配置值
func (c *Config) GetTime(key string) time.Time {
	if key == "" {
		return time.Time{}
	}

	// 使用新的原子存储系统
	if val, exists := c.getRaw(key); exists {
		if result, err := cast.ToTimeE(val); err == nil {
			return result
		}
	}
	return time.Time{}
}

// GetDuration 获取时间间隔配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 时间间隔类型的配置值
func (c *Config) GetDuration(key string) time.Duration {
	if key == "" {
		return 0
	}

	// 使用新的原子存储系统
	if val, exists := c.getRaw(key); exists {
		if result, err := cast.ToDurationE(val); err == nil {
			return result
		}
	}
	return 0
}

// GetWithError 获取配置值并返回错误信息
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 配置值和可能的错误
func (c *Config) GetWithError(key string) (any, error) {
	if key == "" {
		return nil, fmt.Errorf("empty configuration key")
	}

	// 使用新的原子存储系统
	val, exists := c.getRaw(key)
	if !exists {
		return nil, fmt.Errorf("configuration key '%s' not found", key)
	}
	return val, nil
}
