package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/darkit/sysconf"
	"github.com/darkit/sysconf/validation"
)

// 演示点：
// 1) 启动前清理历史文件，确保环境干净
// 2) 启用 ChaCha20 加密写盘（自动生成密钥）
// 3) 热重载监听：5 秒后自动修改配置，再触发回调
// 4) 环境变量覆盖密码
const demoConfig = `
app:
  name: "HotReloadDemo"
  env: "dev"

server:
  host: "127.0.0.1"
  port: 8080

redis:
  host: "127.0.0.1"
  port: 6379

database:
  host: "localhost"
  port: 5432
  username: "demo"
  password: "demo-pass"
  timeout: "15s"
`

func main() {
	cfgPath := "./examples"
	cfgName := "app_secure"
	cfgFile := fmt.Sprintf("%s/%s.yaml", cfgPath, cfgName)

	// 1) 清理历史文件，保证演示可重复
	_ = os.Remove(cfgFile)

	// 2) 创建加密 + 热重载配置
	cfg, err := sysconf.New(
		sysconf.WithPath(cfgPath),
		sysconf.WithName(cfgName),
		sysconf.WithMode("yaml"),
		sysconf.WithContent(demoConfig),
		sysconf.WithEncryption(""), // 自动生成密钥
		sysconf.WithEnvOptions(sysconf.EnvOptions{Prefix: "APP", Enabled: true, SmartCase: true}),
		sysconf.WithValidators(
			validation.NewDatabaseValidator(),
			validation.NewWebServerValidator(),
			validation.NewRedisValidator(),
		),
		sysconf.WithWriteFlushDelay(0), // 立刻写盘，便于观察
	)
	if err != nil {
		log.Fatalf("创建配置失败: %v", err)
	}

	// 演示环境变量覆盖敏感字段
	os.Setenv("APP_DATABASE_PASSWORD", "env-secret")
	defer os.Unsetenv("APP_DATABASE_PASSWORD")

	dump(cfg, "初始配置")

	// 3) 启动热重载监听
	ctx, cancel := context.WithCancel(context.Background())
	cfg.WatchWithContext(ctx, func() {
		dump(cfg, "检测到文件变更后的配置")
	})

	// 4) 5秒后模拟配置更新（通过 Set 会触发写盘 + 监听）
	time.AfterFunc(5*time.Second, func() {
		log.Println("\n--- 模拟写入新端口 9090（加密存盘） ---")
		_ = cfg.Set("server.port", 9090)
	})

	// 10 秒展示窗口
	time.Sleep(10 * time.Second)
	cancel()
	log.Println("✅ 热重载加密演示结束")
}

func dump(cfg *sysconf.Config, title string) {
	fmt.Printf("\n[%s]\n", title)
	fmt.Printf("server.host=%s port=%d\n", cfg.GetString("server.host"), cfg.GetInt("server.port"))
	fmt.Printf("database.password=%s\n", cfg.GetString("database.password"))
	fmt.Printf("加密密钥(仅演示): %s\n", cfg.GetEncryptionKey())
}
