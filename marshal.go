package sysconf

import (
	"fmt"
	"reflect"
	"strings"

	mapstructure "github.com/go-viper/mapstructure/v2"
)

// Marshal 将结构体或其他数据类型序列化并保存到配置中
//
// 参数:
//   - value: 要序列化的值（通常是结构体）
//   - prefix: 可选的配置键前缀，用于指定保存路径
//
// 返回值:
//   - error: 序列化过程中遇到的错误，成功则为nil
func (c *Config) Marshal(value any, prefix ...string) error {
	if c.viper == nil {
		return fmt.Errorf("viper instance not initialized")
	}

	var configMap map[string]any
	if err := mapstructure.Decode(value, &configMap); err != nil {
		return fmt.Errorf("failed to convert struct to map: %v", err)
	}

	configMap = deepMerge(c.viper.AllSettings(), configMap)

	if len(prefix) > 0 {
		c.setMapToViper(strings.Join(prefix, "."), configMap)
	} else {
		c.setMapToViper("", configMap)
	}

	return nil
}

// setMapToViper 递归设置map到viper中
//
// 参数:
//   - prefix: 配置键前缀
//   - m: 要设置的映射数据
func (c *Config) setMapToViper(prefix string, m map[string]any) {
	for key, val := range m {
		k := key
		if prefix != "" {
			k = prefix + "." + key
		}

		// 先检查值是否为 nil
		if val == nil {
			c.Set(k, nil)
			continue
		}

		valType := reflect.TypeOf(val)
		if valType != nil && valType.Kind() == reflect.Map {
			// 如果值是map，递归处理
			if mapVal, ok := val.(map[string]any); ok {
				c.setMapToViper(k, mapVal)
			} else {
				// 类型断言失败，直接设置值
				c.Set(k, val)
			}
		} else {
			// 直接设置值
			c.Set(k, val)
		}
	}
}

// deepMerge 深度合并两个 map
//
// 参数:
//   - m1: 目标映射，合并结果将存储在此
//   - m2: 源映射，其值将合并到m1中
//
// 返回值:
//   - map[string]any: 合并后的映射
func deepMerge(m1, m2 map[string]any) map[string]any {
	if m1 == nil {
		m1 = make(map[string]any)
	}

	for k, v2 := range m2 {
		// 处理 v2 为 nil 的情况
		if v2 == nil {
			m1[k] = nil
			continue
		}

		// 查看 m1 中是否有相同的键
		if v1, ok := m1[k]; ok && v1 != nil {
			// 如果 m1 和 m2 在这个键上的值都是 map，递归合并
			m1Map, ok1 := v1.(map[string]any)
			m2Map, ok2 := v2.(map[string]any)

			if ok1 && ok2 {
				// 递归合并这两个子 map
				m1[k] = deepMerge(m1Map, m2Map)
				continue
			}
		}
		// 如果不是 map 类型或 m1 中不存在该键，直接赋值
		m1[k] = v2
	}

	return m1
}
