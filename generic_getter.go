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
	isUint     bool
	isFloat    bool
	isBool     bool
	isDuration bool
	isTime     bool
	converter  converterFunc // 预编译的转换函数
}

// GetAs 泛型获取配置值，提供类型安全的访问方式。
// 这是包级函数（Go 不允许泛型方法），与 Get/GetInt 走同一条 getRaw 路径：
// 环境变量优先于缓存与文件，避免 GetAs 与 Get* 读到不同来源。
// 支持类型: string, int, int8..int64, uint..uint64, float32/64, bool, time.Duration, time.Time
//
// 使用示例:
//
//	host := sysconf.GetAs[string](cfg, "database.host", "localhost")
//	port := sysconf.GetAs[int](cfg, "database.port", 5432)
//	timeout := sysconf.GetAs[time.Duration](cfg, "timeout", 30*time.Second)
func GetAs[T any](c *Config, key string, defaultValue ...T) T {
	if c == nil || key == "" {
		return defaultOrZero(defaultValue)
	}

	val, exists := c.getRaw(key)
	if !exists || val == nil {
		return defaultOrZero(defaultValue)
	}

	if converted, ok := convertValue[T](val); ok {
		return converted
	}

	if len(defaultValue) > 0 {
		c.logger.Warnf("Failed to convert value for key '%s', using default", key)
		return defaultValue[0]
	}

	var zero T
	c.logger.Warnf("Failed to convert value for key '%s', using zero value", key)
	return zero
}

// GetAsWithError 返回转换后的值和错误，便于用 errors.Is 区分键不存在或转换失败。
func GetAsWithError[T any](cfg *Config, key string) (T, error) {
	var zero T

	if cfg == nil {
		return zero, ErrNilConfig
	}
	if key == "" {
		return zero, ErrEmptyKey
	}

	val, exists := cfg.getRaw(key)
	if !exists || val == nil {
		return zero, fmt.Errorf("%w: %q", ErrKeyNotFound, key)
	}

	converted, err := convertTo[T](val)
	if err != nil {
		return zero, fmt.Errorf("%w: key %q to %T: %v", ErrConversion, key, zero, err)
	}

	return converted, nil
}

// GetSliceAs 泛型获取切片配置值
// 支持类型: []string, []int, []float64, []bool
//
// 使用示例:
//
//	features := sysconf.GetSliceAs[string](cfg, "server.features")
//	ports := sysconf.GetSliceAs[int](cfg, "server.ports", []int{8080})
func GetSliceAs[T any](c *Config, key string, def ...[]T) []T {
	if c == nil || key == "" {
		return firstSliceDefault(def)
	}

	val, exists := c.getRaw(key)
	if !exists || val == nil {
		return firstSliceDefault(def)
	}

	// 尝试直接类型断言
	if slice, ok := val.([]T); ok {
		return append([]T(nil), slice...)
	}

	// 处理interface{}切片
	if interfaceSlice, ok := val.([]any); ok {
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

	return firstSliceDefault(def)
}

// getTypeInfo 获取类型信息（带缓存），使用 sync.Map 实现无锁读取
func getTypeInfo[T any]() *typeInfo {
	targetType := reflect.TypeFor[T]()

	// 无锁快速路径：从 sync.Map 读取
	if cached, ok := typeCache.Load(targetType); ok {
		return cached.(*typeInfo)
	}

	// 缓存未命中，计算类型信息并创建预编译转换器
	info := &typeInfo{
		kind:       targetType.Kind(),
		isString:   targetType.Kind() == reflect.String,
		isInt:      targetType.Kind() >= reflect.Int && targetType.Kind() <= reflect.Int64,
		isUint:     targetType.Kind() >= reflect.Uint && targetType.Kind() <= reflect.Uint64,
		isFloat:    targetType.Kind() == reflect.Float32 || targetType.Kind() == reflect.Float64,
		isBool:     targetType.Kind() == reflect.Bool,
		isDuration: targetType == reflect.TypeFor[time.Duration](),
		isTime:     targetType == reflect.TypeFor[time.Time](),
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
		case reflect.Int8:
			return func(val any) (any, bool) {
				if i, err := cast.ToInt8E(val); err == nil {
					return i, true
				}
				return nil, false
			}
		case reflect.Int16:
			return func(val any) (any, bool) {
				if i, err := cast.ToInt16E(val); err == nil {
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
	case info.isUint:
		// 补齐无符号整型：此前 uint/uint8/.../uint64 不满足 isInt，直接落 default 恒返回 false，
		// GetAs[uint] 静默退成零值、MustGetAs[uint] 必 panic。
		switch info.kind {
		case reflect.Uint8:
			return func(val any) (any, bool) {
				if u, err := cast.ToUint8E(val); err == nil {
					return u, true
				}
				return nil, false
			}
		case reflect.Uint16:
			return func(val any) (any, bool) {
				if u, err := cast.ToUint16E(val); err == nil {
					return u, true
				}
				return nil, false
			}
		case reflect.Uint32:
			return func(val any) (any, bool) {
				if u, err := cast.ToUint32E(val); err == nil {
					return u, true
				}
				return nil, false
			}
		case reflect.Uint64:
			return func(val any) (any, bool) {
				if u, err := cast.ToUint64E(val); err == nil {
					return u, true
				}
				return nil, false
			}
		default:
			return func(val any) (any, bool) {
				if u, err := cast.ToUintE(val); err == nil {
					return u, true
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
func convertValue[T any](val any) (T, bool) {
	var zero T

	// 快速路径：直接类型匹配
	if converted, ok := deepCloneValue(val).(T); ok {
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
func convertTo[T any](val any) (T, error) {
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
//	port := sysconf.GetWithFallback[int](cfg, "server.port", "app.port", "port")
func GetWithFallback[T any](c *Config, keys ...string) T {
	if c == nil {
		var zero T
		return zero
	}

	for _, key := range keys {
		if key != "" {
			if val, exists := c.getRaw(key); exists && val != nil {
				if converted, ok := convertValue[T](val); ok {
					return converted
				}
			}
		}
	}

	var zero T
	return zero
}
