package sysconf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/darkit/sysconf/internal/path"
)

var (
	workPathOnce  sync.Once
	workPathValue string
	globalOnce    sync.Once
	globalConfig  *Config

	ErrInvalidKey       = errors.New("invalid configuration key")
	ErrInitGlobalConfig = errors.New("failed to initialize global config")
)

const (
	defaultWriteDelay        = 3 * time.Second
	defaultCacheWarmupDelay  = 10 * time.Millisecond
	defaultCacheRebuildDelay = 50 * time.Millisecond
)

// EnvOptions 环境变量配置选项
type EnvOptions struct {
	Prefix    string // 环境变量前缀
	Enabled   bool   // 是否启用环境变量
	SmartCase bool   // 支持多种大小写格式的环境变量
}

// 配置验证器接口
type ConfigValidator interface {
	// Validate 验证配置
	// config: 当前所有配置的map形式
	// 返回: 验证错误，如果验证通过则返回nil
	Validate(config map[string]any) error
	// GetName 获取验证器名称
	GetName() string
}

// 配置验证函数类型（用于简化接口实现）
type ConfigValidateFunc func(config map[string]any) error

// Validate 实现ConfigValidator接口
func (f ConfigValidateFunc) Validate(config map[string]any) error {
	return f(config)
}

// GetName 实现ConfigValidator接口
func (f ConfigValidateFunc) GetName() string {
	return "函数式验证器"
}

// Config 统一配置实现
type Config struct {
	// 核心数据存储 - 使用atomic.Value实现无锁读取
	data atomic.Value // 存储map[string]any

	// 并发控制
	mu sync.RWMutex // 保护元数据和写操作

	// 基本配置
	logger  Logger // 日志记录器
	path    string // 配置文件路径
	mode    string // 配置文件类型
	name    string // 配置文件名称
	content string // 默认配置文件内容

	// 功能组件
	envOptions    EnvOptions        // 环境变量配置选项
	envKeyCache   sync.Map          // 环境变量键派生缓存
	cryptoOptions CryptoOptions     // 加密配置选项
	crypto        ConfigCrypto      // 加密实现实例
	validators    []ConfigValidator // 配置验证器列表
	pflags        []*pflag.FlagSet  // 命令行标志绑定

	// 文件监控和写入控制
	lastUpdate    time.Time   // 配置最后更新时间
	writeTimer    *time.Timer // 延迟写入定时器
	pendingWrites bool        // 是否有待写入的更改
	writeDelay    time.Duration

	// viper兼容层（用于文件操作和环境变量）
	viper *viper.Viper

	// 高性能缓存 - 简化版本，无复杂版本控制
	cacheEnabled bool // 是否启用缓存
	// 缓存调度参数
	cacheWarmupDelay  time.Duration
	cacheRebuildDelay time.Duration

	// 兼容字段（保持与现有代码的兼容性）
	readCache    atomic.Value // 只读缓存，存储map[string]any
	cacheVersion int64        // 缓存版本号，用于检测是否需要更新
	cacheMu      sync.Mutex   // 缓存更新互斥锁
	writeMu      sync.Mutex   // 写入操作的互斥锁（来自setter.go）
}

// Option 配置选项
type Option func(*Config)

// New 创建新的统一配置实例
func New(opts ...Option) (*Config, error) {
	workPathOnce.Do(func() {
		workPathValue = WorkPath()
	})

	// 创建统一配置实例
	c := &Config{
		viper:             viper.New(),
		path:              workPathValue,
		mode:              "yaml",
		logger:            &NopLogger{}, // 默认空日志记录器
		cacheEnabled:      true,         // 默认启用缓存
		writeDelay:        defaultWriteDelay,
		cacheWarmupDelay:  defaultCacheWarmupDelay,
		cacheRebuildDelay: defaultCacheRebuildDelay,
	}

	// 初始化原子数据存储
	c.data.Store(make(map[string]any))

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
	c.WatchWithContext(context.Background(), callbacks...)
}

