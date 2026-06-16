package main

import (
	"fmt"
	"log"
	"time"

	"github.com/darkit/sysconf"
)

// Config 示例配置结构
type Config struct {
	App struct {
		Name string `config:"name"`
	} `config:"app"`
	Database struct {
		Password string `config:"password"`
	} `config:"database"`
	JWT struct {
		SecretKey string `config:"secret_key"`
	} `config:"jwt"`
}

func main() {
	fmt.Println("=== sysconf 配置加密功能演示 ===")

	configContent := `
app:
  name: "加密演示应用"
database:
  password: "secret_password_123"
jwt:
  secret_key: "jwt_secret_key_456"
`

	// 演示基本加密功能
	fmt.Println("\n1. 基本加密功能演示")
	demoBasicEncryption(configContent)

	// 演示自定义加密器
	fmt.Println("\n2. 自定义加密器演示")
	demoCustomCrypto(configContent)

	// 演示性能测试
	fmt.Println("\n3. 性能测试")
	performanceTest(configContent)
}

func demoBasicEncryption(content string) {
	fmt.Println("🔒 使用默认 ChaCha20-Poly1305 加密:")

	// 创建加密配置
	config, err := sysconf.New(
		sysconf.WithPath("./demo_configs"),
		sysconf.WithName("encrypted_demo"),
		sysconf.WithMode("yaml"),
		sysconf.WithEncryption("my_encryption_key"),
		sysconf.WithContent(content),
	)
	if err != nil {
		log.Fatalf("创建加密配置失败: %v", err)
	}

	var appConfig Config
	if err := config.Unmarshal(&appConfig); err != nil {
		log.Fatalf("解析加密配置失败: %v", err)
	}

	fmt.Printf("   加密类型: %s\n", config.GetCryptoType())
	fmt.Printf("   应用名称: %s\n", appConfig.App.Name)
	fmt.Printf("   数据库密码: %s\n", appConfig.Database.Password)
	fmt.Printf("   JWT密钥: %s\n", appConfig.JWT.SecretKey)
	fmt.Printf("   加密密钥: %s\n", config.GetEncryptionKey())

	// 测试配置更新
	if err := config.Set("database.password", "new_encrypted_password"); err != nil {
		log.Printf("更新配置失败: %v", err)
	} else {
		fmt.Printf("   ✅ 配置更新成功，新密码已加密保存\n")
	}
}

func demoCustomCrypto(content string) {
	fmt.Println("🛠️ 自定义加密器演示:")

	// 创建自定义加密器
	customCrypto := &SimpleXORCrypto{key: []byte("custom_key_1234567890abcdef")}

	config, err := sysconf.New(
		sysconf.WithPath("./demo_configs"),
		sysconf.WithName("custom_crypto_demo"),
		sysconf.WithMode("yaml"),
		sysconf.WithEncryptionCrypto(customCrypto),
		sysconf.WithContent(content),
	)
	if err != nil {
		log.Fatalf("创建自定义加密配置失败: %v", err)
	}

	var appConfig Config
	if err := config.Unmarshal(&appConfig); err != nil {
		log.Fatalf("解析自定义加密配置失败: %v", err)
	}

	fmt.Printf("   加密类型: %s\n", config.GetCryptoType())
	fmt.Printf("   应用名称: %s\n", appConfig.App.Name)
	fmt.Printf("   自定义密钥: %s\n", customCrypto.GetKey())
	fmt.Printf("   ✅ 自定义加密器工作正常\n")
}

func performanceTest(content string) {
	fmt.Println("⏱️ 性能测试 (1000次加密/解密操作):")

	testData := []byte(content)
	iterations := 1000

	// 测试默认ChaCha20加密性能
	defaultCrypto, err := sysconf.NewDefaultCrypto("test_key")
	if err != nil {
		log.Fatalf("创建默认加密器失败: %v", err)
	}
	start := time.Now()
	for range iterations {
		encrypted, err := defaultCrypto.Encrypt(testData)
		if err != nil {
			log.Fatalf("默认加密失败: %v", err)
		}
		if _, err := defaultCrypto.Decrypt(encrypted); err != nil {
			log.Fatalf("默认解密失败: %v", err)
		}
	}
	defaultTime := time.Since(start)

	// 测试自定义简单加密性能
	customCrypto := &SimpleXORCrypto{key: []byte("test_key_1234567890abcdef")}
	start = time.Now()
	for range iterations {
		encrypted, err := customCrypto.Encrypt(testData)
		if err != nil {
			log.Fatalf("自定义加密失败: %v", err)
		}
		if _, err := customCrypto.Decrypt(encrypted); err != nil {
			log.Fatalf("自定义解密失败: %v", err)
		}
	}
	customTime := time.Since(start)

	fmt.Printf("   🔒 默认 ChaCha20:  %v (安全推荐)\n", defaultTime)
	fmt.Printf("   ⚡ 自定义 XOR:     %v (性能优先)\n", customTime)
	fmt.Printf("   📊 性能对比:      ChaCha20 是 XOR 的 %.2fx\n",
		float64(defaultTime)/float64(customTime))

	fmt.Println("   💡 建议: 对于频繁的配置更新场景，可考虑使用自定义轻量级加密器")
}

// =============================================================================
// 自定义加密器示例 - 简单XOR加密（仅用于演示）
// =============================================================================

type SimpleXORCrypto struct {
	key []byte
}

func (s *SimpleXORCrypto) Encrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("数据不能为空")
	}

	encrypted := make([]byte, len(data))
	for i, b := range data {
		encrypted[i] = b ^ s.key[i%len(s.key)]
	}

	// 添加简单的前缀标识
	result := append([]byte("CUSTOM_XOR:"), encrypted...)
	return result, nil
}

func (s *SimpleXORCrypto) Decrypt(data []byte) ([]byte, error) {
	if !s.IsEncrypted(data) {
		return nil, fmt.Errorf("数据格式无效")
	}

	// 移除前缀
	encrypted := data[11:] // "CUSTOM_XOR:" 的长度是 11

	decrypted := make([]byte, len(encrypted))
	for i, b := range encrypted {
		decrypted[i] = b ^ s.key[i%len(s.key)]
	}

	return decrypted, nil
}

func (s *SimpleXORCrypto) IsEncrypted(data []byte) bool {
	if len(data) < 11 {
		return false
	}
	return string(data[:11]) == "CUSTOM_XOR:"
}

func (s *SimpleXORCrypto) GetKey() string {
	return string(s.key)
}
