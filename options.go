package sysconf

import "github.com/spf13/pflag"

// WithPath 设置配置文件路径
func WithPath(path string) Option {
	return func(c *Config) {
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

// WithContent 设置默认配置文件内容
func WithContent(content string) Option {
	return func(c *Config) {
		c.content = content
	}
}

// WithBindPFlags 设置命令行标志绑定
func WithBindPFlags(pflag ...*pflag.FlagSet) Option {
	return func(c *Config) {
		c.pflag = pflag
	}
}
