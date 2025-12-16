package sysconf

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/spf13/cast"
)

// 类型转换缓存，使用 sync.Map 实现无锁读取
var typeCache sync.Map // map[reflect.Type]*typeInfo

// converterFunc 预编译的类型转换函数
type converterFunc func(val any) (any, bool)

type typeInfo struct {
	kind       reflect.Kind
	isString   bool
	isInt      bool
	isFloat    bool
	isBool     bool
	isDuration bool
	isTime     bool
	converter  converterFunc // 预编译的转换函数
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

	// 使用完整的 getRaw 查找链（包含嵌套查找、环境变量回退）
	val, exists := c.getRaw(key)
	if !exists || val == nil {
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

// GetAsWithError 返回转换后的值和错误，便于区分键不存在或转换失败的具体原因
func GetAsWithError[T any](cfg *Config, key string) (T, error) {
	var zero T

	if key == "" {
		return zero, fmt.Errorf("key cannot be empty")
	}

	raw := cfg.Get(key)
	if raw == nil {
		return zero, fmt.Errorf("key %q not found", key)
	}

	converted, err := convertTo[T](raw)
	if err != nil {
		return zero, fmt.Errorf("failed to convert key %q to %T: %w", key, zero, err)
	}

	return converted, nil
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

// getTypeInfo 获取类型信息（带缓存），使用 sync.Map 实现无锁读取
func getTypeInfo[T any]() *typeInfo {
	var zero T
	targetType := reflect.TypeOf(zero)

	// 无锁快速路径：从 sync.Map 读取
	if cached, ok := typeCache.Load(targetType); ok {
		return cached.(*typeInfo)
	}

	// 缓存未命中，计算类型信息并创建预编译转换器
	info := &typeInfo{
		kind:       targetType.Kind(),
		isString:   targetType.Kind() == reflect.String,
		isInt:      targetType.Kind() >= reflect.Int && targetType.Kind() <= reflect.Int64,
		isFloat:    targetType.Kind() == reflect.Float32 || targetType.Kind() == reflect.Float64,
		isBool:     targetType.Kind() == reflect.Bool,
		isDuration: targetType == reflect.TypeOf(time.Duration(0)),
		isTime:     targetType == reflect.TypeOf(time.Time{}),
	}

	// 为每种类型预编译转换函数
	info.converter = buildConverter[T](info)

	// 存入缓存（并发安全，可能有重复计算但无害）
	typeCache.Store(targetType, info)

	return info
}

// buildConverter 为特定类型构建预编译转换函数
// 注意：isDuration 和 isTime 必须在 isInt 之前检查，因为 time.Duration 底层是 int64
func buildConverter[T any](info *typeInfo) converterFunc {
	switch {
	case info.isDuration:
		// Duration 必须最先检查，因为其底层类型是 int64，会导致 isInt=true
		return func(val any) (any, bool) {
			switch v := val.(type) {
			case time.Duration:
				return v, true
			case string:
				if d, err := time.ParseDuration(v); err == nil {
					return d, true
				}
			case int64:
				return time.Duration(v), true
			case int:
				return time.Duration(v), true
			case float64:
				return time.Duration(int64(v)), true
			}
			// 回退：尝试 cast 库
			if d, err := cast.ToDurationE(val); err == nil {
				return d, true
			}
			return nil, false
		}
	case info.isTime:
		return func(val any) (any, bool) {
			if t, ok := val.(time.Time); ok {
				return t, true
			}
			if t, err := cast.ToTimeE(val); err == nil {
				return t, true
			}
			return nil, false
		}
	case info.isString:
		return func(val any) (any, bool) {
			if str, err := cast.ToStringE(val); err == nil {
				return str, true
			}
			return nil, false
		}
	case info.isInt:
		switch info.kind {
		case reflect.Int:
			return func(val any) (any, bool) {
				if i, err := cast.ToIntE(val); err == nil {
					return i, true
				}
				return nil, false
			}
		case reflect.Int32:
			return func(val any) (any, bool) {
				if i, err := cast.ToInt32E(val); err == nil {
					return i, true
				}
				return nil, false
			}
		case reflect.Int64:
			return func(val any) (any, bool) {
				if i, err := cast.ToInt64E(val); err == nil {
					return i, true
				}
				return nil, false
			}
		default:
			return func(val any) (any, bool) {
				if i, err := cast.ToIntE(val); err == nil {
					return i, true
				}
				return nil, false
			}
		}
	case info.isFloat:
		switch info.kind {
		case reflect.Float32:
			return func(val any) (any, bool) {
				if f, err := cast.ToFloat32E(val); err == nil {
					return f, true
				}
				return nil, false
			}
		default:
			return func(val any) (any, bool) {
				if f, err := cast.ToFloat64E(val); err == nil {
					return f, true
				}
				return nil, false
			}
		}
	case info.isBool:
		return func(val any) (any, bool) {
			if b, err := cast.ToBoolE(val); err == nil {
				return b, true
			}
			return nil, false
		}
	default:
		// 回退到通用转换
		return func(val any) (any, bool) {
			return nil, false
		}
	}
}

// convertValue 通用类型转换函数，使用预编译转换器
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

	// 获取缓存的类型信息（含预编译转换器）
	info := getTypeInfo[T]()

	// 使用预编译转换器
	if info.converter != nil {
		if result, ok := info.converter(val); ok {
			if converted, ok := result.(T); ok {
				return converted, true
			}
		}
	}

	return zero, false
}

// convertTo 将任意值尝试转换为目标类型，返回错误以便上层处理
func convertTo[T any](val interface{}) (T, error) {
	if converted, ok := convertValue[T](val); ok {
		return converted, nil
	}
	var zero T
	return zero, fmt.Errorf("convert failed")
}

// MustGetAs 泛型获取配置值，如果不存在或转换失败则panic
// 适用于必须存在的配置项
func MustGetAs[T any](c *Config, key string) T {
	val, err := GetAsWithError[T](c, key)
	if err != nil {
		panic(fmt.Sprintf("MustGetAs failed: %v", err))
	}
	return val
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
