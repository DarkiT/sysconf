package sysconf

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cast"
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

// WorkPath 获取工作目录，如果不可写则使用用户主目录
func WorkPath(parts ...string) string {
	workPathOnce.Do(func() {
		// 1. 首先尝试可执行文件目录
		if exe, err := os.Executable(); err == nil {
			if dir, err := filepath.EvalSymlinks(filepath.Dir(exe)); err == nil {
				// 检查目录是否存在且可访问
				if info, err := os.Stat(dir); err == nil && info.IsDir() {
					// 检查目录是否可写
					testFile := filepath.Join(dir, ".write_test")
					if err := os.WriteFile(testFile, []byte("test"), 0o666); err == nil {
						_ = os.Remove(testFile)
						workPathValue = dir
						return
					}
				}
			}
		}

		// 2. 然后尝试用户主目录
		if homeDir, err := os.UserHomeDir(); err == nil {
			appDir := filepath.Join(homeDir, ".config")
			if err := os.MkdirAll(appDir, 0o755); err == nil {
				workPathValue = appDir
				return
			}
		}

		// 3. 最后使用当前目录
		workPathValue = "."
	})

	// 过滤和验证路径部分
	validParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}

	// 如果组合后的路径是绝对路径，直接返回
	if filepath.IsAbs(filepath.Join(validParts...)) {
		return filepath.Join(validParts...)
	}

	// 构建最终路径
	fullPath := filepath.Join(append([]string{workPathValue}, validParts...)...)

	// 对相对路径进行安全性检查
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(workPathValue)) {
		return workPathValue
	}

	return fullPath
}

// New 创建新的配置实例
func New(opts ...Option) (*Config, error) {
	// 创建默认配置
	c := &Config{
		viper: viper.New(),
		path:  WorkPath(),
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

// Get 获取配置值
func (c *Config) Get(key string, def ...any) any {
	if key == "" {
		if len(def) > 0 {
			return def[0]
		}
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	val := c.viper.Get(key)
	if val == nil && len(def) > 0 {
		return def[0]
	}
	return val
}

// GetBool 获取布尔值配置
func (c *Config) GetBool(parts ...string) bool {
	if len(parts) == 0 {
		return false
	}

	var defaultVal bool
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseBool(parts[len(parts)-1])
		}
	} else {
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseBool(parts[len(parts)-1])
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetBool(key)
}

// GetFloat 获取浮点数配置
func (c *Config) GetFloat(parts ...string) float64 {
	if len(parts) == 0 {
		return 0
	}

	var defaultVal float64
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseFloat(parts[len(parts)-1], 64)
		}
	} else {
		if len(parts) > 1 {
			defaultVal, _ = strconv.ParseFloat(parts[len(parts)-1], 64)
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetFloat64(key)
}

// GetInt 获取整数配置
func (c *Config) GetInt(parts ...string) int {
	if len(parts) == 0 {
		return 0
	}

	var defaultVal int
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal, _ = strconv.Atoi(parts[len(parts)-1])
		}
	} else {
		if len(parts) > 1 {
			defaultVal, _ = strconv.Atoi(parts[len(parts)-1])
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetInt(key)
}

// GetString 获取字符串配置
func (c *Config) GetString(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	var defaultVal string
	var key string

	if strings.Contains(parts[0], ".") {
		key = parts[0]
		if len(parts) > 1 {
			defaultVal = parts[len(parts)-1]
		}
	} else {
		if len(parts) > 1 {
			defaultVal = parts[len(parts)-1]
			key = strings.Join(parts[:len(parts)-1], ".")
		} else {
			key = parts[0]
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		return defaultVal
	}

	return c.viper.GetString(key)
}

// GetStringSlice 获取字符串切片配置
func (c *Config) GetStringSlice(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetStringSlice(key)
}

// GetIntSlice 获取整数切片配置
func (c *Config) GetIntSlice(key string) []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetIntSlice(key)
}

// GetStringMap 获取字符串映射配置
func (c *Config) GetStringMap(key string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetStringMap(key)
}

// GetStringMapString 获取字符串-字符串映射配置
func (c *Config) GetStringMapString(key string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetStringMapString(key)
}

// GetTime 获取时间配置
func (c *Config) GetTime(key string) time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetTime(key)
}

// GetDuration 获取时间间隔配置
func (c *Config) GetDuration(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetDuration(key)
}

// Set 设置配置值
func (c *Config) Set(key string, value any) error {
	if key == "" {
		return ErrInvalidKey
	}

	c.mu.Lock()
	c.viper.Set(key, value)
	c.mu.Unlock()

	// 如果配置文件名称不存在则不保存文件
	if c.name == "" {
		return nil
	}
	// 用独立的互斥锁处理写入操作
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// 标记有待写入的更改
	c.pendingWrites = true

	// 如果定时器已存在，重置它
	if c.writeTimer != nil {
		c.writeTimer.Stop()
	}

	// 创建新的延迟写入定时器
	c.writeTimer = time.AfterFunc(3*time.Second, func() {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()

		if !c.pendingWrites {
			return
		}

		if err := c.viper.WriteConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if errors.As(err, &configFileNotFoundError) {
				configFile := filepath.Join(c.path, c.name+"."+c.mode)
				_ = c.viper.WriteConfigAs(configFile)
			}
		}

		c.pendingWrites = false
	})

	return nil
}

