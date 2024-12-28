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
	"github.com/spf13/viper"
)

const (
	ExitConfigError = 78 // EX_CONFIG in sysexits.h
)

var (
	workPathOnce  sync.Once
	workPathValue string
	globalOnce    sync.Once
	globalConfig  *Config

	ErrInvalidKey = errors.New("invalid configuration key")
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

// WorkPath 获取工作目录，出错时返回当前目录
func WorkPath(parts ...string) string {
	workPathOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			workPathValue = "."
			return
		}

		dir, err := filepath.EvalSymlinks(filepath.Dir(exe))
		if err != nil {
			workPathValue = "."
			return
		}

		workPathValue = dir
	})

	return filepath.Join(append([]string{workPathValue}, parts...)...)
}

// New 创建新的配置实例
func New(opts ...Option) (*Config, error) {
	// 创建默认配置
	c := &Config{
		viper: viper.New(),
		path:  WorkPath("."),
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
		config, err := New(opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error initializing global config: %v", err)
			os.Exit(ExitConfigError)
		}
		globalConfig = config
	})
	return globalConfig
}

// Register 注册配置项到全局配置
func Register(module, key string, value interface{}) error {
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
func (c *Config) Get(key string, def ...interface{}) interface{} {
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
func (c *Config) GetStringMap(key string) map[string]interface{} {
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
func (c *Config) Set(key string, value interface{}) error {
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
func (c *Config) Unmarshal(obj interface{}, key ...string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 如果是结构体指针，则设置默认值
	if reflect.TypeOf(obj).Kind() == reflect.Ptr &&
		reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
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
		TagName:          "config",
		ZeroFields:       false,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("create decoder: %w", err)
	}

	// 获取配置数据
	var data map[string]interface{}
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
	if reflect.TypeOf(obj).Kind() == reflect.Ptr &&
		reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
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
		prefix := strings.ToUpper(sanitizeEnvPrefix(c.envOptions.Prefix))
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

	supportedModes := map[string]bool{
		"yaml":       true,
		"yml":        true,
		"json":       true,
		"toml":       true,
		"ini":        true,
		"env":        true,
		"properties": true,
	}

	if !supportedModes[c.mode] {
		return fmt.Errorf("unsupported config mode: %s", c.mode)
	}

	return nil
}

func (c *Config) validatePath() error {
	if c.path == "" {
		c.path = "."
		return nil
	}

	if strings.ContainsAny(c.path, "\x00") {
		return fmt.Errorf("path contains illegal characters")
	}

	absPath, err := filepath.Abs(c.path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	c.path = absPath

	info, err := os.Stat(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(c.path, 0o755); err != nil {
				return fmt.Errorf("create directory failed: %w", err)
			}
			return nil
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: %w", err)
		}
		return fmt.Errorf("check path failed: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", c.path)
	}

	testFile := filepath.Join(c.path, ".test")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("directory not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)

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

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "1", "t", "true", "yes", "y", "on":
		return true, nil
	case "0", "f", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}

func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func parseSlice(s string, t reflect.Type) (reflect.Value, error) {
	// 处理空字符串情况
	if s == "" {
		return reflect.MakeSlice(t, 0, 0), nil
	}

	// 尝试解析 JSON 数组
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var slice []interface{}
		if err := json.Unmarshal([]byte(s), &slice); err == nil {
			return convertJSONArrayToSlice(slice, t)
		}
	}

	// 处理逗号分隔的字符串
	return parseCommaDelimitedSlice(s, t)
}

// 处理逗号分隔的字符串
func parseCommaDelimitedSlice(s string, t reflect.Type) (reflect.Value, error) {
	parts := strings.Split(s, ",")
	slice := reflect.MakeSlice(t, len(parts), len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		val := reflect.New(t.Elem()).Elem()

		if err := setSliceElement(val, part, t.Elem().Kind()); err != nil {
			return reflect.Value{}, fmt.Errorf("parse slice element %d: %w", i, err)
		}
		slice.Index(i).Set(val)
	}

	return slice, nil
}

// 设置切片元素的值
func setSliceElement(val reflect.Value, part string, kind reflect.Kind) error {
	switch kind {
	case reflect.String:
		val.SetString(part)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setIntElement(val, part)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setUintElement(val, part)
	case reflect.Float32, reflect.Float64:
		return setFloatElement(val, part)
	case reflect.Bool:
		return setBoolElement(val, part)
	default:
		return fmt.Errorf("unsupported slice element type: %s", kind)
	}
}

// 设置整数类型元素
func setIntElement(val reflect.Value, part string) error {
	n, err := parseInt(part)
	if err != nil {
		return err
	}
	val.SetInt(n)
	return nil
}

// 设置无符号整数类型元素
func setUintElement(val reflect.Value, part string) error {
	n, err := parseUint(part)
	if err != nil {
		return err
	}
	val.SetUint(n)
	return nil
}

// 设置浮点数类型元素
func setFloatElement(val reflect.Value, part string) error {
	n, err := parseFloat(part)
	if err != nil {
		return err
	}
	val.SetFloat(n)
	return nil
}

// 设置布尔类型元素
func setBoolElement(val reflect.Value, part string) error {
	b, err := parseBool(part)
	if err != nil {
		return err
	}
	val.SetBool(b)
	return nil
}

func setDefaultValues(obj interface{}) error {
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
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(v)
		} else if d, err := time.ParseDuration(value); err == nil {
			field.SetInt(int64(d))
		} else {
			return fmt.Errorf("invalid int value: %s", value)
		}
	case reflect.Float32, reflect.Float64:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			field.SetFloat(v)
		} else {
			return fmt.Errorf("invalid float value: %s", value)
		}
	case reflect.Bool:
		if v, err := parseBool(value); err == nil {
			field.SetBool(v)
		} else {
			return fmt.Errorf("invalid bool value: %s", value)
		}
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

func validateStruct(obj interface{}) error {
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

func stringToSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t.Kind() != reflect.Slice {
			return data, nil
		}

		str := data.(string)
		if str == "" {
			return []string{}, nil
		}

		// 尝试解析JSON数组
		var slice []interface{}
		if err := json.Unmarshal([]byte(str), &slice); err == nil {
			return slice, nil
		}

		// 降级为以逗号分隔的字符串
		return strings.Split(str, ","), nil
	}
}

func stringToMapHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t.Kind() != reflect.Map {
			return data, nil
		}

		str := data.(string)
		if str == "" {
			return map[string]interface{}{}, nil
		}

		// 尝试解析JSON对象
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(str), &m); err != nil {
			return nil, fmt.Errorf("invalid map format: %s", str)
		}
		return m, nil
	}
}

