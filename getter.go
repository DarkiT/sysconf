package sysconf

import (
	"strconv"
	"strings"
	"time"
)

// Get 获取配置值
//
// 参数:
//   - key: 配置键名
//   - def: 可选的默认值，当配置不存在时返回
//
// 返回值:
//   - 配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) Get(key string, def ...any) any {
	if key == "" {
		if len(def) > 0 {
			return def[0]
		}
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	val := c.viper.Get(key)
	if val == nil && len(def) > 0 {
		return def[0]
	}
	return val
}

// GetBool 获取布尔值配置
//
// 参数:
//   - parts: 可变参数，第一个参数是键名（可以包含点号分隔的路径），最后一个参数可选作为默认值
//
// 返回值:
//   - 布尔类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetBool(parts ...string) bool {
	if len(parts) == 0 {
		return false
	}

	var defaultVal bool
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseBool(parts[len(parts)-1])
		}
	} else {
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseBool(parts[len(parts)-1])
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetBool(key)
}

// GetFloat 获取浮点数配置
//
// 参数:
//   - parts: 可变参数，第一个参数是键名（可以包含点号分隔的路径），最后一个参数可选作为默认值
//
// 返回值:
//   - 浮点类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetFloat(parts ...string) float64 {
	if len(parts) == 0 {
		return 0
	}

	var defaultVal float64
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseFloat(parts[len(parts)-1], 64)
		}
	} else {
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseFloat(parts[len(parts)-1], 64)
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetFloat64(key)
}

// GetInt 获取整数配置
//
// 参数:
//   - parts: 可变参数，第一个参数是键名（可以包含点号分隔的路径），最后一个参数可选作为默认值
//
// 返回值:
//   - 整数类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetInt(parts ...string) int {
	if len(parts) == 0 {
		return 0
	}

	var defaultVal int
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal, _ = strconv.Atoi(parts[len(parts)-1])
		}
	} else {
		if len(parts) > 1 {
			defaultVal, _ = strconv.Atoi(parts[len(parts)-1])
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetInt(key)
}

// GetString 获取字符串配置
//
// 参数:
//   - parts: 可变参数，第一个参数是键名（可以包含点号分隔的路径），最后一个参数可选作为默认值
//
// 返回值:
//   - 字符串类型的配置值，如果键不存在且提供了默认值则返回默认值
func (c *Config) GetString(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	var defaultVal string
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal = parts[len(parts)-1]
		}
	} else {
		if len(parts) > 1 {
			defaultVal = parts[len(parts)-1]
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetString(key)
}

// GetStringSlice 获取字符串切片配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 字符串切片类型的配置值
func (c *Config) GetStringSlice(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetStringSlice(key)
}

// GetIntSlice 获取整数切片配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 整数切片类型的配置值
func (c *Config) GetIntSlice(key string) []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetIntSlice(key)
}

// GetStringMap 获取字符串映射配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 字符串映射类型的配置值，映射值为任意类型
func (c *Config) GetStringMap(key string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetStringMap(key)
}

// GetStringMapString 获取字符串-字符串映射配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 字符串到字符串的映射类型配置值
func (c *Config) GetStringMapString(key string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetStringMapString(key)
}

// GetTime 获取时间配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 时间类型的配置值
func (c *Config) GetTime(key string) time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetTime(key)
}

// GetDuration 获取时间间隔配置
//
// 参数:
//   - key: 配置键名
//
// 返回值:
//   - 时间间隔类型的配置值
func (c *Config) GetDuration(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetDuration(key)
}