// SetEnvPrefix 更新环境变量选项
func (c *Config) SetEnvPrefix(prefix string) error {
	c.mu.Lock()
	c.envOptions.Prefix = prefix
	c.envOptions.Enabled = prefix != "" // 如果有前缀就启用环境变量
	err := c.initializeEnv()
	c.mu.Unlock()
	return err
}

// Unmarshal 将配置解析到结构体
// key 为空时解析整个配置，否则解析指定的配置段
// 支持 default 标签设置默认值
// 支持 required 标签验证必填字段
func (c *Config) Unmarshal(obj any, key ...string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 如果是结构体指针，则设置默认值
	if reflect.TypeOf(obj).Kind() == reflect.Ptr && reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
		if err := setDefaultValues(obj); err != nil {
			return fmt.Errorf("set defaults: %w", err)
		}
	}

	// 创建解码器配置
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(time.RFC3339),
			stringToSliceHookFunc(),
			stringToMapHookFunc(),
		),
		Result:           obj,
		WeaklyTypedInput: true,
		TagName:          strings.Join([]string{"config", strings.Join(viper.SupportedExts, ", ")}, ","),
		ZeroFields:       false,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("create decoder: %w", err)
	}

	// 获取配置数据
	var data map[string]any
	if len(key) > 0 && key[0] != "" {
		configKey := strings.Join(key, ".")
		sub := c.viper.Sub(configKey)
		if sub != nil {
			data = sub.AllSettings()
		}
	} else {
		data = c.viper.AllSettings()
	}

	// 如果没有配置数据，保持默认值
	if len(data) == 0 {
		return nil
	}

	// 解码配置
	if err := decoder.Decode(data); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	// 如果是结构体指针，则验证必填字段
	if reflect.TypeOf(obj).Kind() == reflect.Ptr && reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
		if err := validateStruct(obj); err != nil {
			return fmt.Errorf("validate: %w", err)
		}
	}

	return nil
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

