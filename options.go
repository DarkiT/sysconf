package sysconf

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// WithPath 设置配置文件路径
func WithPath(path string) Option {
	return func(c *Config) {
		// 获取文件名部分
		fileName := filepath.Base(path)

		// 处理两种情况：
		// 1. 文件有明确的扩展名（如config.yaml）
		// 2. 文件是隐藏文件但有扩展名（如.config.yaml）
		ext := filepath.Ext(path)
		if ext != "" {
			c.mode = strings.TrimPrefix(ext, ".")
			c.path = filepath.Dir(path)
			c.name = strings.TrimSuffix(fileName, ext)
			return
		}

		// 处理特殊情况：隐藏文件没有明确扩展名（如.config）
		if strings.HasPrefix(fileName, ".") && !strings.Contains(fileName[1:], ".") {
			// 这是一个没有扩展名的隐藏文件
			c.path = filepath.Dir(path)
			c.name = fileName // 保留完整的隐藏文件名作为配置名称
			// 注意：这种情况下需要通过WithMode显式设置配置模式
			return
		}

		// 如果是一个目录路径，直接使用
		c.path = path
	}
}

// WithMode 设置配置文件模式
func WithMode(mode string) Option {
	return func(c *Config) {
		c.mode = mode
	}
}

// WithName 设置配置文件名称
func WithName(name string) Option {
	return func(c *Config) {
		c.name = name
	}
}

// WithEnvOptions 设置环境变量选项
func WithEnvOptions(opts EnvOptions) Option {
	return func(c *Config) {
		c.envOptions = opts
	}
}

// WithEnv 便利函数：启用环境变量并设置前缀，默认开启智能大小写匹配
func WithEnv(prefix string) Option {
	return WithEnvOptions(EnvOptions{
		Prefix:    prefix,
		Enabled:   true,
		SmartCase: true, // 🆕 默认启用智能大小写匹配
	})
}

// WithEnvSmartCase 便利函数：设置环境变量选项并明确指定智能大小写匹配
func WithEnvSmartCase(prefix string, smartCase bool) Option {
	return WithEnvOptions(EnvOptions{
		Prefix:    prefix,
		Enabled:   true,
		SmartCase: smartCase,
	})
}

// WithContent 设置默认配置文件内容
func WithContent(content string) Option {
	return func(c *Config) {
		c.content = content
	}
}

// WithBindPFlags 设置命令行标志绑定
func WithBindPFlags(flags ...*pflag.FlagSet) Option {
	return func(c *Config) {
		c.pflags = flags
	}
}

// PFlagOptions 命令行标志绑定扩展选项
type PFlagOptions struct {
	FlagSets    []*pflag.FlagSet
	KeyMapper   func(flag *pflag.Flag) string
	OnlyChanged bool
	Validate    func(flag *pflag.Flag) error
}

// WithPFlags 兼容别名：绑定命令行标志
func WithPFlags(flags ...*pflag.FlagSet) Option {
	return WithBindPFlags(flags...)
}

// WithPFlagOptions 使用扩展选项绑定命令行标志
func WithPFlagOptions(opts PFlagOptions) Option {
	return func(c *Config) {
		c.pflags = opts.FlagSets
		c.pflagOptions = opts
	}
}

// WithLogger 设置配置的日志记录器
func WithLogger(logger Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}

// WithValidator 添加配置验证器
func WithValidator(validator ConfigValidator) Option {
	return func(c *Config) {
		if c.validators == nil {
			c.validators = make([]ConfigValidator, 0)
		}
		c.validators = append(c.validators, validator)
	}
}

// WithValidateFunc 添加配置验证函数（便利方法）
func WithValidateFunc(fn func(config map[string]any) error) Option {
	return WithValidator(ConfigValidateFunc(fn))
}

// WithValidators 批量添加多个验证器
func WithValidators(validators ...ConfigValidator) Option {
	return func(c *Config) {
		if c.validators == nil {
			c.validators = make([]ConfigValidator, 0, len(validators))
		}
		c.validators = append(c.validators, validators...)
	}
}

// WithCrypto 设置配置加密选项
func WithCrypto(opts CryptoOptions) Option {
	return func(c *Config) {
		c.cryptoOptions = opts
	}
}

// WithEncryption 便利函数：启用配置加密并设置密钥
// key: 加密密钥，如果为空则生成随机密钥
func WithEncryption(key string) Option {
	return WithCrypto(CryptoOptions{
		Enabled: true,
		Key:     key,
	})
}

// WithEncryptionCrypto 便利函数：启用配置加密并使用自定义加密器
// crypto: 自定义加密实现
func WithEncryptionCrypto(crypto ConfigCrypto) Option {
	return WithCrypto(CryptoOptions{
		Enabled: true,
		Crypto:  crypto,
	})
}

// WithWriteDebounceDelay 设置防抖写入延迟。
// delay > 0 时启用防抖写入，delay <= 0 时回落为立即写入。
func WithWriteDebounceDelay(delay time.Duration) Option {
	return func(c *Config) {
		if delay < 0 {
			delay = 0
		}
		c.writeDelay = delay
	}
}

// WithWatchDebounce 设置配置文件监听的防抖时间。
func WithWatchDebounce(delay time.Duration) Option {
	return func(c *Config) {
		if delay < 0 {
			delay = 0
		}
		c.watchDebounce = delay
	}
}

// WithCacheTiming 设置读取缓存的预热与重建延迟。
// 传入 0 或负值可用于禁用对应延迟并在同一 goroutine 中立即刷新。
func WithCacheTiming(warmup, rebuild time.Duration) Option {
	return func(c *Config) {
		if warmup < 0 {
			warmup = 0
		}
		if rebuild < 0 {
			rebuild = 0
		}
		c.cacheWarmupDelay = warmup
		c.cacheRebuildDelay = rebuild
	}
}
