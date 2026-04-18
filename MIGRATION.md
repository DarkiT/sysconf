# Viper 迁移指南

本文档帮助您从 `spf13/viper` 迁移到 `darkit/sysconf`。sysconf 完全兼容 viper 的 API，同时提供更强大的功能和更好的性能。

## 📋 目录

- [快速迁移](#快速迁移)
- [API 对比](#api-对比)
- [功能增强](#功能增强)
- [常见问题](#常见问题)

## 快速迁移

### 1. 替换导入

```go
// 之前
import "github.com/spf13/viper"

// 之后
import "github.com/darkit/sysconf"
```

### 2. 更新初始化代码

```go
// 之前 (viper)
v := viper.New()
v.SetConfigName("config")
v.SetConfigType("yaml")
v.AddConfigPath(".")
v.ReadInConfig()

// 之后 (sysconf) - 完全兼容
cfg, err := sysconf.New(
    sysconf.WithName("config"),
    sysconf.WithMode("yaml"),
    sysconf.WithPath("."),
)
if err != nil {
    log.Fatal(err)
}
```

### 3. 读取配置（无需修改）

```go
// 以下代码在 viper 和 sysconf 中完全相同
host := cfg.GetString("database.host")
port := cfg.GetInt("database.port")
debug := cfg.GetBool("app.debug")
```

## API 对比

### 基础 API

| Viper API | Sysconf API | 状态 |
|-----------|-------------|------|
| `viper.New()` | `sysconf.New(...)` | ✅ 兼容 |
| `v.Set()` | `cfg.Set()` | ✅ 兼容 |
| `v.GetString()` | `cfg.GetString()` | ✅ 兼容 |
| `v.GetInt()` | `cfg.GetInt()` | ✅ 兼容 |
| `v.GetBool()` | `cfg.GetBool()` | ✅ 兼容 |
| `v.GetDuration()` | `cfg.GetDuration()` | ✅ 兼容 |
| `v.GetStringSlice()` | `cfg.GetStringSlice()` | ✅ 兼容 |
| `v.GetStringMap()` | `cfg.GetStringMap()` | ✅ 兼容 |
| `v.Unmarshal()` | `cfg.Unmarshal()` | ✅ 兼容 |
| `v.WatchConfig()` | `cfg.Watch()` | ✅ 增强 |
| `v.OnConfigChange()` | `cfg.Watch()` | ✅ 增强 |

### 环境变量

```go
// 之前 (viper)
v.SetEnvPrefix("APP")
v.AutomaticEnv()

// 之后 (sysconf) - 更简洁
cfg, err := sysconf.New(
    sysconf.WithEnv("APP"), // 自动启用环境变量和智能大小写匹配
)
```

**增强功能**：
- 自动支持多种大小写格式（`app_database_host`, `APP_DATABASE_HOST`, `App_Database_Host`）
- 无需手动调用 `AutomaticEnv()`

### 命令行集成

```go
// 之前 (viper + cobra)
rootCmd.Flags().String("host", "", "Database host")
viper.BindPFlag("host", rootCmd.Flags().Lookup("host"))

// 之后 (sysconf + cobra) - 更简洁
rootCmd.Flags().String("host", "", "Database host")
cfg, err := sysconf.New(
    sysconf.WithPFlags(rootCmd.Flags()), // 一行搞定
)
```

### 配置默认值

```go
// 之前 (viper)
v.SetDefault("database.host", "localhost")
v.SetDefault("database.port", 5432)

// 之后 (sysconf) - 使用默认配置内容
defaultConfig := `
database:
  host: "localhost"
  port: 5432
`
cfg, err := sysconf.New(
    sysconf.WithContent(defaultConfig),
)

// 或使用结构体标签
type Config struct {
    Database struct {
        Host string `config:"host" default:"localhost"`
        Port int    `config:"port" default:"5432"`
    }
}
```

## 功能增强

### 1. 线程安全

sysconf 是**线程安全**的，无需额外同步：

```go
// sysconf - 线程安全，无需担心竞态条件
go func() {
    cfg.Set("key", "value1")
}()
go func() {
    cfg.Set("key", "value2")
}()
```

### 2. 泛型 API（Go 1.24+）

```go
// 使用泛型获得编译时类型安全
host := sysconf.GetAs[string](cfg, "database.host", "localhost")
port := sysconf.GetAs[int](cfg, "database.port", 5432)
timeout := sysconf.GetAs[time.Duration](cfg, "timeout", 30*time.Second)

// 必须存在的配置
apiKey := sysconf.MustGetAs[string](cfg, "api.secret_key") // 缺失时 panic
```

### 3. 验证系统

```go
import "github.com/darkit/sysconf/validation"

// 添加验证器
cfg, err := sysconf.New(
    sysconf.WithValidators(
        validation.NewDatabaseValidator(),
        validation.NewWebServerValidator(),
    ),
)

// 无效值会被自动拦截
cfg.Set("server.port", 70000) // ❌ 错误：端口必须在 1-65535 之间
```

### 4. 加密支持

```go
// 启用配置加密
cfg, err := sysconf.New(
    sysconf.WithEncryption("your-secret-password"),
)

// 敏感配置自动加密
cfg.Set("database.password", "super-secret")
```

### 5. 批量设置

```go
// 原子性批量更新
err := cfg.SetMultiple(map[string]any{
    "database.host": "localhost",
    "database.port": 5432,
    "app.debug":     false,
})
```

## 迁移检查清单

- [ ] 替换导入路径 `spf13/viper` → `darkit/sysconf`
- [ ] 更新初始化代码使用 `sysconf.New()`
- [ ] 检查环境变量配置（使用 `WithEnv()` 替代 `SetEnvPrefix` + `AutomaticEnv`）
- [ ] 检查命令行集成（使用 `WithPFlags()` 替代 `BindPFlag`）
- [ ] 运行测试确保功能正常
- [ ] （可选）添加验证器增强配置安全性
- [ ] （可选）使用泛型 API 获得类型安全

## 常见问题

### Q: 迁移后性能有变化吗？

A: sysconf 在读取性能上有显著提升：
- **Viper**: 读取操作需要加锁
- **Sysconf**: 使用 `atomic.Value` 实现无锁读取，延迟降低 10 倍以上

### Q: 可以混合使用 viper 和 sysconf 吗？

A: 不建议。虽然 API 兼容，但内部实现不同，混合使用可能导致不可预期的问题。

### Q: 热重载功能有变化吗？

A: 有增强。sysconf 的 `Watch()` 支持：
- 防抖动机制（避免频繁触发）
- 上下文控制（可取消监听）
- 多个回调函数

```go
// sysconf 增强的热重载
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

stop := cfg.WatchWithContext(ctx, func() {
    log.Println("配置已更新")
})

// 需要时取消监听
cancel()
```

### Q: 如何处理配置加密？

A: sysconf 内置 ChaCha20-Poly1305 加密：

```go
cfg, err := sysconf.New(
    sysconf.WithEncryption("your-password"),
)
```

Viper 没有内置加密功能，这是 sysconf 的额外优势。

### Q: 验证失败会怎样？

A: `Set()` 会返回错误：

```go
if err := cfg.Set("server.port", 70000); err != nil {
    log.Printf("配置无效: %v", err)
    // 输出: 配置无效: validator 'Web Server Configuration Validator' - 
    //        field 'server.port': port number must be between 1-65535
}
```

## 完整迁移示例

### 之前 (Viper)

```go
package main

import (
    "log"
    "github.com/spf13/viper"
    "github.com/spf13/cobra"
)

func main() {
    var rootCmd = &cobra.Command{
        Use: "myapp",
        Run: func(cmd *cobra.Command, args []string) {
            v := viper.New()
            v.SetConfigName("config")
            v.SetConfigType("yaml")
            v.AddConfigPath(".")
            
            v.SetEnvPrefix("MYAPP")
            v.AutomaticEnv()
            
            v.BindPFlag("host", cmd.Flags().Lookup("host"))
            v.BindPFlag("port", cmd.Flags().Lookup("port"))
            
            if err := v.ReadInConfig(); err != nil {
                log.Fatal(err)
            }
            
            host := v.GetString("host")
            port := v.GetInt("port")
            log.Printf("Server: %s:%d", host, port)
        },
    }
    
    rootCmd.Flags().String("host", "localhost", "Server host")
    rootCmd.Flags().Int("port", 8080, "Server port")
    
    rootCmd.Execute()
}
```

### 之后 (Sysconf)

```go
package main

import (
    "log"
    "github.com/darkit/sysconf"
    "github.com/spf13/cobra"
)

func main() {
    var rootCmd = &cobra.Command{
        Use: "myapp",
        Run: func(cmd *cobra.Command, args []string) {
            cfg, err := sysconf.New(
                sysconf.WithName("config"),
                sysconf.WithMode("yaml"),
                sysconf.WithPath("."),
                sysconf.WithEnv("MYAPP"),        // 替代 SetEnvPrefix + AutomaticEnv
                sysconf.WithPFlags(cmd.Flags()), // 替代 BindPFlag
            )
            if err != nil {
                log.Fatal(err)
            }
            
            // 使用泛型 API（可选但推荐）
            host := sysconf.GetAs[string](cfg, "host", "localhost")
            port := sysconf.GetAs[int](cfg, "port", 8080)
            log.Printf("Server: %s:%d", host, port)
        },
    }
    
    rootCmd.Flags().String("host", "localhost", "Server host")
    rootCmd.Flags().Int("port", 8080, "Server port")
    
    rootCmd.Execute()
}
```

## 获取帮助

如果您在迁移过程中遇到问题：

1. 查看 [GitHub Issues](https://github.com/darkit/sysconf/issues)
2. 发起 [Discussion](https://github.com/darkit/sysconf/discussions)
3. 阅读 [API 文档](https://pkg.go.dev/github.com/darkit/sysconf)

---

**文档版本**: 1.0  
**最后更新**: 2026-03-03
