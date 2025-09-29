package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/darkit/sysconf"
	"github.com/darkit/sysconf/validation"
)

// 创建示例配置文件内容
const defaultConfig = `# 🛡️ Sysconf 完整配置示例，集成新验证器系统
app:
  name: "MyApp"
  version: "1.0.0"
  env: "development"

# 服务器配置 - 由 NewWebServerValidator 验证
server:
  host: localhost
  port: 8080
  timeout: 30s
  features:
    - http
    - grpc
    - websocket

# 数据库配置 - 由 NewDatabaseValidator 验证
database:
  host: localhost
  port: 5432
  username: postgres
  password: "demo"  # 演示用密码，生产环境请使用环境变量
  database: "myapp"
  timeout: 10s
  max_conns: 10
  type: "postgresql"  # 验证器支持的数据库类型
  options:
    sslmode: disable
    timezone: UTC

# Redis配置 - 由 NewRedisValidator 验证
redis:
  host: localhost
  port: 6379
  addresses:
    - localhost:6379
  password: ""
  db: 0
  timeout: 5s

# 日志配置 - 由 NewLogValidator 验证
logging:
  level: info
  format: json
  path: logs/app.log

# 邮件配置 - 由 NewEmailValidator 验证 (用于演示)
email:
  smtp:
    host: smtp.example.com
    port: 587
    username: user@example.com
    password: mailpass123
  from: noreply@example.com

# API配置 - 由 NewAPIValidator 验证 (用于演示)
api:
  base_url: https://api.example.com
  timeout: 30
  rate_limit:
    enabled: true
    requests_per_minute: 1000
  auth:
    api_key: sk-demo1234567890

# 监控指标配置 - 用于演示浮点数切片功能
metrics:
  thresholds: [0.7, 0.85, 0.95]  # 性能阈值

# 数据分析配置 - 用于演示浮点数切片功能  
analytics:
  weights: ["1.0", "2.5", "3.2"]  # 分析权重（字符串格式）
`

// AppConfig 应用完整配置结构体
type AppConfig struct {
	App struct {
		Name    string `config:"name" default:"MyApp" validate:"required,min=1"`                            // 应用名称：必填，有默认值作为后备
		Version string `config:"version" default:"1.0.0" validate:"required,semver"`                        // 版本号：必填，有默认值，需符合语义化版本
		Env     string `config:"env" default:"development" validate:"required,oneof=development test prod"` // 环境：必填，有默认值，限定枚举值
	} `config:"app"`

	Server struct {
		Host     string        `config:"host" default:"localhost" validate:"required,hostname_rfc1123"`    // 服务器主机：必填，有默认值
		Port     int           `config:"port" default:"8080" validate:"required,min=1,max=65535"`          // 端口：必填，有默认值，范围验证
		Timeout  time.Duration `config:"timeout" default:"30s" validate:"required"`                        // 超时：必填，有默认值
		Features []string      `config:"features" default:"[\"http\", \"grpc\"]" validate:"dive,required"` // 功能特性：默认启用基础功能
	} `config:"server"`

	Database struct {
		Host     string        `config:"host" default:"localhost" validate:"required,hostname_rfc1123"`              // 数据库主机：必填，有默认值
		Port     int           `config:"port" default:"5432" validate:"required,min=1,max=65535"`                    // 数据库端口：必填，有默认值
		Username string        `config:"username" default:"postgres" validate:"required,min=1"`                      // 用户名：必填，有默认值
		Password string        `config:"password" validate:"required,min=1"`                                         // 密码：必填，无默认值（敏感信息）
		Database string        `config:"database" default:"myapp" validate:"required,min=1"`                         // 数据库名：必填，有默认值
		Type     string        `config:"type" default:"postgresql" validate:"oneof=postgresql mysql sqlite mongodb"` // 🆕 数据库类型：有默认值，限定枚举
		MaxConns int           `config:"max_conns" default:"10" validate:"min=1,max=100"`                            // 最大连接数：有默认值，非必填，但有范围限制
		Timeout  time.Duration `config:"timeout" default:"10s" validate:"required"`                                  // 超时：必填，有默认值
		Options  struct {
			SSLMode  string `config:"sslmode" default:"disable" validate:"oneof=disable require"` // SSL模式：有默认值，限定枚举
			Timezone string `config:"timezone" default:"UTC" validate:"timezone"`                 // 时区：有默认值，格式验证
		} `config:"options"`
	} `config:"database"`

	Redis struct {
		Host      string        `config:"host" default:"localhost" validate:"required,hostname_rfc1123"`              // 🆕 Redis主机：必填，有默认值
		Port      int           `config:"port" default:"6379" validate:"required,min=1,max=65535"`                    // 🆕 Redis端口：必填，有默认值
		Addresses []string      `config:"addresses" default:"[\"localhost:6379\"]" validate:"required,dive,required"` // Redis地址：必填，有默认值
		Password  string        `config:"password" default:"" validate:""`                                            // Redis密码：可选
		DB        int           `config:"db" default:"0" validate:"min=0,max=15"`                                     // 数据库索引：有默认值，范围限制
		Timeout   time.Duration `config:"timeout" default:"5s" validate:"required"`                                   // 超时：必填，有默认值
	} `config:"redis"`

	Logging struct {
		Level  string `config:"level" default:"info" validate:"required,oneof=debug info warn error"` // 日志级别：必填，有默认值，枚举验证
		Format string `config:"format" default:"json" validate:"required,oneof=json text"`            // 日志格式：必填，有默认值，枚举验证
		Path   string `config:"path" default:"logs/app.log" validate:"required"`                      // 日志路径：必填，有默认值
	} `config:"logging"`

	// 🆕 新增邮件配置，由验证器系统验证
	Email struct {
		SMTP struct {
			Host     string `config:"host" validate:"required,hostname"`        // SMTP主机：必填，主机名验证
			Port     int    `config:"port" validate:"required,min=1,max=65535"` // SMTP端口：必填，端口范围验证
			Username string `config:"username" validate:"required,email"`       // 用户名：必填，邮箱格式验证
			Password string `config:"password" validate:"required,min=1"`       // 密码：必填，非空验证
		} `config:"smtp"`
		From string `config:"from" validate:"required,email"` // 发件人：必填，邮箱格式验证
	} `config:"email"`

	// 🆕 新增API配置，由验证器系统验证
	API struct {
		BaseURL   string `config:"base_url" validate:"required,url"`          // API基础URL：必填，URL格式验证
		Timeout   int    `config:"timeout" validate:"required,min=1,max=300"` // 超时时间：必填，范围验证
		RateLimit struct {
			Enabled           bool `config:"enabled"`                                        // 是否启用限流
			RequestsPerMinute int  `config:"requests_per_minute" validate:"min=1,max=10000"` // 每分钟请求数：范围验证
		} `config:"rate_limit"`
		Auth struct {
			APIKey string `config:"api_key" validate:"required,min=10"` // API密钥：必填，最小长度验证
		} `config:"auth"`
	} `config:"api"`
}

