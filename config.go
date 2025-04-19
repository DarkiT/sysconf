package sysconf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	workPathOnce  sync.Once
	workPathValue string
	globalOnce    sync.Once
	globalConfig  *Config

	ErrInvalidKey       = errors.New("invalid configuration key")
	ErrInitGlobalConfig = errors.New("failed to initialize global config")
	// defaultExcludedFlags 默认要从viper绑定中排除的标志
	defaultExcludedFlags = map[string]bool{
		// 帮助标志
		"help": true,
		"h":    true,

		// 版本标志
		"version": true,
		"v":       true,

		// 完成相关标志
		"completion":            true,
		"gen-completion":        true,
		"completion-bash":       true,
		"completion-zsh":        true,
		"completion-fish":       true,
		"completion-powershell": true,
	}
)

// EnvOptions 环境变量配置选项
type EnvOptions struct {
	Prefix  string // 环境变量前缀
	Enabled bool   // 是否启用环境变量
}

// Config 配置结构体
type Config struct {
	viper                *viper.Viper
	logger               Logger           // 日志记录器
	path                 string           // 配置文件路径
	mode                 string           // 配置文件类型
	name                 string           // 配置文件名称
	content              string           // 默认配置文件内容
	pflag                []*pflag.FlagSet // 默认命令行标志绑定
	envOptions           EnvOptions       // 环境变量配置选项
	lastUpdate           time.Time        // 配置最后更新时间
	writeTimer           *time.Timer      // 延迟写入定时器
	pendingWrites        bool             // 是否有待写入的更改
	mu                   sync.RWMutex     // 读取操作的锁
	writeMu              sync.Mutex       // 写入操作的互斥锁
	defaultExcludedFlags map[string]bool  // 默认排除的标志
	excludedFlags        map[string]bool  // 用户自定义排除的标志
}

// Option 配置选项
type Option func(*Config)

// New 创建新的配置实例
func New(opts ...Option) (*Config, error) {
	workPathOnce.Do(func() {
		workPathValue = WorkPath()
	})
	// 创建默认配置
	c := &Config{
		viper:                viper.New(),
		path:                 workPathValue,
		mode:                 "yaml",
		defaultExcludedFlags: defaultExcludedFlags,
	}

	// 应用自定义选项
	for _, opt := range opts {
		opt(c)
	}

	// 初始化配置
	if err := c.initialize(); err != nil {
		return nil, fmt.Errorf("initialize config: %w", err)
	}

	return c, nil
}

// Default 获取全局单例配置实例
func Default(opts ...Option) *Config {
	globalOnce.Do(func() {
		var err error
		globalConfig, err = New(opts...)
		if err != nil {
			// 保留panic以确保程序终止
			panic(fmt.Errorf("%w: %v", ErrInitGlobalConfig, err))
		}
	})
	return globalConfig
}

// Register 注册配置项到全局配置
func Register(module, key string, value any) error {
	// 参数验证
	if module == "" || key == "" {
		return fmt.Errorf("register module or key is empty")
	}

	// 获取全局配置并设置值
	return Default().Set(module+"."+key, value)
}

// Watch 监听配置变化
func (c *Config) Watch(callbacks ...func()) {
	c.viper.OnConfigChange(func(e fsnotify.Event) {
		if e.Op&fsnotify.Write == 0 {
			return
		}

		c.mu.Lock()
		now := time.Now()
		if now.Sub(c.lastUpdate) < time.Second {
			c.mu.Unlock()
			return
		}
		c.lastUpdate = now
		c.mu.Unlock()

		c.logger.Infof("Config file change detected: %s", e.Name)
		if len(c.viper.AllKeys()) > 0 {
			for _, cb := range callbacks {
				cb()
			}
			c.logger.Debugf("Executed %d config change callbacks", len(callbacks))
		}
	})

	c.viper.WatchConfig()
	c.logger.Infof("Config file watching started")
}

// Viper 返回底层的 viper 实例
func (c *Config) Viper() *viper.Viper {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper
}

func (c *Config) createDefaultConfig() error {
	if c.content == "" {
		return nil
	}

	configFile := filepath.Join(c.path, c.name+"."+c.mode)

	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		c.logger.Errorf("Failed to create config directory: %v", err)
		return fmt.Errorf("create config directory: %w", err)
	}

	c.logger.Infof("Creating default config file: %s", configFile)
	if err := os.WriteFile(configFile, []byte(c.content), 0o644); err != nil {
		c.logger.Errorf("Failed to write default config: %v", err)
		return fmt.Errorf("write default config: %w", err)
	}

	if err := c.viper.ReadInConfig(); err != nil {
		c.logger.Errorf("Failed to read new config: %v", err)
		return fmt.Errorf("read new config: %w", err)
	}

	c.logger.Infof("Default config file created successfully")
	return nil
}