// WatchWithContext 监听配置变化并返回取消函数，用于显式停止监听。
func (c *Config) WatchWithContext(ctx context.Context, callbacks ...func()) context.CancelFunc {
	if ctx == nil {
		ctx = context.Background()
	}

	watchCtx, cancel := context.WithCancel(ctx)

	c.viper.OnConfigChange(func(e fsnotify.Event) {
		if e.Op&fsnotify.Write == 0 {
			return
		}

		select {
		case <-watchCtx.Done():
			return
		default:
		}

		c.mu.Lock()
		now := time.Now()
		if now.Sub(c.lastUpdate) < time.Second {
			c.mu.Unlock()
			return
		}
		c.lastUpdate = now

		if err := c.reloadConfigLocked(); err != nil {
			c.logger.Errorf("Failed to reload config after change: %v", err)
			c.mu.Unlock()
			return
		}
		c.syncFromViperUnsafe()
		c.mu.Unlock()

		c.invalidateCache()
		c.logger.Infof("Config file change detected: %s", e.Name)

		for _, cb := range callbacks {
			cb()
		}
		c.logger.Debugf("Executed %d config change callbacks", len(callbacks))
	})

	c.viper.WatchConfig()
	c.logger.Infof("Config file watching started")

	return cancel
}

// reloadConfigLocked 在检测到文件变更时重新加载配置文件
//
// 该方法要求调用方已经获得写锁，避免与其他写操作竞态。
func (c *Config) reloadConfigLocked() error {
	if c.name == "" {
		return nil
	}

	if c.cryptoOptions.Enabled {
		return c.readConfigFile()
	}

	return c.viper.ReadInConfig()
}

// Viper 返回底层的 viper 实例
func (c *Config) Viper() *viper.Viper {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper
}

// AddValidator 添加配置验证器
func (c *Config) AddValidator(validator ConfigValidator) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.validators = append(c.validators, validator)
}

// AddValidateFunc 添加配置验证函数（便利方法）
func (c *Config) AddValidateFunc(fn func(config map[string]any) error) {
	c.AddValidator(ConfigValidateFunc(fn))
}

// ClearValidators 清除所有验证器
func (c *Config) ClearValidators() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.validators = nil
}

// GetValidators 获取当前所有验证器（只读）
func (c *Config) GetValidators() []ConfigValidator {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本，避免外部修改
	validators := make([]ConfigValidator, len(c.validators))
	copy(validators, c.validators)
	return validators
}

func (c *Config) createDefaultConfig() error {
	if c.content == "" {
		return nil
	}

	// 支持纯内存配置：如果没有设置name，则不创建物理文件
	if c.name == "" {
		c.logger.Infof("Loading configuration in memory-only mode (no file name specified)")
		return c.loadContentToMemory()
	}

	// 有name时，创建物理文件（原有逻辑）
	configFile := filepath.Join(c.path, c.name+"."+c.mode)

	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		c.logger.Errorf("Failed to create config directory: %v", err)
		return fmt.Errorf("create config directory: %w", err)
	}

	// 检查目录写入权限
	if err := c.checkDirectoryPermissions(filepath.Dir(configFile)); err != nil {
		c.logger.Errorf("Config directory permission check failed: %v", err)
		return fmt.Errorf("config directory permission: %w", err)
	}

	c.logger.Infof("Creating default config file: %s", configFile)

	// 创建备份（如果文件已存在）
	if err := c.createBackupIfExists(configFile); err != nil {
		c.logger.Warnf("Failed to create backup: %v", err)
	}

	// 准备要写入的数据
	data := []byte(c.content)

	// 如果启用了加密，先加密数据
	if c.cryptoOptions.Enabled && c.crypto != nil {
		c.logger.Debugf("Encrypting default config content")
		encryptedData, err := c.crypto.Encrypt(data)
		if err != nil {
			c.logger.Errorf("Failed to encrypt default config: %v", err)
			return fmt.Errorf("encrypt default config: %w", err)
		}
		data = encryptedData
		c.logger.Infof("Default config content encrypted successfully")
	}

	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		c.logger.Errorf("Failed to write default config: %v", err)
		return fmt.Errorf("write default config: %w", err)
	}

	// 读取刚创建的配置文件
	if c.cryptoOptions.Enabled {
		// 如果启用了加密，使用自定义读取方法
		if err := c.readConfigFile(); err != nil {
			c.logger.Errorf("Failed to read new encrypted config: %v", err)
			return fmt.Errorf("read new encrypted config: %w", err)
		}
	} else {
		// 没有启用加密时，使用viper标准方法
		if err := c.viper.ReadInConfig(); err != nil {
			c.logger.Errorf("Failed to read new config: %v", err)
			return fmt.Errorf("read new config: %w", err)
		}
	}

	c.logger.Infof("Default config file created successfully")
	return nil
}

