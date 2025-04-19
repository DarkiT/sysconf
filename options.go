package sysconf

import (
	"path/filepath"
	"strings"

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

// WithContent 设置默认配置文件内容
func WithContent(content string) Option {
	return func(c *Config) {
		c.content = content
	}
}

// WithBindPFlags 设置命令行标志绑定
func WithBindPFlags(flags ...*pflag.FlagSet) Option {
	return func(c *Config) {
		c.pflag = flags
	}
}

// WithLogger 设置配置的日志记录器
func WithLogger(logger Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}

// WithExcludedFlags 设置要排除的命令行标志（不会被绑定）
func WithExcludedFlags(flags []string) Option {
	return func(c *Config) {
		if c.excludedFlags == nil {
			c.excludedFlags = make(map[string]bool)
		}
		for _, flag := range flags {
			c.excludedFlags[flag] = true
		}
	}
}