func (c *Config) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pendingWrites = false
	if c.writeTimer != nil {
		c.writeTimer.Stop()
	}

	// 确保有一个日志记录器
	if c.logger == nil {
		c.logger = &NopLogger{}
	}

	c.viper = viper.New()

	if err := c.initializeEnv(); err != nil {
		return fmt.Errorf("initialize env: %w", err)
	}

	// 绑定命令行参数（排除Cobra默认参数）
	for _, flags := range c.pflag {
		// 获取所有注册的flags
		flags.VisitAll(func(f *pflag.Flag) {
			// 排除不需要绑定的参数
			if !c.isExcludedFlag(f.Name) {
				// 只绑定非排除参数
				if err := c.viper.BindPFlag(f.Name, f); err != nil {
					c.logger.Errorf("Failed to bind flag %s: %v", f.Name, err)
				}
			}
		})
	}

	if c.path != "" {
		if err := c.validatePath(); err != nil {
			return fmt.Errorf("validate path: %w", err)
		}
		c.viper.AddConfigPath(c.path)
	}

	if err := c.validateMode(); err != nil {
		return fmt.Errorf("validate mode: %w", err)
	}

	if c.mode != "" {
		c.viper.SetConfigType(c.mode)
	}

	if c.name != "" {
		c.viper.SetConfigName(c.name)
	}

	if err := c.loadOrCreateConfig(); err != nil {
		return err
	}

	return nil
}

func (c *Config) initializeEnv() error {
	if !c.envOptions.Enabled {
		return nil
	}

	if c.envOptions.Prefix != "" {
		prefix := strings.ToUpper(c.envOptions.Prefix)
		c.viper.SetEnvPrefix(prefix)
	}

	c.viper.SetEnvKeyReplacer(strings.NewReplacer(
		".", "_",
	))

	c.viper.AutomaticEnv()
	return nil
}

func (c *Config) loadOrCreateConfig() error {
	// 尝试加载配置
	err := c.viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			c.logger.Infof("Config file not found, creating default config")
			// 配置文件不存在，创建默认配置
			if err := c.createDefaultConfig(); err != nil {
				return fmt.Errorf("create default config: %w", err)
			}
			return nil
		}
		c.logger.Errorf("Failed to read config file: %v", err)
		return fmt.Errorf("read config: %w", err)
	}
	
	c.logger.Infof("Successfully loaded config file: %s", c.viper.ConfigFileUsed())
	return nil
}

func (c *Config) validateMode() error {
	if c.mode == "" {
		c.mode = "yaml" // 默认为yaml
		c.logger.Debugf("Config mode not specified, using default mode: yaml")
		return nil
	}

	// 检查是否是支持的文件类型
	for _, ext := range viper.SupportedExts {
		if ext == c.mode {
			return nil
		}
	}

	c.logger.Errorf("Unsupported config mode: %s (supported modes: %s)", 
		c.mode, strings.Join(viper.SupportedExts, ", "))
	return fmt.Errorf("unsupported config mode: %s (supported: %s)", 
		c.mode, strings.Join(viper.SupportedExts, ", "))
}

// validatePath 验证并规范化配置文件路径
func (c *Config) validatePath() error {
	// 处理空路径情况
	if c.path == "" {
		c.path = "."
		return nil
	}

	// 使用 Clean 清理路径，去除 .. 和多余的分隔符
	cleanPath := filepath.Clean(c.path)

	// 获取绝对路径
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// 规范化路径（处理符号链接）
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("evaluate symlinks: %w", err)
	}

	// 如果路径不存在，使用 absPath
	if os.IsNotExist(err) {
		c.path = absPath
	} else {
		c.path = realPath
	}

	// 检查目录权限和可写性
	if err := c.ensureDirectoryAccess(); err != nil {
		return err
	}

	return nil
}

// ensureDirectoryAccess 确保目录存在且可写
func (c *Config) ensureDirectoryAccess() error {
	// 检查目录状态
	info, err := os.Stat(c.path)
	if err == nil {
		// 目录存在，检查是否为目录
		if !info.IsDir() {
			return fmt.Errorf("path is not a directory: %s", c.path)
		}
	} else if os.IsNotExist(err) {
		// 目录不存在，尝试创建
		if err := os.MkdirAll(c.path, 0o755); err != nil {
			return fmt.Errorf("create directory failed: %w", err)
		}
		return nil
	} else if os.IsPermission(err) {
		return fmt.Errorf("permission denied: %w", err)
	} else {
		return fmt.Errorf("check path failed: %w", err)
	}

	// 使用临时文件测试目录可写性
	tempFile, err := os.CreateTemp(c.path, ".config_write_test")
	if err != nil {
		return fmt.Errorf("directory not writable: %w", err)
	}

	// 清理临时文件
	tempName := tempFile.Name()
	_ = tempFile.Close()
	_ = os.Remove(tempName)

	return nil
}

// isExcludedFlag 判断标志是否应该被排除在viper绑定之外
func (c *Config) isExcludedFlag(flagName string) bool {
	// 首先检查用户自定义排除标志
	if c.excludedFlags[flagName] {
		return true
	}
	// 然后检查默认排除标志
	return c.defaultExcludedFlags[flagName]
}