// 新增：将配置内容加载到内存中（不创建物理文件）
func (c *Config) loadContentToMemory() error {
	c.logger.Debugf("Loading config content to memory")

	// 使用bytes.NewReader创建一个读取器
	reader := strings.NewReader(c.content)

	// 设置配置类型，确保viper知道如何解析内容
	if c.mode != "" {
		c.viper.SetConfigType(c.mode)
	}

	// 从内存中读取配置
	if err := c.viper.ReadConfig(reader); err != nil {
		c.logger.Errorf("Failed to read config from memory: %v", err)
		return fmt.Errorf("read config from memory: %w", err)
	}

	c.logger.Infof("Configuration loaded successfully in memory-only mode")
	return nil
}

// checkDirectoryPermissions 检查目录权限
func (c *Config) checkDirectoryPermissions(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dir)
	}

	// 创建临时文件测试写入权限
	tempFile := filepath.Join(dir, ".write_test_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.WriteFile(tempFile, []byte("test"), 0o644); err != nil {
		return fmt.Errorf("no write permission in directory %s: %w", dir, err)
	}

	// 清理临时文件
	if err := os.Remove(tempFile); err != nil {
		c.logger.Warnf("Failed to clean up temp file %s: %v", tempFile, err)
	}

	return nil
}

// createBackupIfExists 如果配置文件存在则创建备份
func (c *Config) createBackupIfExists(configFile string) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil // 文件不存在，无需备份
	}

	backupFile := configFile + ".backup." + fmt.Sprintf("%d", time.Now().Unix())

	// 读取原文件
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read original config: %w", err)
	}

	// 写入备份文件
	if err := os.WriteFile(backupFile, data, 0o644); err != nil {
		return fmt.Errorf("write backup config: %w", err)
	}

	c.logger.Infof("Config backup created: %s", backupFile)
	return nil
}

func (c *Config) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pendingWrites = false
	c.envKeyCache = sync.Map{}
	if c.writeTimer != nil {
		c.writeTimer.Stop()
	}

	// 确保有一个日志记录器
	if c.logger == nil {
		c.logger = &NopLogger{}
	}

	c.viper = viper.New()

	if err := c.initializeEnv(); err != nil {
		return c.wrapError(err, "初始化环境变量")
	}

	// 绑定命令行参数
	for _, flagSet := range c.pflags {
		// 获取所有注册的flags
		flagSet.VisitAll(func(f *pflag.Flag) {
			if err := c.viper.BindPFlag(f.Name, f); err != nil {
				c.logger.Errorf("Failed to bind flag %s: %v", f.Name, err)
			}
		})
	}

	if c.path != "" {
		if err := c.validatePath(); err != nil {
			return c.wrapError(err, "验证配置文件路径")
		}
		c.viper.AddConfigPath(c.path)
	}

	if err := c.validateMode(); err != nil {
		return c.wrapError(err, "验证配置文件模式")
	}

	if c.mode != "" {
		c.viper.SetConfigType(c.mode)
	}

	if c.name != "" {
		c.viper.SetConfigName(c.name)
	}

	// 初始化加密配置
	if err := c.initializeCrypto(); err != nil {
		return c.wrapError(err, "初始化加密配置")
	}

	if err := c.loadOrCreateConfig(); err != nil {
		return err // loadOrCreateConfig 已经使用了 wrapError
	}

	// 同步viper数据到原子存储（已在锁内，直接调用内部方法）
	c.syncFromViperUnsafe()

	// 启用读取缓存以优化并发访问性能（保持兼容性）
	c.enableReadCache()

	return nil
}

func (c *Config) initializeEnv() error {
	if !c.envOptions.Enabled {
		return nil
	}

	// 设置环境变量前缀（自动转大写）
	if c.envOptions.Prefix != "" {
		prefix := strings.ToUpper(c.envOptions.Prefix)
		c.viper.SetEnvPrefix(prefix)
	}

	// 设置键名替换规则（点号转下划线）
	c.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 如果启用智能大小写匹配，设置自定义环境变量获取函数
	if c.envOptions.SmartCase {
		c.setupSmartCaseEnv()
	} else {
		// 使用标准的自动环境变量绑定（仅支持大写）
		c.viper.AutomaticEnv()
	}

	return nil
}

