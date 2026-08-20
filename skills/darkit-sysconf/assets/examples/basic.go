//go:build ignore

// basic 演示加载配置、env 覆盖、类型 getter、泛型 GetAs、struct 解析与热重载。
// 运行： go run basic.go
package main

import (
	"log"
	"time"

	"github.com/darkit/sysconf"
)

// AppConfig 目标结构体；tag 支持 config/sysconf/json，default 填默认值。
type AppConfig struct {
	Server struct {
		Host    string        `config:"host" default:"0.0.0.0"`
		Port    uint16        `config:"port" default:"8080"` // uint 字段的 default 也被支持
		Timeout time.Duration `config:"timeout" default:"30s"`
	} `config:"server"`
}

func main() {
	const content = `
server:
  host: "127.0.0.1"
  port: 9090
`
	app, cfg, err := sysconf.Load[AppConfig](
		sysconf.WithContent(content),
		sysconf.WithMode("yaml"),
		sysconf.WithEnv("MYAPP"), // 用 MYAPP_ 前缀覆盖，务必带前缀（见 SKILL 坑 #1）
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()

	// 类型 getter（方法）
	log.Printf("host=%s port=%d", cfg.GetString("server.host"), cfg.GetInt("server.port"))

	// 泛型 getter（包级函数），类型安全
	port := sysconf.GetAs[uint16](cfg, "server.port", 8080)
	log.Printf("typed port=%d", port)

	log.Printf("struct: %+v", app.Server)

	// 运行时写入 + 读回（返回值是拷贝，改它不影响内部存储）
	_ = cfg.Set("server.host", "0.0.0.0")
	log.Printf("after set: host=%s", cfg.GetString("server.host"))
}
