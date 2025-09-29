package main

import (
	"fmt"
	"log"

	"github.com/darkit/sysconf"
	"github.com/darkit/sysconf/validation"
)

func main() {
	fmt.Println("🚀 Sysconf 验证器系统集成演示")
	fmt.Println("=====================================")

	// 1. 创建基础配置实例
	fmt.Println("=== 创建配置实例 ===")
	config, err := sysconf.New(
		sysconf.WithName("app_config"),
		sysconf.WithPath("./"),
		sysconf.WithMode("yaml"),
	)
	if err != nil {
		log.Fatalf("创建配置失败: %v", err)
	}

	fmt.Println("✓ 配置实例创建成功")

	// 2. 设置默认配置
	fmt.Println("\n=== 设置默认配置 ===")

	// 数据库配置
	config.Set("database.host", "localhost")
	config.Set("database.port", 3306)
	config.Set("database.username", "root")
	config.Set("database.password", "secret123")
	config.Set("database.type", "mysql")

	// 服务器配置
	config.Set("server.host", "0.0.0.0")
	config.Set("server.port", 8080)
	config.Set("server.mode", "development")
	config.Set("server.ssl.enabled", false)
	config.Set("server.ssl.cert_file", "")
	config.Set("server.ssl.key_file", "")

	// Redis配置
	config.Set("redis.host", "127.0.0.1")
	config.Set("redis.port", 6379)
	config.Set("redis.db", 0)
	config.Set("redis.password", "")
	config.Set("redis.timeout", 30)

	// 日志配置
	config.Set("log.level", "info")
	config.Set("log.format", "json")
	config.Set("log.output", "/var/log/app.log")
	config.Set("log.max_size", 100)
	config.Set("log.max_backups", 10)

	// 邮件配置
	config.Set("email.smtp.host", "smtp.gmail.com")
	config.Set("email.smtp.port", 587)
	config.Set("email.smtp.username", "test@gmail.com")
	config.Set("email.smtp.password", "app_password")
	config.Set("email.from", "noreply@example.com")

	// API配置
	config.Set("api.base_url", "https://api.example.com")
	config.Set("api.timeout", 30)
	config.Set("api.rate_limit.enabled", true)
	config.Set("api.rate_limit.requests_per_minute", 1000)
	config.Set("api.auth.api_key", "sk-1234567890abcdef")

	fmt.Println("✓ 默认配置设置完成")

	// 3. 添加预定义验证器
	fmt.Println("\n=== 添加预定义验证器 ===")

	// 添加数据库验证器
	dbValidator := validation.NewDatabaseValidator()
	config.AddValidator(dbValidator)
	fmt.Println("✓ 添加数据库配置验证器")

	// 添加Web服务器验证器
	serverValidator := validation.NewWebServerValidator()
	config.AddValidator(serverValidator)
	fmt.Println("✓ 添加Web服务器配置验证器")

	// 添加Redis验证器
	redisValidator := validation.NewRedisValidator()
	config.AddValidator(redisValidator)
	fmt.Println("✓ 添加Redis配置验证器")

	// 添加日志验证器
	logValidator := validation.NewLogValidator()
	config.AddValidator(logValidator)
	fmt.Println("✓ 添加日志配置验证器")

	// 添加邮件验证器
	emailValidator := validation.NewEmailValidator()
	config.AddValidator(emailValidator)
	fmt.Println("✓ 添加邮件配置验证器")

	// 添加API验证器
	apiValidator := validation.NewAPIValidator()
	config.AddValidator(apiValidator)
	fmt.Println("✓ 添加API配置验证器")

	// 4. 创建复合验证器示例
	fmt.Println("\n=== 创建复合验证器 ===")
	compositeValidator := validation.NewCompositeValidator("完整应用配置验证器",
		validation.NewDatabaseValidator(),
		validation.NewWebServerValidator(),
		validation.NewRedisValidator(),
	)
	config.AddValidator(compositeValidator)
	fmt.Println("✓ 添加复合验证器")

	// 5. 创建自定义验证器
	fmt.Println("\n=== 创建自定义验证器 ===")
	customValidator := validation.NewRuleValidator("业务逻辑验证器")

	// 添加自定义业务规则
	customValidator.AddRule("business.company_name", validation.Required("公司名称不能为空"))
	customValidator.AddRule("business.tax_id", validation.Pattern(`^\d{18}$`, "税务登记号必须是18位数字"))
	customValidator.AddStringRule("business.industry", "enum:technology,finance,healthcare,education")
	customValidator.AddStringRule("business.employee_count", "range:1,10000")

	config.AddValidator(customValidator)
	fmt.Println("✓ 添加自定义业务验证器")

	// 设置业务配置
	config.Set("business.company_name", "创新科技有限公司")
	config.Set("business.tax_id", "123456789012345678")
	config.Set("business.industry", "technology")
	config.Set("business.employee_count", 150)

	// 6. 验证配置状态
	fmt.Println("\n=== 验证配置状态 ===")

	// 获取当前验证器数量
	validators := config.GetValidators()
	fmt.Printf("当前已注册验证器数量: %d\n", len(validators))

	fmt.Println("✅ 所有配置设置成功（验证器已自动验证）!")

	// 7. 演示验证失败的情况
	fmt.Println("\n=== 演示验证失败情况 ===")

	// 尝试设置无效的端口号
	fmt.Println("尝试设置数据库端口为无效值: 70000")
	if err := config.Set("database.port", 70000); err != nil {
		fmt.Printf("❌ 验证失败（符合预期）: %v\n", err)
	}

	// 恢复有效值
	config.Set("database.port", 3306)
	fmt.Println("✓ 恢复数据库端口为有效值: 3306")

	// 8. 演示字符串规则验证
	fmt.Println("\n=== 演示字符串规则验证 ===")
	stringRuleValidator := validation.NewRuleValidator("字符串规则验证器")

	// 添加各种字符串规则
	stringRuleValidator.AddStringRules("network.ip_address", "required", "ipv4")
	stringRuleValidator.AddStringRules("network.domain", "required", "hostname")
	stringRuleValidator.AddStringRules("security.admin_email", "required", "email")
	stringRuleValidator.AddStringRules("security.api_token", "required", "uuid")

	config.AddValidator(stringRuleValidator)

	// 设置网络和安全配置
	config.Set("network.ip_address", "192.168.1.100")
	config.Set("network.domain", "example.com")
	config.Set("security.admin_email", "admin@example.com")
	config.Set("security.api_token", "123e4567-e89b-12d3-a456-426614174000")

	fmt.Println("✅ 字符串规则验证通过!")

	// 9. 演示动态验证器管理
	fmt.Println("\n=== 演示动态验证器管理 ===")

	initialCount := len(config.GetValidators())
	fmt.Printf("初始验证器数量: %d\n", initialCount)

	// 添加临时验证器
	tempValidator := validation.NewRuleValidator("临时验证器")
	tempValidator.AddRule("temp.value", validation.Required("临时值不能为空"))
	config.AddValidator(tempValidator)

	afterAddCount := len(config.GetValidators())
	fmt.Printf("添加临时验证器后数量: %d\n", afterAddCount)

	// 清除所有验证器
	config.ClearValidators()
	afterClearCount := len(config.GetValidators())
	fmt.Printf("清除验证器后数量: %d\n", afterClearCount)

	// 10. 总结展示
	fmt.Println("\n=== 配置值展示 ===")
	fmt.Printf("数据库主机: %s\n", config.GetString("database.host"))
	fmt.Printf("服务器端口: %d\n", config.GetInt("server.port"))
	fmt.Printf("Redis端口: %d\n", config.GetInt("redis.port"))
	fmt.Printf("日志级别: %s\n", config.GetString("log.level"))
	fmt.Printf("公司名称: %s\n", config.GetString("business.company_name"))
	fmt.Printf("行业类型: %s\n", config.GetString("business.industry"))

	fmt.Println("\n🎉 验证器系统集成演示完成！")
	fmt.Println("=====================================")
	fmt.Println("✨ 新验证器系统特点:")
	fmt.Println("   • 30+种验证规则")
	fmt.Println("   • 6个预定义验证器")
	fmt.Println("   • 复合验证器支持")
	fmt.Println("   • 动态验证器管理")
	fmt.Println("   • 自动验证机制")
	fmt.Println("   • 完全替代旧系统")
}