// 智能大小写环境变量处理
func (c *Config) setupSmartCaseEnv() {
	// 启用标准的自动环境变量绑定
	c.viper.AutomaticEnv()

	// 预处理环境变量，查找所有可能的配置键对应的环境变量变体
	c.bindSmartCaseEnvVars()
}

// 绑定智能大小写环境变量
func (c *Config) bindSmartCaseEnvVars() {
	startTime := time.Now()
	envVars := os.Environ()
	totalEnvs := len(envVars)

	prefix := ""
	if c.envOptions.Prefix != "" {
		prefix = strings.ToUpper(c.envOptions.Prefix) + "_"
	}

	// 性能优化：设置合理的处理阈值
	const (
		maxEnvsWithoutPrefix = 300                    // 无前缀时的最大环境变量数
		maxEnvsWithPrefix    = 1000                   // 有前缀时的最大环境变量数
		maxProcessingTime    = 100 * time.Millisecond // 最大处理时间
	)

	// 优化：如果环境变量过多，使用不同策略
	maxAllowed := maxEnvsWithoutPrefix
	if prefix != "" {
		maxAllowed = maxEnvsWithPrefix
	}

	if totalEnvs > maxAllowed {
		c.logger.Warnf("Large environment detected (%d vars), using optimized binding strategy", totalEnvs)
		if prefix == "" {
			c.logger.Infof("Consider setting an environment variable prefix using WithEnvPrefix() for better performance")
			// 无前缀时跳过智能绑定
			return
		}
	}

	// 预分配切片以提高性能
	matchingVars := make([]struct{ key, configKey string }, 0, min(totalEnvs, 100))

	// 第一阶段：快速筛选匹配的环境变量
	for _, env := range envVars {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			key := parts[0]

			// 如果设置了前缀，只处理匹配前缀的环境变量
			if prefix != "" {
				if strings.HasPrefix(strings.ToUpper(key), prefix) {
					// 移除前缀并转换为配置键格式
					configKey := strings.TrimPrefix(strings.ToUpper(key), prefix)
					configKey = strings.ToLower(strings.ReplaceAll(configKey, "_", "."))
					matchingVars = append(matchingVars, struct{ key, configKey string }{key, configKey})
				}
			} else if len(matchingVars) < maxEnvsWithoutPrefix {
				// 没有前缀时，限制处理数量
				configKey := strings.ToLower(strings.ReplaceAll(key, "_", "."))
				matchingVars = append(matchingVars, struct{ key, configKey string }{key, configKey})
			}
		}

		// 时间保护：如果处理时间过长，提前结束
		if time.Since(startTime) > maxProcessingTime {
			c.logger.Warnf("Environment processing timeout, processed %d vars", len(matchingVars))
			break
		}
	}

	// 第二阶段：批量绑定环境变量
	for _, pair := range matchingVars {
		c.viper.BindEnv(pair.configKey, pair.key)
		c.logger.Debugf("Bound env var: %s -> %s", pair.key, pair.configKey)
	}

	duration := time.Since(startTime)
	c.logger.Infof("Smart case env binding completed in %v, processed %d/%d environment variables",
		duration, len(matchingVars), totalEnvs)

	// 性能警告：如果处理时间过长，建议使用前缀
	if duration > maxProcessingTime && prefix == "" {
		c.logger.Warnf("Environment variable processing took %v, consider using WithEnvPrefix() for better performance", duration)
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 初始化加密配置
func (c *Config) initializeCrypto() error {
	if !c.cryptoOptions.Enabled {
		c.logger.Debugf("Encryption disabled")
		return nil
	}

	// 如果用户提供了自定义加密器，直接使用
	if c.cryptoOptions.Crypto != nil {
		c.crypto = c.cryptoOptions.Crypto
		c.logger.Infof("Using custom crypto implementation")
		return nil
	}

	// 使用默认的ChaCha20加密器
	defaultCrypto, err := NewDefaultCrypto(c.cryptoOptions.Key)
	if err != nil {
		c.logger.Errorf("Failed to create default crypto: %v", err)
		return fmt.Errorf("create default crypto: %w", err)
	}

	c.crypto = defaultCrypto
	c.logger.Infof("Encryption enabled with ChaCha20-Poly1305")

	// 如果没有提供密钥且生成了随机密钥，记录警告
	if c.cryptoOptions.Key == "" {
		c.logger.Warnf("Using auto-generated encryption key: %s", defaultCrypto.GetKey())
		c.logger.Warnf("Please save this key securely! Without it, encrypted config cannot be decrypted!")
	}

	return nil
}

func (c *Config) loadOrCreateConfig() error {
	// 纯内存配置模式：如果没有设置name，直接创建默认配置到内存
	if c.name == "" {
		c.logger.Infof("Memory-only mode: skipping file operations")
		if err := c.createDefaultConfig(); err != nil {
			return c.wrapError(err, "创建内存配置")
		}
		return nil
	}

	// 如果启用了加密，使用自定义的读取方法
	if c.cryptoOptions.Enabled {
		err := c.readConfigFile()
		if err != nil {
			if os.IsNotExist(err) {
				c.logger.Infof("Config file not found, creating default config")
				// 配置文件不存在，创建默认配置
				if err := c.createDefaultConfig(); err != nil {
					return c.wrapError(err, "创建默认加密配置")
				}
				return nil
			}
			c.logger.Errorf("Failed to read encrypted config file: %v", err)
			return c.wrapError(err, "读取加密配置文件")
		}
		c.logger.Infof("Successfully loaded encrypted config file")
		return nil
	}

	// 没有启用加密时，使用viper的标准读取方法
	err := c.viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			c.logger.Infof("Config file not found, creating default config")
			// 配置文件不存在，创建默认配置
			if err := c.createDefaultConfig(); err != nil {
				return c.wrapError(err, "创建默认配置")
			}
			return nil
		}
		c.logger.Errorf("Failed to read config file: %v", err)
		return c.wrapError(err, "读取配置文件")
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

// WorkPath 获取工作目录
// - 空参数或 "." 返回当前工作目录
// - 绝对路径直接返回
// - 以 "~" 开头的路径展开为用户主目录
// - 相对路径基于当前工作目录展开
func WorkPath(parts ...string) string {
	resolver := path.NewResolver()
	return resolver.Resolve(parts...)
}

// reinitialize 重新初始化配置（用于避免锁冲突）
func (c *Config) reinitialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 重新绑定环境变量
	if err := c.initializeEnv(); err != nil {
		return fmt.Errorf("reinitialize env: %w", err)
	}

	// 同步环境变量和viper数据到原子存储
	c.syncFromViperUnsafe()

	return nil
}

