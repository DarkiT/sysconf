package sysconf

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/spf13/cast"
)

// 类型转换缓存，避免重复反射操作
var (
	typeCache   = make(map[reflect.Type]typeInfo)
	typeCacheMu sync.RWMutex
)

type typeInfo struct {
	kind       reflect.Kind
	isString   bool
	isInt      bool
	isFloat    bool
	isBool     bool
	isDuration bool
	isTime     bool
}

// GetAs 泛型获取配置值，提供类型安全的访问方式
// 支持类型: string, int, int32, int64, float32, float64, bool, time.Duration, time.Time
//
// 使用示例:
//
//	host := cfg.GetAs[string]("database.host", "localhost")
//	port := cfg.GetAs[int]("database.port", 5432)
//	timeout := cfg.GetAs[time.Duration]("timeout", 30*time.Second)
func GetAs[T any](c *Config, key string, defaultValue ...T) T {
	if key == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		var zero T
		return zero
	}

	// 优先从缓存获取
	if val, exists := c.getCachedValue(key); exists {
		if converted, ok := convertValue[T](val); ok {
			return converted
		}
	}

	// 从viper获取
	c.mu.RLock()
	val := c.viper.Get(key)
	c.mu.RUnlock()

	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		var zero T
		return zero
	}

	if converted, ok := convertValue[T](val); ok {
		return converted
	}

	// 转换失败，返回默认值或零值
	if len(defaultValue) > 0 {
		c.logger.Warnf("Failed to convert value for key '%s', using default", key)
		return defaultValue[0]
	}

	var zero T
	c.logger.Warnf("Failed to convert value for key '%s', using zero value", key)
	return zero
}

// GetSliceAs 泛型获取切片配置值
// 支持类型: []string, []int, []float64, []bool
//
// 使用示例:
//
//	features := cfg.GetSliceAs[string]("server.features")
//	ports := cfg.GetSliceAs[int]("server.ports")
func GetSliceAs[T any](c *Config, key string) []T {
	if key == "" {
		return []T{}
	}

	c.mu.RLock()
	val := c.viper.Get(key)
	c.mu.RUnlock()

	if val == nil {
		return []T{}
	}

	// 尝试直接类型断言
	if slice, ok := val.([]T); ok {
		return slice
	}

	// 处理interface{}切片
	if interfaceSlice, ok := val.([]interface{}); ok {
		result := make([]T, 0, len(interfaceSlice))
		for _, item := range interfaceSlice {
			if converted, ok := convertValue[T](item); ok {
				result = append(result, converted)
			}
		}
		return result
	}

	// 处理其他切片类型的转换
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Slice {
		result := make([]T, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			item := rv.Index(i).Interface()
			if converted, ok := convertValue[T](item); ok {
				result = append(result, converted)
			}
		}
		return result
	}

	return []T{}
}

// getTypeInfo 获取类型信息（带缓存）
func getTypeInfo[T any]() typeInfo {
	var zero T
	targetType := reflect.TypeOf(zero)

	// 先尝试从缓存读取
	typeCacheMu.RLock()
	if info, exists := typeCache[targetType]; exists {
		typeCacheMu.RUnlock()
		return info
	}
	typeCacheMu.RUnlock()

	// 缓存未命中，计算类型信息
	info := typeInfo{
		kind:       targetType.Kind(),
		isString:   targetType.Kind() == reflect.String,
		isInt:      targetType.Kind() >= reflect.Int && targetType.Kind() <= reflect.Int64,
		isFloat:    targetType.Kind() == reflect.Float32 || targetType.Kind() == reflect.Float64,
		isBool:     targetType.Kind() == reflect.Bool,
		isDuration: targetType == reflect.TypeOf(time.Duration(0)),
		isTime:     targetType == reflect.TypeOf(time.Time{}),
	}

	// 写入缓存
	typeCacheMu.Lock()
	typeCache[targetType] = info
	typeCacheMu.Unlock()

	return info
}

// convertValue 通用类型转换函数
func convertValue[T any](val interface{}) (T, bool) {
	var zero T

	// 快速路径：直接类型匹配
	if converted, ok := val.(T); ok {
		return converted, true
	}

	// 如果值为nil，直接返回零值
	if val == nil {
		return zero, false
	}

	// 获取缓存的类型信息
	typeInfo := getTypeInfo[T]()

	// 基于类型信息进行优化转换
	if typeInfo.isString {
		if str, err := cast.ToStringE(val); err == nil {
			if result, ok := any(str).(T); ok {
				return result, true
			}
		}
	} else if typeInfo.isInt {
		switch typeInfo.kind {
		case reflect.Int:
			if i, err := cast.ToIntE(val); err == nil {
				if result, ok := any(i).(T); ok {
					return result, true
				}
			}
		case reflect.Int32:
			if i, err := cast.ToInt32E(val); err == nil {
				if result, ok := any(i).(T); ok {
					return result, true
				}
			}
		case reflect.Int64:
			if i, err := cast.ToInt64E(val); err == nil {
				if result, ok := any(i).(T); ok {
					return result, true
				}
			}
		}
	} else if typeInfo.isFloat {
		switch typeInfo.kind {
		case reflect.Float32:
			if f, err := cast.ToFloat32E(val); err == nil {
				if result, ok := any(f).(T); ok {
					return result, true
				}
			}
		case reflect.Float64:
			if f, err := cast.ToFloat64E(val); err == nil {
				if result, ok := any(f).(T); ok {
					return result, true
				}
			}
		}
	} else if typeInfo.isBool {
		if b, err := cast.ToBoolE(val); err == nil {
			if result, ok := any(b).(T); ok {
				return result, true
			}
		}
	}

	// 处理time.Duration特殊情况
	if typeInfo.isDuration {
		switch v := val.(type) {
		case string:
			if d, err := time.ParseDuration(v); err == nil {
				if result, ok := any(d).(T); ok {
					return result, true
				}
			}
		case int64:
			if result, ok := any(time.Duration(v)).(T); ok {
				return result, true
			}
		}
	}

	// 处理time.Time特殊情况
	if typeInfo.isTime {
		if timeVal, err := cast.ToTimeE(val); err == nil {
			if result, ok := any(timeVal).(T); ok {
				return result, true
			}
		}
	}

	return zero, false
}

// MustGetAs 泛型获取配置值，如果不存在或转换失败则panic
// 适用于必须存在的配置项
func MustGetAs[T any](c *Config, key string) T {
	if key == "" {
		panic("empty configuration key")
	}

	c.mu.RLock()
	val := c.viper.Get(key)
	c.mu.RUnlock()

	if val == nil {
		panic(fmt.Sprintf("configuration key '%s' not found", key))
	}

	if converted, ok := convertValue[T](val); ok {
		return converted
	}

	panic(fmt.Sprintf("failed to convert value for key '%s' to type %T", key, *new(T)))
}

// GetWithFallback 获取配置值，支持多个fallback键
// 按顺序尝试每个键，直到找到有效值
//
// 使用示例:
//
//	port := cfg.GetWithFallback[int]("server.port", "app.port", "port")
func GetWithFallback[T any](c *Config, keys ...string) T {
	for _, key := range keys {
		if key != "" {
			c.mu.RLock()
			val := c.viper.Get(key)
			c.mu.RUnlock()

			if val != nil {
				if converted, ok := convertValue[T](val); ok {
					return converted
				}
			}
		}
	}

	var zero T
	return zero
}
