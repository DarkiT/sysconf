package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/darkit/sysconf"
)

// 创建示例配置文件内容
const defaultConfig = `
server:
  host: localhost
  port: 8080
  timeout: 30s
  features:
    - http
    - grpc

database:
  host: localhost
  port: 5432
  user: postgres
  password: secret
  timeout: 10s
  max_conns: 10
  options:
    sslmode: disable
    timezone: UTC

redis:
  addresses:
    - localhost:6379
  password: ""
  db: 0
  timeout: 5s

logging:
  level: info
  format: json
  path: logs/app.log
`

// AppConfig 定义应用配置结构
type AppConfig struct {
	Server struct {
		Host     string        `config:"host" default:"localhost"`     // 服务器主机
		Port     int           `config:"port" default:"8080"`          // 服务器端口
		Timeout  time.Duration `config:"timeout" default:"30s"`        // 超时时间
		Features []string      `config:"features" default:"http,grpc"` // 功能列表
	} `config:"server"`

	Database struct {
		Host     string            `config:"host" default:"localhost"` // 数据库主机
		Port     int               `config:"port" default:"5432"`      // 数据库端口
		User     string            `config:"user" required:"true"`     // 数据库用户
		Password string            `config:"password" required:"true"` // 数据库密码
		Timeout  time.Duration     `config:"timeout" default:"10s"`    // 超时时间
		MaxConns int               `config:"max_conns" default:"10"`   // 最大连接数
		Options  map[string]string `config:"options"`                  // 数据库选项
	} `config:"database"`

	Redis struct {
		Addresses []string      `config:"addresses" default:"localhost:6379"` // Redis 地址列表
		Password  string        `config:"password"`                           // Redis 密码
		DB        int           `config:"db" default:"0"`                     // Redis 数据库索引
		Timeout   time.Duration `config:"timeout" default:"5s"`               // 超时时间
	} `config:"redis"`

	Logging struct {
		Level  string `config:"level" default:"info"`        // 日志级别
		Format string `config:"format" default:"json"`       // 日志格式
		Path   string `config:"path" default:"logs/app.log"` // 日志文件路径
	} `config:"logging"`
}

// App 定义应用结构
type App struct {
	config atomic.Value // 存储 *AppConfig
	cfg    *sysconf.Config
}

// NewApp 创建应用实例
func NewApp() (*App, error) {
	// 创建配置实例
	cfg, err := sysconf.New(
		sysconf.WithPath("."),
		sysconf.WithMode("yaml"),
		sysconf.WithName("app"),
		// sysconf.WithContent(defaultConfig),
		sysconf.WithEnvOptions(sysconf.EnvOptions{
			Prefix:  "APP",
			Enabled: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("创建配置失败: %w", err)
	}

	// 或者后续更新环境变量选项
	_ = cfg.SetEnvPrefix("APP")

	app := &App{
		cfg: cfg,
	}

	// 加载初始配置
	if err := app.loadConfig(); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 启动配置监听
	app.watchConfig()

	return app, nil
}

// loadConfig 加载配置
func (a *App) loadConfig() error {
	var cfg AppConfig
	if err := a.cfg.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}
	a.config.Store(&cfg)
	return nil
}

// watchConfig 监听配置变化
func (a *App) watchConfig() {
	a.cfg.Watch(func() {
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
	log.Printf("服务器配置已更改: %+v", cfg.Server)
	log.Printf("数据库配置已更改: %+v", cfg.Database)
	log.Printf("Redis 配置已更改: %+v", cfg.Redis)
}

// GetConfig 安全地获取当前配置
func (a *App) GetConfig() *AppConfig {
	return a.config.Load().(*AppConfig)
}

// DemoConfigUsage 演示配置使用
func (a *App) DemoConfigUsage() {
	// 1. 使用结构体方式访问配置
	cfg := a.GetConfig()
	log.Printf("服务器主机: %s\n", cfg.Server.Host)
	log.Printf("服务器端口: %d\n", cfg.Server.Port)
	log.Printf("服务器功能: %v\n", cfg.Server.Features)

	// 2. 使用键值方式访问配置
	host := a.cfg.GetString("server.host")
	port := a.cfg.GetInt("server.port")
	log.Printf("服务器主机（通过键）: %s\n", host)
	log.Printf("服务器端口（通过键）: %d\n", port)

	// 3. 演示默认值
	unknownValue := a.cfg.GetString("unknown.key", "default-value")
	log.Printf("未知值（默认值）: %s\n", unknownValue)

	// 4. 演示类型化访问
	var serverConfig map[string]interface{}
	if err := a.cfg.Unmarshal(&serverConfig, "server"); err != nil {
		log.Printf("反序列化服务器配置失败: %v", err)
	}
	log.Printf("服务器配置映射: %+v\n", serverConfig)

	// 或者使用结构体
	var serverStruct struct {
		Host     string        `config:"host"`
		Port     int           `config:"port"`
		Timeout  time.Duration `config:"timeout"`
		Features []string      `config:"features"`
	}
	if err := a.cfg.Unmarshal(&serverStruct, "server"); err != nil {
		log.Printf("反序列化服务器配置失败: %v", err)
	}
	log.Printf("服务器配置结构体: %+v\n", serverStruct)
}

// DemoGlobalConfig 演示全局配置使用
func DemoGlobalConfig() {
	// 注册全局配置
	sysconf.Register("app", "name", "MyApp")
	sysconf.Register("app", "version", "1.0.0")

	// 获取全局配置
	appName := sysconf.Default().GetString("app.name")
	appVersion := sysconf.Default().GetString("app.version")

	log.Printf("应用名称: %s\n", appName)
	log.Printf("应用版本: %s\n", appVersion)
}

func main() {
	// 创建应用实例
	app, err := NewApp()
	if err != nil {
		log.Fatalf("创建应用失败: %v", err)
	}

	// 演示配置使用
	app.DemoConfigUsage()

	// 演示全局配置
	DemoGlobalConfig()

	// 保持程序运行以观察配置变化
	log.Println("应用正在运行。请更新 ./app.yaml 以查看变化...")

	time.AfterFunc(10*time.Second, func() {
		_ = app.cfg.Set("server.port", 9090)
		time.Sleep(5 * time.Second)
		_ = app.cfg.Set("server.port", 9898)
		_ = app.cfg.Set("redis.addresses", "127.0.0.1:6379")
	})
	time.Sleep(20 * time.Second)
}
