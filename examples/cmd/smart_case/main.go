package main

import (
	"fmt"
	"log"
	"os"

	"github.com/darkit/sysconf"
)

func main() {
	fmt.Println("🚀 智能大小写环境变量匹配演示")
	fmt.Println("=====================================")

	// 🆕 演示智能大小写匹配功能
	demoSmartCaseEnvVars()
}

// demoSmartCaseEnvVars 演示智能大小写环境变量匹配
func demoSmartCaseEnvVars() {
	fmt.Println("\n📋 设置各种大小写格式的环境变量...")

	// 设置不同大小写格式的环境变量
	testEnvVars := map[string]string{
		"myapp_database_host":    "lowercase.example.com", // 小写
		"MYAPP_SERVER_PORT":      "8080",                  // 大写
		"MyApp_Cache_Timeout":    "5m",                    // 混合大小写
		"myapp_features_enabled": "true",                  // 小写布尔值
	}

	// 设置环境变量
	for key, value := range testEnvVars {
		os.Setenv(key, value)
		fmt.Printf("   ✅ %s=%s\n", key, value)
	}

	// 清理函数
	defer func() {
		fmt.Println("\n🧹 清理环境变量...")
		for key := range testEnvVars {
			os.Unsetenv(key)
		}
	}()

	fmt.Println("\n🔧 创建配置实例（启用智能大小写匹配）...")

	// 🆕 使用便利函数WithEnv，自动启用智能大小写匹配
	cfg, err := sysconf.New(
		sysconf.WithPath("/tmp"),           // 🔧 设置临时路径
		sysconf.WithName("demo"),           // 🔧 设置配置文件名
		sysconf.WithMode("yaml"),           // 🔧 明确设置配置格式
		sysconf.WithEnv("MYAPP"),           // 环境变量前缀 + 智能大小写匹配
		sysconf.WithContent(defaultConfig), // 默认配置内容
	)
	if err != nil {
		log.Fatal("创建配置失败:", err)
	}

	fmt.Println("\n📖 读取配置值（自动匹配各种大小写格式）...")

	// 读取配置，智能匹配会自动找到对应的环境变量
	dbHost := cfg.GetString("database.host", "default-host")
	serverPort := cfg.GetInt("server.port", 3000)
	cacheTimeout := cfg.GetString("cache.timeout", "1m")
	featuresEnabled := cfg.GetBool("features.enabled")

	// 显示结果
	fmt.Printf("   🗄️  数据库主机: %s (来自 myapp_database_host)\n", dbHost)
	fmt.Printf("   🌐 服务器端口: %d (来自 MYAPP_SERVER_PORT)\n", serverPort)
	fmt.Printf("   💾 缓存超时: %s (来自 MyApp_Cache_Timeout)\n", cacheTimeout)
	fmt.Printf("   🔧 功能启用: %t (来自 myapp_features_enabled)\n", featuresEnabled)

	fmt.Println("\n✨ 智能大小写匹配特性:")
	fmt.Println("   ✅ 支持全小写: myapp_database_host")
	fmt.Println("   ✅ 支持全大写: MYAPP_SERVER_PORT")
	fmt.Println("   ✅ 支持混合大小写: MyApp_Cache_Timeout")
	fmt.Println("   ✅ 自动转换为配置键名: database.host")
	fmt.Println("   ✅ 向后兼容传统大写格式")

	// 🆕 演示传统模式对比
	fmt.Println("\n🔄 对比传统模式（仅支持大写）...")
	demoTraditionalMode()
}

// demoTraditionalMode 演示传统模式（仅支持大写环境变量）
func demoTraditionalMode() {
	// 设置小写环境变量（传统模式不会识别）
	os.Setenv("traditional_test_value", "should_not_work")
	defer os.Unsetenv("traditional_test_value")

	// 使用传统模式（禁用智能大小写匹配）
	cfg, err := sysconf.New(
		sysconf.WithPath("/tmp"),                        // 🔧 设置临时路径
		sysconf.WithName("traditional"),                 // 🔧 设置配置文件名
		sysconf.WithMode("yaml"),                        // 🔧 明确设置配置格式
		sysconf.WithEnvSmartCase("TRADITIONAL", false),  // 🚫 禁用智能匹配
		sysconf.WithContent(`test: {value: "default"}`), // 基础配置
	)
	if err != nil {
		log.Fatal("创建传统配置失败:", err)
	}

	value := cfg.GetString("test.value", "default")

	if value == "default" {
		fmt.Println("   ✅ 传统模式：小写环境变量被忽略（符合预期）")
	} else {
		fmt.Printf("   ❌ 传统模式：意外识别了小写环境变量: %s\n", value)
	}
}

const defaultConfig = `
database:
  host: "localhost"
  port: 5432

server:
  port: 3000
  
cache:
  timeout: "1m"

features:
  enabled: false
`
