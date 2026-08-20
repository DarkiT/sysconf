package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/darkit/sysconf"
	"github.com/darkit/sysconf/validation"
)

// 与 README「快速开始」一致的默认配置
const defaultConfig = `
app:
  name: "MyApp"
  version: "1.0.0"
  debug: false

database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "demo"          # 必填字段，演示用，生产请用环境变量
  timeout: "30s"
  max_conns: 10

server:
  features: ["http", "grpc", "websocket"]
  ports: [8080, 8443]
`

// 与 README 示例对应的结构体映射
type AppConfig struct {
	App struct {
		Name    string `config:"name" default:"MyApp" validate:"required,min=1"`
		Version string `config:"version" default:"1.0.0" validate:"required,semver"`
		Debug   bool   `config:"debug" default:"false"`
	} `config:"app"`

	Database struct {
		Host     string        `config:"host" default:"localhost" validate:"required,hostname_rfc1123"`
		Port     int           `config:"port" default:"5432" validate:"required,min=1,max=65535"`
		Username string        `config:"username" default:"postgres" validate:"required,min=1"`
		Password string        `config:"password" validate:"required,min=1"`
		Timeout  time.Duration `config:"timeout" default:"30s" validate:"required"`
		MaxConns int           `config:"max_conns" default:"10" validate:"min=1,max=100"`
	} `config:"database"`

	Server struct {
		Features []string `config:"features"`
		Ports    []int    `config:"ports"`
	} `config:"server"`
}

func main() {
	log.Println("🚀 Sysconf 示例（对齐 README 快速开始）")

	// 构建配置：明确路径、文件名与格式，提供完整默认值
	cfg, err := sysconf.New(
		sysconf.WithPath("./examples"),     // 配置目录
		sysconf.WithName("app"),            // 配置文件名 app.yaml
		sysconf.WithMode("yaml"),           // 配置格式
		sysconf.WithContent(defaultConfig), // 写入默认配置（含必填密码）
		sysconf.WithEnvOptions(sysconf.EnvOptions{
			Prefix:    "APP", // 允许 APP_* 环境变量覆盖
			Enabled:   true,
			SmartCase: true,
		}),
		sysconf.WithSkipInitValidation(),
		sysconf.WithValidators(
			validation.NewDatabaseValidator(),
			validation.NewWebServerValidator(),
		),
	)
	if err != nil {
		log.Fatalf("创建配置失败: %v", err)
	}

	// 读取配置到结构体
	var config AppConfig
	if err := cfg.Unmarshal(&config); err != nil {
		log.Fatalf("配置解析失败: %v", err)
	}
	printConfig("初始配置", config)

	// 演示验证：无效端口会被拦截
	if err := cfg.Set("database.port", 70000); err != nil {
		log.Printf("✅ 验证器拦截无效端口: %v", err)
	}
	_ = cfg.Set("database.port", 5432) // 设回有效值

	// 演示环境变量覆盖敏感字段
	_ = os.Setenv("APP_DATABASE_PASSWORD", "super-secret")
	defer func() {
		_ = os.Unsetenv("APP_DATABASE_PASSWORD")
	}()
	if err := cfg.Unmarshal(&config); err != nil {
		log.Fatalf("重新加载配置失败: %v", err)
	}
	printConfig("环境变量覆盖后", config)

	// 安全获取字段示例（编译期类型安全）
	host := sysconf.GetAs(cfg, "database.host", "localhost")
	port := sysconf.GetAs(cfg, "database.port", 5432)
	timeout := sysconf.GetAs(cfg, "database.timeout", 30*time.Second)
	log.Printf("类型安全读取: host=%s port=%d timeout=%v", host, port, timeout)

	log.Println("✅ 示例结束，配置文件位于 ./examples/app.yaml")
}

func printConfig(title string, cfg AppConfig) {
	fmt.Printf("\n--- %s ---\n", title)
	fmt.Printf("应用: %s v%s (debug=%v)\n", cfg.App.Name, cfg.App.Version, cfg.App.Debug)
	fmt.Printf("数据库: %s:%d 连接超时 %v 最大连接数 %d\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Timeout, cfg.Database.MaxConns)
	masked := "<空>"
	if cfg.Database.Password != "" {
		masked = cfg.Database.Password[:1] + "***"
	}
	fmt.Printf("数据库密码: %s\n", masked)
	fmt.Printf("服务器特性: %v 端口: %v\n", cfg.Server.Features, cfg.Server.Ports)
}