// App 定义应用结构
type App struct {
	config atomic.Value // 存储 *AppConfig
	cfg    *sysconf.Config
}

// NewApp 创建应用实例
func NewApp() (*App, error) {
	// 创建配置实例，集成新验证器系统
	cfg, err := sysconf.New(
		sysconf.WithPath("."),
		sysconf.WithMode("yaml"),
		sysconf.WithName("app"),
		sysconf.WithContent(defaultConfig), // 提供默认配置内容
		sysconf.WithEnvOptions(sysconf.EnvOptions{
			Prefix:  "APP",
			Enabled: true,
		}),
		sysconf.WithWriteFlushDelay(0),                   // 演示立即写入
		sysconf.WithCacheTiming(0, 100*time.Millisecond), // 演示缓存调优
		// 🆕 集成新验证器系统
		sysconf.WithValidators(
			validation.NewWebServerValidator(), // Web服务器配置验证
			validation.NewDatabaseValidator(),  // 数据库配置验证
			validation.NewRedisValidator(),     // Redis配置验证
			validation.NewLogValidator(),       // 日志配置验证
			validation.NewEmailValidator(),     // 邮件配置验证
			validation.NewAPIValidator(),       // API配置验证
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	app := &App{
		cfg: cfg,
	}

	// 加载初始配置
	if err := app.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 启动配置监听
	app.watchConfig()

	return app, nil
}

// loadConfig 加载配置
func (a *App) loadConfig() error {
	var cfg AppConfig
	if err := a.cfg.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	a.config.Store(&cfg)
	log.Printf("配置加载成功: %+v", cfg.App)
	return nil
}

// watchConfig 监听配置变化
func (a *App) watchConfig() {
	ctx, cancel := context.WithCancel(context.Background())
	log.Println("配置监听已启动，可随时调用 cancel() 终止监听")
	go func() {
		time.Sleep(12 * time.Second)
		log.Println("演示：12 秒后自动取消热重载监听")
		cancel()
	}()

	a.cfg.WatchWithContext(ctx, func() {
		log.Println("检测到配置文件变化，正在重新加载...")
		if err := a.loadConfig(); err != nil {
			log.Printf("重新加载配置失败: %v", err)
			return
		}
		log.Println("配置重新加载成功")
		a.onConfigChange()
	})
}

// onConfigChange 配置变化处理
func (a *App) onConfigChange() {
	cfg := a.config.Load().(*AppConfig)
	// 在这里处理配置变化
	log.Printf("服务器配置已更改: %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("数据库配置已更改: %s:%d", cfg.Database.Host, cfg.Database.Port)
}

// GetConfig 安全地获取当前配置
func (a *App) GetConfig() *AppConfig {
	return a.config.Load().(*AppConfig)
}

// DemoBasicUsage 演示基础配置使用
func (a *App) DemoBasicUsage() {
	log.Println("\n=== 基础配置访问演示 ===")

	// 1. 使用结构体方式访问配置
	cfg := a.GetConfig()
	log.Printf("应用名称: %s", cfg.App.Name)
	log.Printf("应用版本: %s", cfg.App.Version)
	log.Printf("服务器地址: %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("服务器功能: %v", cfg.Server.Features)

	// 2. 使用键值方式访问配置
	host := a.cfg.GetString("server.host")
	port := a.cfg.GetInt("server.port")
	timeout := a.cfg.GetDuration("server.timeout")
	log.Printf("服务器主机（通过键）: %s", host)
	log.Printf("服务器端口（通过键）: %d", port)
	log.Printf("服务器超时（通过键）: %v", timeout)

	// 3. 演示默认值功能
	unknownValue := a.cfg.GetString("unknown.key", "default-value")
	log.Printf("未知键（使用默认值）: %s", unknownValue)

	// 4. 演示不同数据类型的获取
	features := a.cfg.GetStringSlice("server.features")
	log.Printf("服务器功能列表: %v", features)

	options := a.cfg.GetStringMapString("database.options")
	log.Printf("数据库选项: %v", options)
}

// DemoAdvancedFeatures 演示高级功能
func (a *App) DemoAdvancedFeatures() {
	log.Println("\n=== 高级功能演示 ===")

	// 1. 演示子配置解析
	var serverConfig struct {
		Host     string        `config:"host"`
		Port     int           `config:"port"`
		Timeout  time.Duration `config:"timeout"`
		Features []string      `config:"features"`
	}

	if err := a.cfg.Unmarshal(&serverConfig, "server"); err != nil {
		log.Printf("解析服务器配置失败: %v", err)
	} else {
		log.Printf("服务器配置结构体: %+v", serverConfig)
	}

	// 2. 演示配置动态更新
	log.Println("动态更新配置...")
	originalPort := a.cfg.GetInt("server.port")
	log.Printf("原始端口: %d", originalPort)

	if err := a.cfg.Set("server.port", 9090); err != nil {
		log.Printf("设置端口失败: %v", err)
	} else {
		time.Sleep(100 * time.Millisecond) // 等待配置更新
		newPort := a.cfg.GetInt("server.port")
		log.Printf("更新后端口: %d", newPort)
	}

	// 3. 演示深拷贝：设置 map 后继续修改原始 map，验证不会污染内部存储
	log.Println("验证深拷贝行为...")
	originalMap := map[string]any{
		"grandchild": "原始值",
		"sibling":    "兄弟值",
	}
	if err := a.cfg.Set("parent.child", originalMap); err != nil {
		log.Printf("设置嵌套 map 失败: %v", err)
	} else {
		originalMap["grandchild"] = "被修改"
		log.Printf("外部修改 map 后的 GetString: %s", a.cfg.GetString("parent.child.grandchild"))
	}

	// 4. 演示配置验证（尝试设置无效值）
	log.Println("尝试设置无效配置...")
	if err := a.cfg.Set("server.port", "invalid"); err != nil {
		log.Printf("设置无效端口值被拒绝: %v", err)
	}
}

// DemoEnvironmentVariables 演示环境变量功能
func DemoEnvironmentVariables() {
	log.Println("\n=== 环境变量演示 ===")

	// 设置一些环境变量进行演示
	os.Setenv("APP_SERVER_HOST", "production-server")
	os.Setenv("APP_SERVER_PORT", "9000")
	os.Setenv("APP_DATABASE_PASSWORD", "super-secret")
	// 演示 SmartCase：混合大小写与不同前缀
	os.Setenv("app_server_timeout", "45s")
	os.Setenv("App_Server_Features", "[\"http\",\"metrics\"]")

	defer func() {
		// 清理环境变量
		os.Unsetenv("APP_SERVER_HOST")
		os.Unsetenv("APP_SERVER_PORT")
		os.Unsetenv("APP_DATABASE_PASSWORD")
		os.Unsetenv("app_server_timeout")
		os.Unsetenv("App_Server_Features")
	}()

	// 创建支持环境变量的配置
	cfg, err := sysconf.New(
		sysconf.WithPath("."),
		sysconf.WithMode("yaml"),
		sysconf.WithName("app"),
		sysconf.WithContent(defaultConfig),
		sysconf.WithEnvOptions(sysconf.EnvOptions{
			Prefix:  "APP",
			Enabled: true,
		}),
	)
	if err != nil {
		log.Printf("创建配置失败: %v", err)
		return
	}

	// 演示环境变量覆盖配置值
	host := cfg.GetString("server.host")           // 应该是 "production-server"
	port := cfg.GetInt("server.port")              // 应该是 9000
	password := cfg.GetString("database.password") // 应该是 "super-secret"
	timeout := cfg.GetDuration("server.timeout")
	features := cfg.GetStringSlice("server.features")

	log.Printf("环境变量覆盖 - 服务器主机: %s", host)
	log.Printf("环境变量覆盖 - 服务器端口: %d", port)
	log.Printf("环境变量覆盖 - 数据库密码: %s", maskPassword(password))
	log.Printf("SmartCase 覆盖 - 服务器超时: %v", timeout)
	log.Printf("SmartCase 覆盖 - 服务器功能: %v", features)

	// 再次读取，验证缓存命中
	log.Printf("再次读取验证缓存命中 - 服务器超时: %v", cfg.GetDuration("server.timeout"))
}

// DemoGlobalConfig 演示全局配置使用
func DemoGlobalConfig() {
	log.Println("\n=== 全局配置演示 ===")

	// 注册全局配置
	if err := sysconf.Register("global", "app_name", "GlobalApp"); err != nil {
		log.Printf("注册全局配置失败: %v", err)
		return
	}
	if err := sysconf.Register("global", "version", "2.0.0"); err != nil {
		log.Printf("注册全局配置失败: %v", err)
		return
	}

	// 获取全局配置
	appName := sysconf.Default().GetString("global.app_name")
	appVersion := sysconf.Default().GetString("global.version")

	log.Printf("全局应用名称: %s", appName)
	log.Printf("全局应用版本: %s", appVersion)
}

// DemoErrorHandling 演示错误处理
func (a *App) DemoErrorHandling() {
	log.Println("\n=== 错误处理演示 ===")

	// 1. 演示配置验证错误
	type InvalidConfig struct {
		RequiredField string `config:"required_field" required:"true"`
	}

	var invalidCfg InvalidConfig
	if err := a.cfg.Unmarshal(&invalidCfg, "nonexistent"); err != nil {
		log.Printf("预期的验证错误: %v", err)
	}

	// 2. 演示类型转换错误
	value, err := a.cfg.GetWithError("server.host")
	if err != nil {
		log.Printf("获取配置值错误: %v", err)
	} else {
		log.Printf("成功获取配置值: %v (类型: %T)", value, value)
	}

	// 3. 演示键不存在的情况
	nonExistentValue := a.cfg.GetString("non.existent.key", "fallback-value")
	log.Printf("不存在的键（使用回退值）: %s", nonExistentValue)
}

// maskPassword 遮盖密码显示
func maskPassword(password string) string {
	if password == "" {
		return "<空>"
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}

// DemoConfigValidation 演示配置验证功能
func (a *App) DemoConfigValidation() {
	fmt.Println("\n--- 配置验证演示 ---")

	cfg := a.cfg

	// 演示端口验证
	fmt.Println("尝试设置无效端口值...")
	if err := cfg.Set("server.port", "invalid"); err != nil {
		fmt.Printf("✅ 配置验证成功拦截无效值: %v\n", err)
	}

	if err := cfg.Set("server.port", 70000); err != nil {
		fmt.Printf("✅ 配置验证成功拦截超出范围的端口: %v\n", err)
	}

	// 演示正确的端口设置
	if err := cfg.Set("server.port", 9090); err == nil {
		fmt.Printf("✅ 成功设置有效端口: %d\n", cfg.GetInt("server.port"))
	}

	// 演示超时验证
	fmt.Println("尝试设置无效超时值...")
	if err := cfg.Set("server.timeout", "invalid_duration"); err != nil {
		fmt.Printf("✅ 配置验证成功拦截无效超时: %v\n", err)
	}

	// 🔧 重置为有效值，避免影响后续演示
	fmt.Println("重置为有效超时值...")
	if err := cfg.Set("server.timeout", "30s"); err != nil {
		fmt.Printf("❌ 重置超时值失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功重置超时值: %s\n", cfg.GetString("server.timeout"))
	}
}

// DemoFloatSlice 演示浮点数切片功能
func (a *App) DemoFloatSlice() {
	fmt.Println("\n--- 浮点数切片演示 ---")

	cfg := a.cfg

	// 首先检查配置文件是否已经包含这些值
	fmt.Println("🔍 检查配置文件中的预设值...")
	if val, err := cfg.GetWithError("metrics.thresholds"); err == nil {
		fmt.Printf("✅ 配置文件中的 metrics.thresholds: %v (类型: %T)\n", val, val)
	} else {
		fmt.Printf("❌ 配置文件中未找到 metrics.thresholds: %v\n", err)
	}

	if val, err := cfg.GetWithError("analytics.weights"); err == nil {
		fmt.Printf("✅ 配置文件中的 analytics.weights: %v (类型: %T)\n", val, val)
	} else {
		fmt.Printf("❌ 配置文件中未找到 analytics.weights: %v\n", err)
	}

	// 获取配置文件中的原始浮点数切片
	fmt.Println("\n📊 通过GetFloatSlice获取配置文件中的值...")
	originalThresholds := cfg.GetFloatSlice("metrics.thresholds")
	fmt.Printf("原始性能阈值: %v (长度: %d)\n", originalThresholds, len(originalThresholds))

	originalWeights := cfg.GetFloatSlice("analytics.weights")
	fmt.Printf("原始分析权重: %v (长度: %d)\n", originalWeights, len(originalWeights))

	// 动态更新一些浮点数切片数据
	fmt.Println("\n🔧 通过Set方法动态更新值...")
	cfg.Set("metrics.thresholds", []float64{0.8, 0.9, 0.95, 0.99})
	cfg.Set("analytics.weights", []string{"1.2", "3.5", "2.8"}) // 字符串形式的浮点数

	// 立即检查Set后的值
	fmt.Println("\n🔍 检查动态更新后的值...")
	if val, err := cfg.GetWithError("metrics.thresholds"); err == nil {
		fmt.Printf("更新后的 metrics.thresholds: %v (类型: %T)\n", val, val)
	} else {
		fmt.Printf("更新后未找到 metrics.thresholds: %v\n", err)
	}

	if val, err := cfg.GetWithError("analytics.weights"); err == nil {
		fmt.Printf("更新后的 analytics.weights: %v (类型: %T)\n", val, val)
	} else {
		fmt.Printf("更新后未找到 analytics.weights: %v\n", err)
	}

	// 获取更新后的浮点数切片
	fmt.Println("\n📊 通过GetFloatSlice获取更新后的值...")
	updatedThresholds := cfg.GetFloatSlice("metrics.thresholds")
	fmt.Printf("更新后性能阈值: %v (长度: %d)\n", updatedThresholds, len(updatedThresholds))

	updatedWeights := cfg.GetFloatSlice("analytics.weights")
	fmt.Printf("更新后分析权重: %v (长度: %d)\n", updatedWeights, len(updatedWeights))

	// 测试混合类型的切片转换
	fmt.Println("\n🔧 设置混合类型切片...")
	cfg.Set("mixed.values", []interface{}{1, "2.5", 3.7, "4"})

	if val, err := cfg.GetWithError("mixed.values"); err == nil {
		fmt.Printf("混合值原始数据: %v (类型: %T)\n", val, val)
	}

	mixed := cfg.GetFloatSlice("mixed.values")
	fmt.Printf("混合类型转换结果: %v (长度: %d)\n", mixed, len(mixed))

	// 演示从无到有的动态配置
	fmt.Println("\n🆕 演示全新配置键...")
	newKey := "performance.benchmarks"
	if val, err := cfg.GetWithError(newKey); err == nil {
		fmt.Printf("配置文件中的 %s: %v\n", newKey, val)
	} else {
		fmt.Printf("✅ 配置文件中未找到 %s（符合预期）\n", newKey)
	}

	cfg.Set(newKey, []float32{1.5, 2.3, 4.7, 8.1})
	benchmarks := cfg.GetFloatSlice(newKey)
	fmt.Printf("✅ 动态设置后的 %s: %v (长度: %d)\n", newKey, benchmarks, len(benchmarks))
}

// DemoValidationSystem 演示新验证器系统
func (a *App) DemoValidationSystem() {
	fmt.Println("\n=== 🛡️ 验证器系统全面演示 ===")

	cfg := a.cfg

	// 1. 显示当前已注册的验证器
	validators := cfg.GetValidators()
	fmt.Printf("✅ 当前已注册验证器数量: %d\n", len(validators))

	// 2. 演示预定义验证器的验证效果
	fmt.Println("\n--- 预定义验证器验证演示 ---")

	// 测试服务器配置验证
	fmt.Println("🔍 测试服务器配置验证:")
	testServerValidation(cfg)

	// 测试数据库配置验证
	fmt.Println("\n🔍 测试数据库配置验证:")
	testDatabaseValidation(cfg)

	// 测试Redis配置验证
	fmt.Println("\n🔍 测试Redis配置验证:")
	testRedisValidation(cfg)

	// 3. 演示自定义验证器
	fmt.Println("\n--- 自定义验证器演示 ---")
	demonstrateCustomValidators(cfg)

	// 4. 演示30+种验证规则
	fmt.Println("\n--- 30+种验证规则演示 ---")
	demonstrateValidationRules(cfg)

	// 5. 演示复合验证器
	fmt.Println("\n--- 复合验证器演示 ---")
	demonstrateCompositeValidator(cfg)

	// 6. 演示动态验证器管理
	fmt.Println("\n--- 动态验证器管理演示 ---")
	demonstrateDynamicValidators(cfg)
}

// testServerValidation 测试服务器配置验证
func testServerValidation(cfg *sysconf.Config) {
	// 测试有效端口
	if err := cfg.Set("server.port", 8080); err != nil {
		fmt.Printf("❌ 设置有效端口失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功设置有效端口: 8080\n")
	}

	// 测试无效端口
	if err := cfg.Set("server.port", 70000); err != nil {
		fmt.Printf("✅ 验证器成功拦截无效端口: %v\n", err)
	} else {
		fmt.Printf("❌ 验证器未能拦截无效端口\n")
	}

	// 测试主机名验证
	if err := cfg.Set("server.host", "valid-hostname"); err != nil {
		fmt.Printf("❌ 设置有效主机名失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功设置有效主机名: valid-hostname\n")
	}
}

// testDatabaseValidation 测试数据库配置验证
func testDatabaseValidation(cfg *sysconf.Config) {
	// 测试用户名验证
	if err := cfg.Set("database.username", "validuser"); err != nil {
		fmt.Printf("❌ 设置有效用户名失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功设置数据库用户名: validuser\n")
	}

	// 测试空用户名
	if err := cfg.Set("database.username", ""); err != nil {
		fmt.Printf("✅ 验证器成功拦截空用户名: %v\n", err)
	} else {
		fmt.Printf("❌ 验证器未能拦截空用户名\n")
	}

	// 测试数据库类型验证
	if err := cfg.Set("database.type", "postgresql"); err != nil {
		fmt.Printf("❌ 设置有效数据库类型失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功设置数据库类型: postgresql\n")
	}

	// 测试无效数据库类型
	if err := cfg.Set("database.type", "invaliddb"); err != nil {
		fmt.Printf("✅ 验证器成功拦截无效数据库类型: %v\n", err)
	} else {
		fmt.Printf("❌ 验证器未能拦截无效数据库类型\n")
	}
}

// testRedisValidation 测试Redis配置验证
func testRedisValidation(cfg *sysconf.Config) {
	// 测试Redis端口
	if err := cfg.Set("redis.port", 6379); err != nil {
		fmt.Printf("❌ 设置有效Redis端口失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功设置Redis端口: 6379\n")
	}

	// 测试Redis数据库索引
	if err := cfg.Set("redis.db", 5); err != nil {
		fmt.Printf("❌ 设置有效Redis DB失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功设置Redis DB: 5\n")
	}

	// 测试无效Redis数据库索引
	if err := cfg.Set("redis.db", 20); err != nil {
		fmt.Printf("✅ 验证器成功拦截无效Redis DB: %v\n", err)
	} else {
		fmt.Printf("❌ 验证器未能拦截无效Redis DB\n")
	}
}

// demonstrateCustomValidators 演示自定义验证器
func demonstrateCustomValidators(cfg *sysconf.Config) {
	// 创建业务逻辑验证器
	businessValidator := validation.NewRuleValidator("业务逻辑验证器")

	// 添加公司信息验证规则
	businessValidator.AddRule("company.name", validation.Required("公司名称不能为空"))
	businessValidator.AddRule("company.tax_id", validation.Pattern(`^\d{18}$`, "税务登记号必须是18位数字"))
	businessValidator.AddStringRule("company.industry", "enum:technology,finance,healthcare,education")
	businessValidator.AddStringRule("company.employee_count", "range:1,10000")

	// 添加自定义验证器
	cfg.AddValidator(businessValidator)
	fmt.Printf("✅ 添加自定义业务验证器\n")

	// 测试自定义验证规则
	cfg.Set("company.name", "创新科技有限公司")
	cfg.Set("company.tax_id", "123456789012345678")
	cfg.Set("company.industry", "technology")
	cfg.Set("company.employee_count", 150)
	fmt.Printf("✅ 设置公司配置: %s, 行业: %s, 员工: %d\n",
		cfg.GetString("company.name"),
		cfg.GetString("company.industry"),
		cfg.GetInt("company.employee_count"))

	// 测试无效的税务登记号
	if err := cfg.Set("company.tax_id", "invalid-tax-id"); err != nil {
		fmt.Printf("✅ 验证器成功拦截无效税务登记号: %v\n", err)
	}
}

// demonstrateValidationRules 演示30+种验证规则
func demonstrateValidationRules(cfg *sysconf.Config) {
	// 创建综合验证器展示各种规则
	comprehensiveValidator := validation.NewRuleValidator("综合验证规则演示")

	// 网络相关验证
	comprehensiveValidator.AddStringRule("network.email", "email")
	comprehensiveValidator.AddStringRule("network.url", "url")
	comprehensiveValidator.AddStringRule("network.ipv4", "ipv4")
	comprehensiveValidator.AddStringRule("network.ipv6", "ipv6")
	comprehensiveValidator.AddStringRule("network.hostname", "hostname")

	// 格式验证
	comprehensiveValidator.AddStringRule("format.uuid", "uuid")
	comprehensiveValidator.AddStringRule("format.json", "json")
	comprehensiveValidator.AddStringRule("format.base64", "base64")
	comprehensiveValidator.AddStringRule("format.phone", "phonenumber")
	comprehensiveValidator.AddStringRule("format.alphanum", "alphanum")

	// 数值验证
	comprehensiveValidator.AddStringRule("numbers.range_value", "range:1,100")
	comprehensiveValidator.AddStringRule("numbers.port", "port")

	cfg.AddValidator(comprehensiveValidator)
	fmt.Printf("✅ 添加综合验证规则演示器\n")

	// 测试各种验证规则
	testCases := map[string]interface{}{
		"network.email":       "admin@example.com",
		"network.url":         "https://www.example.com",
		"network.ipv4":        "192.168.1.1",
		"network.ipv6":        "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		"network.hostname":    "api.example.com",
		"format.uuid":         "123e4567-e89b-12d3-a456-426614174000",
		"format.json":         `{"key": "value"}`,
		"format.base64":       "SGVsbG8gV29ybGQ=",
		"format.phone":        "+1234567890",
		"format.alphanum":     "abc123",
		"numbers.range_value": 50,
		"numbers.port":        443,
	}

	fmt.Println("🔍 测试有效值:")
	for key, value := range testCases {
		if err := cfg.Set(key, value); err != nil {
			fmt.Printf("❌ 设置 %s 失败: %v\n", key, err)
		} else {
			fmt.Printf("✅ 成功设置 %s: %v\n", key, value)
		}
	}

	// 测试无效值
	fmt.Println("\n🔍 测试无效值:")
	invalidCases := map[string]interface{}{
		"network.email":       "invalid-email",
		"network.url":         "not-a-url",
		"network.ipv4":        "999.999.999.999",
		"format.uuid":         "not-a-uuid",
		"format.json":         "invalid json",
		"numbers.range_value": 150,
		"numbers.port":        70000,
	}

	for key, value := range invalidCases {
		if err := cfg.Set(key, value); err != nil {
			fmt.Printf("✅ 验证器成功拦截 %s 的无效值: %v\n", key, err)
		} else {
			fmt.Printf("❌ 验证器未能拦截 %s 的无效值\n", key)
		}
	}
}

// demonstrateCompositeValidator 演示复合验证器
func demonstrateCompositeValidator(cfg *sysconf.Config) {
	// 创建复合验证器
	composite := validation.NewCompositeValidator("企业级应用验证器",
		validation.NewDatabaseValidator(),
		validation.NewWebServerValidator(),
		validation.NewRedisValidator(),
		validation.NewEmailValidator(),
		validation.NewAPIValidator(),
	)

	// 添加复合验证器
	cfg.AddValidator(composite)
	fmt.Printf("✅ 添加复合验证器，包含 %d 个子验证器\n", len(composite.GetValidators()))
	fmt.Printf("✅ 复合验证器名称: %s\n", composite.GetName())

	// 设置邮件配置进行测试
	cfg.Set("email.smtp.host", "smtp.example.com")
	cfg.Set("email.smtp.port", 587)
	cfg.Set("email.smtp.username", "user@example.com")
	cfg.Set("email.smtp.password", "password123")
	cfg.Set("email.from", "noreply@example.com")
	fmt.Printf("✅ 成功设置邮件配置，复合验证器验证通过\n")

	// 设置API配置进行测试
	cfg.Set("api.base_url", "https://api.example.com")
	cfg.Set("api.timeout", 30)
	cfg.Set("api.rate_limit.enabled", true)
	cfg.Set("api.rate_limit.requests_per_minute", 1000)
	cfg.Set("api.auth.api_key", "sk-1234567890abcdef")
	fmt.Printf("✅ 成功设置API配置，复合验证器验证通过\n")
}

// demonstrateDynamicValidators 演示动态验证器管理
func demonstrateDynamicValidators(cfg *sysconf.Config) {
	initialCount := len(cfg.GetValidators())
	fmt.Printf("📊 初始验证器数量: %d\n", initialCount)

	// 动态添加临时验证器
	tempValidator := validation.NewRuleValidator("临时验证器")
	tempValidator.AddStringRule("temp.value", "required")
	tempValidator.AddStringRule("temp.number", "range:1,1000")

	cfg.AddValidator(tempValidator)
	afterAddCount := len(cfg.GetValidators())
	fmt.Printf("➕ 添加临时验证器后数量: %d\n", afterAddCount)

	// 测试临时验证器
	cfg.Set("temp.value", "test-value")
	cfg.Set("temp.number", 500)
	fmt.Printf("✅ 临时验证器验证通过\n")

	// 测试无效值
	if err := cfg.Set("temp.number", 2000); err != nil {
		fmt.Printf("✅ 临时验证器成功拦截无效值: %v\n", err)
	}

	// 添加函数式验证器
	cfg.AddValidateFunc(func(config map[string]any) error {
		// 自定义业务逻辑验证
		if val, exists := config["temp"].(map[string]any); exists {
			if value, ok := val["value"].(string); ok && value == "forbidden" {
				return fmt.Errorf("forbidden value not allowed")
			}
		}
		return nil
	})

	functionValidatorCount := len(cfg.GetValidators())
	fmt.Printf("➕ 添加函数式验证器后数量: %d\n", functionValidatorCount)

	// 测试函数式验证器
	if err := cfg.Set("temp.value", "forbidden"); err != nil {
		fmt.Printf("✅ 函数式验证器成功拦截禁用值: %v\n", err)
	}

	// 最终清除验证器演示
	fmt.Println("🧹 清除所有验证器...")
	cfg.ClearValidators()
	finalCount := len(cfg.GetValidators())
	fmt.Printf("📊 清除后验证器数量: %d\n", finalCount)

	// 重新添加基础验证器，确保应用正常运行
	cfg.AddValidator(validation.NewWebServerValidator())
	cfg.AddValidator(validation.NewDatabaseValidator())
	fmt.Printf("🔄 重新添加基础验证器，当前数量: %d\n", len(cfg.GetValidators()))
}

// DemoAdvancedTagging 演示新的标签系统
func (a *App) DemoAdvancedTagging() {
	fmt.Println("\n--- 高级标签系统演示 ---")

	cfg := a.cfg

	// 🔧 配置健康检查 - 确保关键配置字段有效
	fmt.Println("🔍 执行配置健康检查...")

	// 检查并修复server.timeout字段
	if timeoutStr := cfg.GetString("server.timeout"); timeoutStr != "" {
		if _, err := time.ParseDuration(timeoutStr); err != nil {
			fmt.Printf("⚠️  检测到无效的超时配置 '%s'，正在修复...\n", timeoutStr)
			cfg.Set("server.timeout", "30s")
		}
	}

	// 检查并修复server.port字段
	if port := cfg.GetInt("server.port"); port <= 0 || port > 65535 {
		fmt.Printf("⚠️  检测到无效的端口配置 %d，正在修复...\n", port)
		cfg.Set("server.port", 8080)
	}

	fmt.Println("✅ 配置健康检查完成")

	var config AppConfig

	// 尝试解析配置到结构体
	if err := cfg.Unmarshal(&config); err != nil {
		fmt.Printf("❌ 配置解析失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 配置解析成功！\n")
	fmt.Printf("应用名称: %s (默认值作为后备)\n", config.App.Name)
	fmt.Printf("应用版本: %s (默认值 + 语义化版本验证)\n", config.App.Version)
	fmt.Printf("运行环境: %s (枚举值限制)\n", config.App.Env)

	// 演示密码字段（必填但无默认值）
	if config.Database.Password != "" {
		fmt.Printf("数据库密码: %s (已设置)\n", strings.Repeat("*", len(config.Database.Password)))
	} else {
		fmt.Println("数据库密码: 未设置 (required字段)")
	}

	fmt.Printf("服务器端口: %d (默认值 + 范围验证)\n", config.Server.Port)
	fmt.Printf("服务器超时: %v (持续时间类型)\n", config.Server.Timeout)
	fmt.Printf("最大连接数: %d (非必填，有默认值和范围限制)\n", config.Database.MaxConns)
}

func main() {
	log.Println("🚀 Sysconf 配置管理库示例演示")
	log.Println("================================")

	// 创建应用实例
	app, err := NewApp()
	if err != nil {
		log.Fatalf("创建应用失败: %v", err)
	}

	// 演示基础配置使用
	app.DemoBasicUsage()

	// 演示高级功能
	app.DemoAdvancedFeatures()

	// 演示环境变量功能
	DemoEnvironmentVariables()

	// 演示全局配置
	DemoGlobalConfig()

	// 演示错误处理
	app.DemoErrorHandling()

	// 演示配置验证功能
	app.DemoConfigValidation()

	// 演示浮点数切片功能
	app.DemoFloatSlice()

	// 演示高级标签系统
	app.DemoAdvancedTagging()

	// 🆕 演示新验证器系统
	app.DemoValidationSystem()

	// 保持程序运行以观察配置变化
	log.Println("\n=== 配置热重载演示 ===")
	log.Println("应用正在运行。请编辑 ./app.yaml 文件以查看热重载效果...")
	log.Println("程序将在10秒后自动更新配置进行演示...")

	// 模拟配置变化
	time.AfterFunc(5*time.Second, func() {
		log.Println("\n--- 模拟配置更新 ---")
		if err := app.cfg.Set("server.port", 9091); err != nil {
			log.Printf("更新配置失败: %v", err)
		}

		time.Sleep(2 * time.Second)
		if err := app.cfg.Set("server.host", "updated-host"); err != nil {
			log.Printf("更新配置失败: %v", err)
		}
	})

	// 运行10秒钟展示热重载
	time.Sleep(10 * time.Second)
	log.Println("\n✅ 示例演示完成!")
}