// 内部方法和辅助函数
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func parseSlice(s string, t reflect.Type) (reflect.Value, error) {
	// 处理空字符串情况
	if s == "" {
		return reflect.MakeSlice(t, 0, 0), nil
	}

	// 尝试解析 JSON 数组
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var slice []any
		if err := json.Unmarshal([]byte(s), &slice); err == nil {
			return convertJSONArrayToSlice(slice, t)
		}
	}

	// 处理逗号分隔的字符串
	parts := strings.Split(s, ",")
	slice := reflect.MakeSlice(t, len(parts), len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		val := reflect.New(t.Elem()).Elem()

		var err error
		switch t.Elem().Kind() {
		case reflect.String:
			val.SetString(part)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var n int64
			n, err = cast.ToInt64E(part)
			if err == nil {
				val.SetInt(n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var n uint64
			n, err = cast.ToUint64E(part)
			if err == nil {
				val.SetUint(n)
			}
		case reflect.Float32, reflect.Float64:
			var n float64
			n, err = cast.ToFloat64E(part)
			if err == nil {
				val.SetFloat(n)
			}
		case reflect.Bool:
			var b bool
			b, err = cast.ToBoolE(part)
			if err == nil {
				val.SetBool(b)
			}
		default:
			return reflect.Value{}, fmt.Errorf("unsupported slice element type: %s", t.Elem().Kind())
		}

		if err != nil {
			return reflect.Value{}, fmt.Errorf("parse slice element %d: %w", i, err)
		}
		slice.Index(i).Set(val)
	}

	return slice, nil
}

func stringToSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.Slice {
			return data, nil
		}

		str := data.(string)
		if str == "" {
			return []string{}, nil
		}

		// 尝试解析 JSON
		var slice []any
		if err := json.Unmarshal([]byte(str), &slice); err == nil {
			return slice, nil
		}

		// 降级为逗号分隔
		return strings.Split(str, ","), nil
	}
}

func stringToMapHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.Map {
			return data, nil
		}

		var m map[string]any

		str := data.(string)
		if str == "" {
			return m, nil
		}

		if err := json.Unmarshal([]byte(str), &m); err != nil {
			return nil, fmt.Errorf("invalid map format: %s", str)
		}
		return m, nil
	}
}

func convertJSONArrayToSlice(arr []any, t reflect.Type) (reflect.Value, error) {
	slice := reflect.MakeSlice(t, len(arr), len(arr))

	for i, item := range arr {
		val := reflect.New(t.Elem()).Elem()

		switch t.Elem().Kind() {
		case reflect.String:
			s, err := cast.ToStringE(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetString(s)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := cast.ToInt64E(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := cast.ToUint64E(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetUint(n)
		case reflect.Float32, reflect.Float64:
			n, err := cast.ToFloat64E(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetFloat(n)
		case reflect.Bool:
			b, err := cast.ToBoolE(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetBool(b)
		default:
			return reflect.Value{}, fmt.Errorf("unsupported slice element type: %s", t.Elem().Kind())
		}

		slice.Index(i).Set(val)
	}

	return slice, nil
}

func setDefaultValues(obj any) error {
	if obj == nil {
		return errors.New("nil pointer")
	}

	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr {
		return errors.New("not a pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return errors.New("not a struct")
	}

	return setDefaultValuesRecursive(val)
}

func setDefaultValuesRecursive(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		tag := typ.Field(i).Tag.Get("default")

		if field.Kind() == reflect.Struct {
			if err := setDefaultValuesRecursive(field); err != nil {
				return err
			}
			continue
		}

		if tag != "" && isZero(field) {
			if err := setFieldValue(field, tag); err != nil {
				return fmt.Errorf("set field %s: %w", typ.Field(i).Name, err)
			}
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 先尝试解析为 duration
		if d, err := time.ParseDuration(value); err == nil {
			field.SetInt(int64(d))
			return nil
		}
		// 再尝试解析为数字
		if v, err := cast.ToInt64E(value); err == nil {
			field.SetInt(v)
			return nil
		}
		return fmt.Errorf("invalid int value: %s", value)
	case reflect.Float32, reflect.Float64:
		v, err := cast.ToFloat64E(value)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(v)
	case reflect.Bool:
		v, err := cast.ToBoolE(value)
		if err != nil {
			return fmt.Errorf("invalid bool value: %s", value)
		}
		field.SetBool(v)
	case reflect.Slice:
		v, err := parseSlice(value, field.Type())
		if err != nil {
			return err
		}
		field.Set(v)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

func validateStruct(obj any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		required := t.Field(i).Tag.Get("required")
		if required == "true" {
			if isZero(field) {
				return fmt.Errorf("field %s is required", t.Field(i).Name)
			}
		}
	}

	return nil
}