// ============================================================================
// 统一数据存储和访问方法 - 基于atomic.Value的高性能并发安全架构
// ============================================================================

// loadData 原子性加载当前配置数据
func (c *Config) loadData() map[string]any {
	if data := c.data.Load(); data != nil {
		return data.(map[string]any)
	}
	return make(map[string]any)
}

// storeData 原子性存储配置数据（创建副本以确保线程安全）
func (c *Config) storeData(newData map[string]any) {
	// 创建深拷贝以确保数据完整性
	dataCopy := make(map[string]any)
	for k, v := range newData {
		dataCopy[k] = v
	}
	c.data.Store(dataCopy)
}

// syncFromViperUnsafe 从viper同步数据到原子存储（不加锁，用于已在锁内的场景）
func (c *Config) syncFromViperUnsafe() {
	// 从viper获取所有数据并进行扁平化处理
	viperData := c.viper.AllSettings()
	flatData := make(map[string]any)

	// 将嵌套数据扁平化，例如 app.name, database.host 等
	c.flattenViperData("", viperData, flatData)

	// 原子性存储
	c.storeData(flatData)
}

// flattenViperData 递归扁平化viper数据
func (c *Config) flattenViperData(prefix string, data map[string]any, result map[string]any) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		// 如果是map，递归处理
		if nestedMap, ok := value.(map[string]any); ok {
			c.flattenViperData(fullKey, nestedMap, result)
		} else {
			result[fullKey] = value
		}
	}
}

// getRaw 无锁读取原始配置值
func (c *Config) getRaw(key string) (any, bool) {
	data := c.loadData()

	// 首先尝试直接匹配
	if value, exists := data[key]; exists {
		return value, true
	}

	// 处理嵌套键查找
	if strings.Contains(key, ".") {
		return c.getNestedValueFromData(data, key)
	}

	// 尝试重构嵌套对象（用于向后兼容）
	if value, found := c.reconstructNestedValue(data, key); found {
		return value, true
	}

	// 回退到 viper 与环境变量查询，确保环境值立即可见
	return c.fetchFromViperOrEnv(key)
}

