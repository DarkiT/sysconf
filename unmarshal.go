package sysconf

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Unmarshal 将配置解析到结构体
// key 为空时解析整个配置，否则解析指定的配置段
// 支持 default 标签设置默认值
// 支持 required 标签验证必填字段
// 支持大驼峰命名风格的结构体字段自动映射到下划线风格的配置键名
func (c *Config) Unmarshal(obj any, key ...string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 如果是结构体指针，则设置默认值
	if reflect.TypeOf(obj).Kind() == reflect.Ptr && reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
		c.logger.Debugf("Setting default values")
		if err := setDefaultValues(obj); err != nil {
			c.logger.Errorf("Failed to set default values: %v", err)
			return fmt.Errorf("set defaults: %w", err)
		}
	}

	// 创建解码器配置
	c.logger.Debugf("Creating decoder config")
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(time.RFC3339),
			stringToSliceHookFunc(),
			stringToMapHookFunc(),
		),
		Result:           obj,
		ZeroFields:       false,
		WeaklyTypedInput: true,
		TagName:          strings.Join([]string{"config", strings.Join(viper.SupportedExts, ", ")}, ","),
		// 启用字段名到键名的自动转换，支持驼峰命名到下划线命名的转换
		MatchName: func(mapKey, fieldName string) bool {
			// 1. 直接匹配
			if mapKey == fieldName {
				return true
			}

			// 2. 驼峰命名转下划线匹配
			snakeFieldName := camelToSnake(fieldName)
			if mapKey == snakeFieldName {
				return true
			}

			// 3. 下划线转驼峰匹配
			camelMapKey := snakeToCamel(mapKey)
			return camelMapKey == fieldName
		},
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		c.logger.Errorf("Failed to create decoder: %v", err)
		return fmt.Errorf("create decoder: %w", err)
	}

	// 获取配置数据
	var data map[string]any
	if len(key) > 0 && key[0] != "" {
		configKey := strings.Join(key, ".")
		c.logger.Debugf("Getting sub-config: %s", configKey)
		sub := c.viper.Sub(configKey)
		if sub != nil {
			data = sub.AllSettings()
		}
	} else {
		c.logger.Debugf("Getting all config settings")
		data = c.viper.AllSettings()
	}

	// 如果没有配置数据，保持默认值
	if len(data) == 0 {
		c.logger.Warnf("No config data found, using default values")
		return nil
	}

	// 解码配置
	c.logger.Debugf("Decoding config")
	if err := decoder.Decode(data); err != nil {
		c.logger.Errorf("Failed to decode config: %v", err)
		return fmt.Errorf("decode config: %w", err)
	}

	// 如果是结构体指针，则验证必填字段
	if reflect.TypeOf(obj).Kind() == reflect.Ptr && reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
		c.logger.Debugf("Validating required fields")
		if err := validateStruct(obj); err != nil {
			c.logger.Errorf("Field validation failed: %v", err)
			return fmt.Errorf("validate: %w", err)
		}
	}

	c.logger.Infof("Config parsed successfully")
	return nil
}

// UnmarshalKey 将指定键的配置反序列化为结构体
func (c *Config) UnmarshalKey(key string, result interface{}, validator func(interface{}) error) error {
	if key == "" {
		c.logger.Errorf("UnmarshalKey: Key is empty")
		return fmt.Errorf("key is empty")
	}
	
	c.logger.Debugf("UnmarshalKey: Parsing key %s", key)
	c.mu.RLock()
	exists := c.viper.IsSet(key)
	c.mu.RUnlock()
	
	if !exists {
		c.logger.Warnf("UnmarshalKey: Key %s does not exist", key)
		// 键不存在时不报错，可能使用的是结构体默认值
	}
	
	// 使用Sub获取子配置
	c.mu.RLock()
	subConfig := c.viper.Sub(key)
	c.mu.RUnlock()
	
	if subConfig != nil {
		// 使用子配置进行解析
		tempConfig := &Config{
			viper:  subConfig,
			logger: c.logger,
		}
		return tempConfig.Unmarshal(result)
	}
	
	// 如果sub返回nil，尝试直接获取并解析值
	c.mu.RLock()
	val := c.viper.Get(key)
	c.mu.RUnlock()
	
	if val == nil {
		c.logger.Warnf("UnmarshalKey: Value for key %s is nil", key)
		// 创建一个新的配置实例并使用默认值
		return c.Unmarshal(result)
	}
	
	// 将值转换为合适的类型并填充到结构体中
	c.logger.Debugf("UnmarshalKey: Using direct value for parsing")
	return c.Unmarshal(result)
}

// Defaulter 定义可设置默认值的结构体接口
type Defaulter interface {
	SetDefaults() error
}
