package main

import (
	"fmt"
	"log"

	"github.com/darkit/sysconf"
	"github.com/darkit/sysconf/validation"
)

func main() {
	fmt.Println("🚀 Sysconf 统一验证器系统演示")
	fmt.Println("=====================================")

	// 演示1：内存模式配置 + 新验证器系统
	demonstrateMemoryConfig()

	// 演示2：多种新验证器组合
	demonstrateNewValidators()

	// 演示3：预定义验证器
	demonstratePredefinedValidators()

	// 演示4：复合验证器
	demonstrateCompositeValidators()

	// 演示5：动态验证器管理
	demonstrateDynamicValidators()
}

// demonstrateMemoryConfig 演示内存模式配置
func demonstrateMemoryConfig() {
	fmt.Println("\n📋 演示1：内存模式配置 + 新验证器")
	fmt.Println("─────────────────────────────────")

	// YAML配置内容
	const yamlConfig = `
app:
  name: "验证器演示应用"
  version: "2.0.0"
  environment: "development"

database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "password123"
  type: "postgresql"

server:
  host: "0.0.0.0"
  port: 8080
  mode: "development"
  ssl:
    enabled: false
`

	// 🆕 使用新验证器系统
	portValidator := validation.NewRuleValidator("端口验证器")
	portValidator.AddStringRule("database.port", "port")
	portValidator.AddStringRule("server.port", "port")

	requiredValidator := validation.NewRuleValidator("必填验证器")
	requiredValidator.AddStringRule("database.host", "required")
	requiredValidator.AddStringRule("database.username", "required")
	requiredValidator.AddStringRule("server.host", "required")

	// 创建内存配置（不指定文件名）
	cfg, err := sysconf.New(
		sysconf.WithMode("yaml"),
		sysconf.WithContent(yamlConfig),
		// 🆕 添加新验证器
		sysconf.WithValidators(portValidator, requiredValidator),
	)
	if err != nil {
		log.Fatalf("❌ 创建配置失败: %v", err)
	}

	// 读取并显示配置
	fmt.Printf("✅ 应用名称: %s\n", cfg.GetString("app.name"))
	fmt.Printf("✅ 数据库端口: %d\n", cfg.GetInt("database.port"))
	fmt.Printf("✅ 服务器端口: %d\n", cfg.GetInt("server.port"))

	// 测试验证器
	fmt.Println("\n🔍 测试新验证器...")
	if err := cfg.Set("database.port", 70000); err != nil {
		fmt.Printf("✅ 验证器成功拦截无效端口: %v\n", err)
	}
}

// demonstrateNewValidators 演示多种新验证器
func demonstrateNewValidators() {
	fmt.Println("\n📋 演示2：多种新验证器组合")
	fmt.Println("─────────────────────────────────")

	const testConfig = `
user:
  email: "admin@example.com"
  phone: "+1234567890"
  website: "https://example.com"
  uuid: "550e8400-e29b-41d4-a716-446655440000"

api:
  endpoint: "https://api.example.com/v1"
  timeout: 30
  rate_limit: 1000

network:
  ipv4: "192.168.1.1"
  ipv6: "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
  port: 443
`

	// 🆕 创建多种新验证器
	emailValidator := validation.NewRuleValidator("邮箱验证器")
	emailValidator.AddStringRule("user.email", "email")

	urlValidator := validation.NewRuleValidator("URL验证器")
	urlValidator.AddStringRule("user.website", "url")
	urlValidator.AddStringRule("api.endpoint", "url")

	networkValidator := validation.NewRuleValidator("网络验证器")
	networkValidator.AddStringRule("network.ipv4", "ipv4")
	networkValidator.AddStringRule("network.ipv6", "ipv6")
	networkValidator.AddStringRule("network.port", "port")

	uuidValidator := validation.NewRuleValidator("UUID验证器")
	uuidValidator.AddStringRule("user.uuid", "uuid")

	cfg, err := sysconf.New(
		sysconf.WithMode("yaml"),
		sysconf.WithContent(testConfig),
		sysconf.WithValidators(emailValidator, urlValidator, networkValidator, uuidValidator),
	)
	if err != nil {
		log.Fatalf("❌ 创建配置失败: %v", err)
	}

	// 显示验证结果
	fmt.Printf("✅ 用户邮箱: %s\n", cfg.GetString("user.email"))
	fmt.Printf("✅ 用户网站: %s\n", cfg.GetString("user.website"))
	fmt.Printf("✅ IPv4地址: %s\n", cfg.GetString("network.ipv4"))
	fmt.Printf("✅ UUID: %s\n", cfg.GetString("user.uuid"))

	// 测试无效值
	fmt.Println("\n🔍 测试无效值...")
	if err := cfg.Set("user.email", "invalid-email"); err != nil {
		fmt.Printf("✅ 邮箱验证器工作正常: %v\n", err)
	}
}