// fetchFromViperOrEnv 从 viper 或环境变量中查找配置值
func (c *Config) fetchFromViperOrEnv(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.viper != nil {
		if c.viper.IsSet(key) || c.viper.InConfig(key) {
			return c.viper.Get(key), true
		}
	}

	envOptions := c.envOptions
	if envOptions.Enabled {
		envKeys := c.deriveEnvKeys(envOptions, key)
		for _, envKey := range envKeys {
			if val, ok := os.LookupEnv(envKey); ok {
				return val, true
			}
		}
	}

	if c.viper != nil {
		if val := c.viper.Get(key); hasConcreteValue(val) {
			return val, true
		}
	}

	return nil, false
}

// deriveEnvKeys 生成可能的环境变量键名
func (c *Config) deriveEnvKeys(opts EnvOptions, key string) []string {
	if key == "" {
		return nil
	}

	sanitized := strings.ReplaceAll(key, ".", "_")
	if sanitized == "" {
		return nil
	}

	cacheKey := fmt.Sprintf("%s|%t|%s", opts.Prefix, opts.SmartCase, sanitized)
	if cached, ok := c.envKeyCache.Load(cacheKey); ok {
		stored := cached.([]string)
		return append([]string(nil), stored...)
	}

	baseVariants := map[string]struct{}{
		strings.ToUpper(sanitized): {},
		strings.ToLower(sanitized): {},
	}

	if opts.SmartCase {
		baseVariants[titleCaseEnv(sanitized)] = struct{}{}
	}

	prefixVariants := map[string]struct{}{"": {}}
	if opts.Prefix != "" {
		prefixVariants[strings.ToUpper(opts.Prefix)] = struct{}{}
		if opts.SmartCase {
			prefixVariants[strings.ToLower(opts.Prefix)] = struct{}{}
			prefixVariants[titleCaseEnv(strings.ToLower(opts.Prefix))] = struct{}{}
		}
	}

	result := make([]string, 0, len(baseVariants)*len(prefixVariants))
	for prefix := range prefixVariants {
		for base := range baseVariants {
			if prefix == "" {
				result = append(result, base)
				continue
			}
			result = append(result, prefix+"_"+base)
		}
	}

	c.envKeyCache.Store(cacheKey, append([]string(nil), result...))
	return result
}

// titleCaseEnv 将下划线分隔的字符串转换为首字母大写形式
func titleCaseEnv(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		if len(lower) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, "_")
}

// hasConcreteValue 判断值是否包含有效数据
func hasConcreteValue(val any) bool {
	if val == nil {
		return false
	}

	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() > 0
	case reflect.Bool:
		// 布尔值即使为false也代表显式设置
		return true
	default:
		return !rv.IsZero()
	}
}

// getNestedValueFromData 处理嵌套键值查找（避免与cache.go中的方法冲突）
func (c *Config) getNestedValueFromData(data map[string]any, key string) (any, bool) {
	// 按点号分割键
	parts := strings.Split(key, ".")
	current := data

	// 构建完整路径进行查找
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一级，直接查找值
			if value, exists := current[part]; exists {
				return value, true
			}
			// 也尝试完整的键路径
			if value, exists := data[key]; exists {
				return value, true
			}
			return nil, false
		}

		// 中间级，需要是map
		if nextLevel, exists := current[part]; exists {
			if nextMap, ok := nextLevel.(map[string]any); ok {
				current = nextMap
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}

	return nil, false
}

// reconstructNestedValue 从扁平化数据重构嵌套对象（用于向后兼容）
func (c *Config) reconstructNestedValue(data map[string]any, key string) (any, bool) {
	// 查找以该键为前缀的所有扁平化键
	prefix := key + "."
	nested := make(map[string]any)
	found := false

	for k, v := range data {
		if strings.HasPrefix(k, prefix) {
			// 移除前缀，获取相对路径
			subKey := strings.TrimPrefix(k, prefix)
			// 在嵌套map中设置值
			c.setNestedValue(nested, subKey, v)
			found = true
		}
	}

	if found {
		return nested, true
	}
	return nil, false
}

