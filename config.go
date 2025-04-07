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
	"github.com/spf13/viper"
)

var (
	workPathOnce  sync.Once
	workPathValue string
	globalOnce    sync.Once
	globalConfig  *Config

	ErrInvalidKey       = errors.New("invalid configuration key")
	ErrInitGlobalConfig = errors.New("failed to initialize global config")
)

// EnvOptions 环境变量配置选项
type EnvOptions struct {
	Prefix  string // 环境变量前缀
	Enabled bool   // 是否启用环境变量
}

// Config 配置结构体
type Config struct {
	viper         *viper.Viper
	path          string       // 配置文件路径
	mode          string       // 配置文件类型
	name          string       // 配置文件名称
	content       string       // 默认配置文件内容
	envOptions    EnvOptions   // 环境变量配置选项
	lastUpdate    time.Time    // 配置最后更新时间
	writeTimer    *time.Timer  // 延迟写入定时器
	pendingWrites bool         // 是否有待写入的更改
	mu            sync.RWMutex // 读取操作的锁
	writeMu       sync.Mutex   // 写入操作的互斥锁
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
		viper: viper.New(),
		path:  workPathValue,
		mode:  "yaml",
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

		if len(c.viper.AllKeys()) > 0 {
			for _, cb := range callbacks {
				cb()
			}
		}
	})

	c.viper.WatchConfig()
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
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(configFile, []byte(c.content), 0o644); err != nil {
		return fmt.Errorf("write default config: %w", err)
	}

	if err := c.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read new config: %w", err)
	}

	return nil
}

func (c *Config) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pendingWrites = false
	if c.writeTimer != nil {
		c.writeTimer.Stop()
	}

	c.viper = viper.New()

	if err := c.initializeEnv(); err != nil {
		return fmt.Errorf("initialize env: %w", err)
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
	if err := c.viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("read config: %w", err)
		}

		if c.content != "" {
			if err := c.createDefaultConfig(); err != nil {
				return fmt.Errorf("create default config: %w", err)
			}
		}
	}

	return nil
}

func (c *Config) validateMode() error {
	if c.mode == "" {
		c.mode = "yaml"
		return nil
	}

	// 检查是否是支持的格式
	for _, ext := range viper.SupportedExts {
		if c.mode == ext {
			return nil
		}
	}

	// 如果不支持，返回错误
	return fmt.Errorf("unsupported config mode: %s (supported: %s)", c.mode, strings.Join(viper.SupportedExts, ", "))
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
