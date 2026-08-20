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

		// 处理特殊情况：隐藏文件没有明确扩展名（如 .env、.config）。
		// filepath.Ext(".env") 会返回 ".env"，必须先于通用扩展名逻辑判断。
		if isHiddenFileWithoutExtension(fileName) {
			c.path = filepath.Dir(path)
			c.name = fileName
			c.configFileName = fileName
			switch fileName {
			case ".env":
				c.mode = "env"
			case ".dotenv":
				c.mode = "dotenv"
			}
			return
		}

		// 处理两种情况：
		// 1. 文件有明确的扩展名（如config.yaml）
		// 2. 文件是隐藏文件但有扩展名（如.config.yaml）
		ext := filepath.Ext(path)
		if ext != "" {
			c.mode = strings.TrimPrefix(ext, ".")
			c.path = filepath.Dir(path)
			c.name = strings.TrimSuffix(fileName, ext)
			c.configFileName = ""
			return
		}

		// 如果是一个目录路径，直接使用
		c.path = path
		c.configFileName = ""
	}
}

func isHiddenFileWithoutExtension(fileName string) bool {
	return fileName != "." &&
		fileName != ".." &&
		strings.HasPrefix(fileName, ".") &&
		!strings.Contains(fileName[1:], ".")
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

// WithEnv 启用带前缀的环境变量覆盖，并默认开启智能大小写匹配。
// 前缀为空时不会启用环境变量：无前缀会盲查 PATH/HOME/USER 等系统变量并静默覆盖配置。
// 若确需无前缀，请显式使用 WithEnvOptions。
func WithEnv(prefix string) Option {
	prefix = strings.TrimSpace(prefix)
	return WithEnvOptions(EnvOptions{
		Prefix:    prefix,
		Enabled:   prefix != "",
		SmartCase: true,
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

// WithFiles 按顺序加载多个配置文件，后者覆盖前者。第一个文件同时作为主文件（供 Watch/写回）。
func WithFiles(paths ...string) Option {
	return func(c *Config) {
		if len(paths) == 0 {
			return
		}
		WithPath(paths[0])(c)
		if len(paths) > 1 {
			c.overlayFiles = append(c.overlayFiles, paths[1:]...)
		}
	}
}

// WithOverlay 在主配置之上叠加更多文件，后者覆盖前者。
func WithOverlay(paths ...string) Option {
	return func(c *Config) {
		c.overlayFiles = append(c.overlayFiles, paths...)
	}
}

// WithSkipInitValidation 跳过 New/Load/热重载时的整份文档校验。
// Set/SetMultiple 的字段级校验仍然生效。适用于预定义验证器只做写入拦截的场景。
func WithSkipInitValidation() Option {
	return func(c *Config) {
		c.skipInitValidation = true
	}
}