// demonstratePredefinedValidators 演示预定义验证器
func demonstratePredefinedValidators() {
	fmt.Println("\n📋 演示3：预定义验证器")
	fmt.Println("─────────────────────────────")

	const appConfig = `
database:
  host: "db.example.com"
  port: 5432
  username: "dbuser"
  password: "dbpass123"
  type: "postgresql"

server:
  host: "0.0.0.0"
  port: 8080
  mode: "production"
  ssl:
    enabled: true
    cert_file: "/etc/ssl/cert.pem"
    key_file: "/etc/ssl/key.pem"

redis:
  host: "redis.example.com"
  port: 6379
  db: 1
  password: "redis123"
  timeout: 60

log:
  level: "info"
  format: "json"
  output: "/var/log/app.log"
  max_size: 100
  max_backups: 10

email:
  smtp:
    host: "smtp.example.com"
    port: 587
    username: "user@example.com"
    password: "emailpass"
  from: "noreply@example.com"

api:
  base_url: "https://api.example.com"
  timeout: 30
  rate_limit:
    enabled: true
    requests_per_minute: 1000
  auth:
    api_key: "api123456"
`

	// 🆕 使用预定义验证器
	cfg, err := sysconf.New(
		sysconf.WithMode("yaml"),
		sysconf.WithContent(appConfig),
		sysconf.WithValidators(
			validation.NewDatabaseValidator(),
			validation.NewWebServerValidator(),
			validation.NewRedisValidator(),
			validation.NewLogValidator(),
			validation.NewEmailValidator(),
			validation.NewAPIValidator(),
		),
	)
	if err != nil {
		log.Fatalf("❌ 创建配置失败: %v", err)
	}

	fmt.Printf("✅ 数据库主机: %s\n", cfg.GetString("database.host"))
	fmt.Printf("✅ 服务器模式: %s\n", cfg.GetString("server.mode"))
	fmt.Printf("✅ Redis主机: %s\n", cfg.GetString("redis.host"))
	fmt.Printf("✅ 日志级别: %s\n", cfg.GetString("log.level"))
	fmt.Printf("✅ SMTP主机: %s\n", cfg.GetString("email.smtp.host"))
	fmt.Printf("✅ API地址: %s\n", cfg.GetString("api.base_url"))

	fmt.Println("✅ 所有预定义验证器验证通过！")
}

// demonstrateCompositeValidators 演示复合验证器
func demonstrateCompositeValidators() {
	fmt.Println("\n📋 演示4：复合验证器")
	fmt.Println("───────────────────────────")

	const webAppConfig = `
database:
  host: "localhost"
  port: 5432
  username: "webapp"
  password: "webpass"
  type: "postgresql"

server:
  host: "localhost"
  port: 3000
  mode: "development"
  ssl:
    enabled: false

log:
  level: "debug"
  format: "text"
  output: "stdout"
`

	// 🆕 创建复合验证器
	composite := validation.NewCompositeValidator(
		"Web应用验证器",
		validation.NewDatabaseValidator(),
		validation.NewWebServerValidator(),
		validation.NewLogValidator(),
	)

	cfg, err := sysconf.New(
		sysconf.WithMode("yaml"),
		sysconf.WithContent(webAppConfig),
		sysconf.WithValidator(composite),
	)
	if err != nil {
		log.Fatalf("❌ 创建配置失败: %v", err)
	}

	fmt.Printf("✅ 复合验证器包含 %d 个子验证器\n", len(composite.GetValidators()))
	fmt.Printf("✅ 验证器名称: %s\n", composite.GetName())
	fmt.Printf("✅ 数据库类型: %s\n", cfg.GetString("database.type"))
	fmt.Printf("✅ 服务器端口: %d\n", cfg.GetInt("server.port"))
}

// demonstrateDynamicValidators 演示动态验证器管理
func demonstrateDynamicValidators() {
	fmt.Println("\n📋 演示5：动态验证器管理")
	fmt.Println("─────────────────────────────")

	// 创建基础配置
	cfg, err := sysconf.New(
		sysconf.WithMode("yaml"),
		sysconf.WithContent(`
app:
  name: "动态验证演示"
  port: 8080
`),
	)
	if err != nil {
		log.Fatalf("❌ 创建配置失败: %v", err)
	}

	fmt.Printf("初始验证器数量: %d\n", len(cfg.GetValidators()))

	// 🆕 动态添加验证器
	portValidator := validation.NewRuleValidator("动态端口验证器")
	portValidator.AddStringRule("app.port", "port")
	cfg.AddValidator(portValidator)

	fmt.Printf("添加端口验证器后: %d 个验证器\n", len(cfg.GetValidators()))

	// 添加自定义验证函数
	cfg.AddValidateFunc(func(config map[string]any) error {
		if appName, exists := config["app"].(map[string]any)["name"]; exists {
			if appName == "" {
				return fmt.Errorf("application name cannot be empty")
			}
		}
		return nil
	})

	fmt.Printf("添加自定义验证函数后: %d 个验证器\n", len(cfg.GetValidators()))

	// 测试验证
	if err := cfg.Set("app.port", 99999); err != nil {
		fmt.Printf("✅ 动态验证器工作正常: %v\n", err)
	}

	// 清除验证器
	cfg.ClearValidators()
	fmt.Printf("清除所有验证器后: %d 个验证器\n", len(cfg.GetValidators()))

	fmt.Println("\n🎉 新验证器系统演示完成！")
	fmt.Println("=====================================")
	fmt.Println("✨ 新系统特点:")
	fmt.Println("   • 30+种验证规则")
	fmt.Println("   • 预定义验证器")
	fmt.Println("   • 复合验证器支持")
	fmt.Println("   • 动态验证器管理")
	fmt.Println("   • 结构化和字符串双重规则")
	fmt.Println("   • 完全替代旧验证器系统")
}