func convertJSONArrayToSlice(arr []interface{}, t reflect.Type) (reflect.Value, error) {
	slice := reflect.MakeSlice(t, len(arr), len(arr))

	for i, item := range arr {
		val := reflect.New(t.Elem()).Elem()

		switch t.Elem().Kind() {
		case reflect.String:
			s, ok := item.(string)
			if !ok {
				return reflect.Value{}, fmt.Errorf("element %d is not a string", i)
			}
			val.SetString(s)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := toInt64(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := toUint64(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetUint(n)
		case reflect.Float32, reflect.Float64:
			n, err := toFloat64(item)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}
			val.SetFloat(n)
		case reflect.Bool:
			b, ok := item.(bool)
			if !ok {
				return reflect.Value{}, fmt.Errorf("element %d is not a boolean", i)
			}
			val.SetBool(b)
		default:
			return reflect.Value{}, fmt.Errorf("unsupported slice element type: %s", t.Elem().Kind())
		}

		slice.Index(i).Set(val)
	}

	return slice, nil
}

func toInt64(v interface{}) (int64, error) {
	switch v := v.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func toUint64(v interface{}) (uint64, error) {
	switch v := v.(type) {
	case uint:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("negative value %f cannot be converted to uint64", v)
		}
		return uint64(v), nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to uint64", v)
	}
}

func toFloat64(v interface{}) (float64, error) {
	switch v := v.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// sanitizeEnvPrefix 清理环境变量前缀，转换为符合 POSIX 标准的环境变量格式
// 规则：
// 1. 只允许大写字母、数字和下划线
// 2. 必须以字母开头
// 3. 自动转换为大写
// 4. 特殊字符转换为下划线
func sanitizeEnvPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}

	var b strings.Builder
	var lastValid bool

	// 跳过开头的非字母字符，确保以字母开头
	for i := 0; i < len(prefix) && !isLetter(prefix[i]); i++ {
		prefix = prefix[1:]
	}

	// 如果没有字母开头，返回空字符串
	if prefix == "" || !isLetter(prefix[0]) {
		return ""
	}

	// 处理剩余字符，保证只包含合法字符
	for i := 0; i < len(prefix); i++ {
		c := prefix[i]
		if isAlphanumeric(c) {
			if !lastValid && b.Len() > 0 {
				b.WriteByte('_')
			}
			b.WriteByte(toUpper(c))
			lastValid = true
		} else {
			lastValid = false
		}
	}

	return b.String()
}

// isLetter 判断字符是否为字母
func isLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// isAlphanumeric 判断字符是否为字母或数字
func isAlphanumeric(c byte) bool {
	return isLetter(c) || (c >= '0' && c <= '9')
}

// toUpper 将字符转换为大写
func toUpper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 32
	}
	return c
}
