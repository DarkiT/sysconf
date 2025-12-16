package main

import (
	"log"

	"github.com/darkit/sysconf"
	"github.com/darkit/sysconf/validation"
)

// 最小示例：与 README「快速开始」一致
func main() {
	cfg, err := sysconf.New(
		sysconf.WithPath("./examples"),
		sysconf.WithName("app"),
		sysconf.WithMode("yaml"),
		sysconf.WithContent(defaultConfig),
		sysconf.WithEnvOptions(sysconf.EnvOptions{Prefix: "APP", Enabled: true, SmartCase: true}),
		sysconf.WithValidators(validation.NewDatabaseValidator(), validation.NewWebServerValidator()),
	)
	if err != nil {
		log.Fatalf("创建配置失败: %v", err)
	}

	host := cfg.GetString("database.host", "localhost")
	port := cfg.GetInt("database.port", "5432")
	timeout := cfg.GetDuration("database.timeout")
	log.Printf("数据库: %s:%d timeout=%v", host, port, timeout)

	// 验证拦截示例
	if err := cfg.Set("database.port", 70000); err != nil {
		log.Printf("验证器拦截: %v", err)
	}
	_ = cfg.Set("database.port", 5432)

	// 泛型读取示例
	_ = sysconf.GetAs(cfg, "database.port", 5432)
}

const defaultConfig = `
app:
  name: "MyApp"
  version: "1.0.0"
  debug: false

database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "demo"
  timeout: "30s"
  max_conns: 10

server:
  features: ["http", "grpc", "websocket"]
  ports: [8080, 8443]
`