// reconstructNestedStructure 从扁平化数据重构完整的嵌套结构
func (c *Config) reconstructNestedStructure(flatData map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range flatData {
		c.setNestedValue(result, key, value)
	}

	return result
}

// setNestedValue 在嵌套map中设置值
func (c *Config) setNestedValue(m map[string]any, key string, value any) {
	if !strings.Contains(key, ".") {
		m[key] = value
		return
	}

	parts := strings.Split(key, ".")
	current := m

	// 创建嵌套结构
	for _, part := range parts[:len(parts)-1] {
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]any)
		}
		if nextMap, ok := current[part].(map[string]any); ok {
			current = nextMap
		} else {
			// 如果类型不匹配，创建新的map
			current[part] = make(map[string]any)
			current = current[part].(map[string]any)
		}
	}

	// 设置最终值
	current[parts[len(parts)-1]] = value
}

// setRaw 原子性设置配置值
func (c *Config) setRaw(key string, value any) error {
	if key == "" {
		return ErrInvalidKey
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 获取当前数据的副本
	currentData := c.loadData()
	newData := make(map[string]any, len(currentData)+1)

	// 移除与当前键相关的旧数据，避免残留脏数据
	prefix := key + "."
	for k, v := range currentData {
		if k == key || strings.HasPrefix(k, prefix) {
			continue
		}
		newData[k] = v
	}

	// 设置新值并自动展开嵌套结构
	c.mergeValueIntoData(newData, key, value)

	// 原子性存储
	c.storeData(newData)

	// 同时同步到viper，确保嵌套查询正常工作
	c.viper.Set(key, value)

	c.logger.Debugf("Set config value: %s", key)
	return nil
}

// mergeValueIntoData 将值写入扁平化数据结构
func (c *Config) mergeValueIntoData(target map[string]any, key string, value any) {
	sanitized := sanitizeValue(value)
	c.mergeSanitizedValue(target, key, sanitized)
}

func (c *Config) mergeSanitizedValue(target map[string]any, key string, sanitized any) {
	switch typed := sanitized.(type) {
	case map[string]any:
		target[key] = typed
		for childKey, childVal := range typed {
			c.mergeSanitizedValue(target, key+"."+childKey, childVal)
		}
	default:
		target[key] = sanitized
	}
}

// sanitizeValue 深拷贝并规范化传入值，确保内部存储不受外部引用影响。
func sanitizeValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		copied := make(map[string]any, len(v))
		for k, val := range v {
			copied[k] = sanitizeValue(val)
		}
		return copied
	case map[any]any:
		copied := make(map[string]any, len(v))
		for rawKey, val := range v {
			strKey, ok := rawKey.(string)
			if !ok {
				continue
			}
			copied[strKey] = sanitizeValue(val)
		}
		return copied
	case []any:
		copied := make([]any, len(v))
		for i, val := range v {
			copied[i] = sanitizeValue(val)
		}
		return copied
	case []string:
		copied := make([]string, len(v))
		copy(copied, v)
		return copied
	case []int:
		copied := make([]int, len(v))
		copy(copied, v)
		return copied
	case []float64:
		copied := make([]float64, len(v))
		copy(copied, v)
		return copied
	case []bool:
		copied := make([]bool, len(v))
		copy(copied, v)
		return copied
	case []map[string]any:
		copied := make([]map[string]any, len(v))
		for i, val := range v {
			copied[i] = sanitizeValue(val).(map[string]any)
		}
		return copied
	case []map[any]any:
		copied := make([]map[string]any, len(v))
		for i, val := range v {
			copied[i] = sanitizeValue(val).(map[string]any)
		}
		return copied
	default:
		return value
	}
}

// ============================================================================
// 配置查询和管理方法
// ============================================================================

// IsSet 检查配置键是否存在
func (c *Config) IsSet(key string) bool {
	_, exists := c.getRaw(key)
	return exists
}

// Keys 获取所有配置键
func (c *Config) Keys() []string {
	data := c.loadData()
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

// AllSettings 获取所有配置（返回副本以保证线程安全）
func (c *Config) AllSettings() map[string]any {
	data := c.loadData()
	result := make(map[string]any)
	for k, v := range data {
		result[k] = v
	}
	return result
}
