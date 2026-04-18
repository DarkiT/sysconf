# Sysconf - Go配置管理库

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/sysconf.svg)](https://pkg.go.dev/github.com/darkit/sysconf)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/sysconf)](https://goreportcard.com/report/github.com/darkit/sysconf)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/sysconf/blob/master/LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/)

**Sysconf** 是一个高性能、线程安全的Go配置管理库，专为企业级应用设计。采用原子存储技术和智能验证系统，提供可靠的并发访问支持。

## ✨ 核心特性

### 🚀 性能与安全
- **⚡ 高性能**: 微秒级配置读取，毫秒级写入
- **🔒 线程安全**: 基于 `atomic.Value` + `sync.RWMutex` 实现并发安全
- **📊 并发支持**: 经过严格并发测试，支持高并发场景
- **💾 智能缓存**: 原子存储配合缓存机制，优化读取性能

### 🔧 配置管理
- **多格式支持**: YAML, JSON, TOML, Dotenv, ENV 等
- **类型安全**: 智能类型转换和泛型API，编译时类型检查
- **结构体映射**: 支持复杂嵌套结构和标签验证
- **默认值系统**: 灵活的默认值设置和回退机制

### 🛡️ 企业级特性
- **智能验证系统**: 字段级验证，30+种内置规则，简化调用逻辑
- **动态验证器匹配**: 消除硬编码，验证器自动推断支持字段
- **热重载**: 防抖动文件监控，支持配置实时更新
- **高级加密**: ChaCha20-Poly1305认证加密，支持自定义加密器

### 🌐 生态集成
- **环境变量**: 智能大小写匹配，支持前缀和嵌套结构
- **命令行集成**: 完整的 pflag/cobra 支持，企业级CLI应用友好
- **Viper兼容**: 完全兼容现有viper生态，无缝迁移

## 📦 安装

```bash
go get github.com/darkit/sysconf
```

**系统要求**: Go 1.24+

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/darkit/sysconf"
    "github.com/darkit/sysconf/validation"
)

func main() {
    // 创建高性能、线程安全的配置实例
    cfg, err := sysconf.New(
        sysconf.WithContent(defaultConfig),    // 默认配置内容
        sysconf.WithPath("configs"),           // 配置文件目录
        sysconf.WithName("app"),               // 配置文件名
        sysconf.WithMode("yaml"),              // 配置格式
    )
    if err != nil {
        log.Fatal("创建配置失败:", err)
    }

    // 类型安全的配置读取（完全线程安全）
    host := cfg.GetString("database.host", "localhost")
    port := cfg.GetInt("database.port", 5432)
    debug := cfg.GetBool("app.debug", false)
    timeout := cfg.GetDuration("database.timeout")
    
    log.Printf("数据库连接: %s:%d", host, port)
    
    // 高并发场景演示 - 无竞态条件
    go func() {
        for i := 0; i < 1000; i++ {
            cfg.Set(fmt.Sprintf("dynamic.key%d", i), fmt.Sprintf("value%d", i))
        }
    }()
    
    go func() {
        for i := 0; i < 1000; i++ {
            _ = cfg.GetString("database.host")
        }
    }()
    
    log.Println("✅ 高并发操作完成，无任何竞态条件")
}

const defaultConfig = `
app:
  name: "MyApp"
  version: "1.0.0"
  debug: false

database:
  host: "localhost"
  port: 5432
  timeout: "30s"

server:
  features: ["http", "grpc", "websocket"]
  ports: [8080, 8443]
`
```

### 企业级验证系统

```go
// 集成智能验证器系统
cfg, err := sysconf.New(
    sysconf.WithContent(defaultConfig),
    sysconf.WithValidators(
        validation.NewDatabaseValidator(),  // 数据库配置验证
        validation.NewWebServerValidator(), // Web服务器配置验证
        validation.NewRedisValidator(),     // Redis配置验证
    ),
)

// 配置设置时自动进行字段级验证
cfg.Set("server.port", 8080)     // ✅ 有效端口
cfg.Set("server.port", 70000)    // ❌ 被验证器拦截
cfg.Set("database.host", "localhost") // ✅ 有效主机名
```

### 类型安全的泛型API

```go
// 🆕 现代化泛型API，编译时类型安全
host := GetAs[string](cfg, "database.host", "localhost")
port := GetAs[int](cfg, "database.port", 5432)
timeout := GetAs[time.Duration](cfg, "timeout", 30*time.Second)
apiKey := MustGetAs[string](cfg, "api.secret_key") // 缺失或转换失败时 panic

// 泛型切片支持
features := GetSliceAs[string](cfg, "server.features")
ports := GetSliceAs[int](cfg, "server.ports")
```

#### MustGetAs[T] - 强制获取（新增）

获取配置值并转换为指定类型，失败时 panic。适用于强制必填配置项。

```go
apiKey := sysconf.MustGetAs[string](cfg, "api.secret_key")  // 缺失或转换失败时 panic
port := sysconf.MustGetAs[int](cfg, "server.port")
```

### 结构体映射

```go
// 定义配置结构体
type AppConfig struct {
    App struct {
        Name    string `config:"name" default:"MyApp" validate:"required,min=1"`
        Version string `config:"version" default:"1.0.0" validate:"required,semver"`
        Env     string `config:"env" default:"development" validate:"required,oneof=development test prod"`
    } `config:"app"`

    Database struct {
        Host     string        `config:"host" default:"localhost" validate:"required,hostname_rfc1123"`
        Port     int           `config:"port" default:"5432" validate:"required,min=1,max=65535"`
        Username string        `config:"username" default:"postgres" validate:"required,min=1"`
        Password string        `config:"password" validate:"required,min=1"`
        Timeout  time.Duration `config:"timeout" default:"30s" validate:"required"`
        MaxConns int           `config:"max_conns" default:"10" validate:"min=1,max=100"`
    } `config:"database"`
}

func main() {
    cfg, _ := sysconf.New(/* 配置选项 */)
    
    var config AppConfig
    if err := cfg.Unmarshal(&config); err != nil {
        log.Fatal("配置解析失败:", err)
    }
    
    // 类型安全的配置访问
    fmt.Printf("应用: %s v%s\n", config.App.Name, config.App.Version)
    fmt.Printf("数据库: %s:%d\n", config.Database.Host, config.Database.Port)
}
```

## 📊 性能特性

### 技术实现

**原子存储架构**:
```go
type Config struct {
    // 核心数据存储 - 使用atomic.Value实现无锁读取
    data atomic.Value // 存储map[string]any
    
    // 并发控制
    mu sync.RWMutex // 保护元数据和写操作
}

// 无锁高性能读取
func (c *Config) loadData() map[string]any {
    if data := c.data.Load(); data != nil {
        return data.(map[string]any)
    }
    return make(map[string]any)
}
```

**并发测试验证**:
- 支持多协程并发读写
- 通过 race detector 测试
- 稳定的性能表现

## 🛡️ 智能验证系统

### 字段级智能验证

支持精细化的字段级验证机制：

```go
// 🆕 字段级智能验证特性
func (c *Config) validateSingleField(key string, value any) error {
    // 只验证相关的验证器和字段
    for _, validator := range validators {
        if !c.validatorSupportsField(validator, key) {
            continue // 跳过不相关的验证器
        }
        
        // 执行单字段验证，跳过required检查避免级联失败
        if err := c.validateField(validator, key, value); err != nil {
            return err
        }
    }
}
```

### 动态验证器匹配

验证器自动推断支持字段：

```go
// 🆕 动态字段检查
func (c *Config) validatorSupportsField(validator ConfigValidator, key string) bool {
    if structValidator, ok := validator.(*validation.StructuredValidator); ok {
        supportedFields := structValidator.GetSupportedFields()
        for _, supportedField := range supportedFields {
            if supportedField == fieldGroup {
                return true
            }
        }
    }
    return false
}

// 验证器自动推断支持的字段
func (r *StructuredValidator) GetSupportedFields() []string {
    fieldPrefixes := make(map[string]bool)
    
    // 从规则中自动提取字段前缀
    for key := range r.rules {
        if prefix := extractFieldPrefix(key); prefix != "" {
            fieldPrefixes[prefix] = true
        }
    }
    
    return prefixes
}
```

### 预定义验证器

提供6种企业级预定义验证器，开箱即用：

```go
// 数据库配置验证器
validator := validation.NewDatabaseValidator()
// 验证：主机名、端口范围、用户名、密码、数据库类型等

// Web服务器验证器
validator := validation.NewWebServerValidator()  
// 验证：服务器配置、SSL设置、运行模式等

// Redis验证器
validator := validation.NewRedisValidator()
// 验证：Redis连接、数据库索引、地址列表等

// 邮件配置验证器
validator := validation.NewEmailValidator()
// 验证：SMTP配置、邮箱格式、认证设置等

// API配置验证器  
validator := validation.NewAPIValidator()
// 验证：API端点、认证密钥、超时设置等

// 日志配置验证器
validator := validation.NewLogValidator()
// 验证：日志级别、输出格式、文件路径等
```

### 支持的验证规则

**30+种内置验证规则**:
- **网络相关**: `email`, `url`, `ipv4`, `ipv6`, `hostname`, `port`
- **数据格式**: `json`, `uuid`, `base64`, `regex`, `alphanum` 
- **数值范围**: `range:1,100`, `length:5,20`, `min:1`, `max:100`
- **业务规则**: `creditcard`, `phonenumber`, `datetime`, `timezone`
- **枚举验证**: `enum:apple,banana,orange`
- **必填验证**: `required` (智能处理，避免级联失败)

## 🔐 高级加密功能

### ChaCha20-Poly1305 认证加密

内置高性能认证加密算法，提供企业级数据保护：

```go
// 轻量级加密（推荐入门）
cfg, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("app"),
    sysconf.WithEncryption("your-secret-password"), // 🔐 启用加密
    sysconf.WithContent(defaultConfig),
)

// 敏感配置自动加密存储
cfg.Set("database.password", "super-secret-password")
cfg.Set("api.secret_key", "sk-1234567890abcdef")

// 读取时自动解密
dbPassword := cfg.GetString("database.password")
apiKey := cfg.GetString("api.secret_key")
```

### 自定义加密器

支持企业级自定义加密需求：

```go
// 自定义加密器接口
type ConfigCrypto interface {
    Encrypt(data []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    IsEncrypted(data []byte) bool
}

// 使用自定义加密器
customCrypto := &YourAESGCMCrypto{key: []byte("your-key")}
cfg, err := sysconf.New(
    sysconf.WithEncryptionCrypto(customCrypto),
)
```

**加密算法特性**:
- ✅ **ChaCha20-Poly1305**: 高性能认证加密
- ✅ **抗侧信道攻击**: 移动设备友好
- ✅ **完整性验证**: AEAD提供机密性和完整性
- ✅ **性能优化**: 软件实现比AES更快更安全

## 🌐 环境变量与命令行集成

### 智能大小写匹配

支持多种环境变量格式，用户友好：

```go
// 启用智能大小写匹配
cfg, err := sysconf.New(
    sysconf.WithEnv("APP"),  // 自动启用智能匹配
)
```

**支持的环境变量格式**:
```bash
# 🆕 智能大小写匹配 - 全部支持
app_database_host=localhost    # ✅ 小写（用户友好）  
APP_DATABASE_HOST=localhost    # ✅ 大写格式
App_Database_Host=localhost    # ✅ 混合大小写
database_port=5432             # ✅ 无前缀小写
DATABASE_OPTIONS_SSL_MODE=require  # ✅ 大写格式
```

### Cobra/PFlag 完整集成

企业级CLI应用的完美选择：

```go
import (
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
    "github.com/darkit/sysconf"
)

func main() {
    // 创建cobra命令
    var rootCmd = &cobra.Command{
        Use: "myapp",
        Run: func(cmd *cobra.Command, args []string) {
            // 创建配置实例，集成pflag
            cfg, err := sysconf.New(
                sysconf.WithPath("configs"),
                sysconf.WithName("app"), 
                sysconf.WithPFlags(cmd.Flags()), // 🆕 完整pflag集成
                sysconf.WithEnv("MYAPP"),
            )
            
            // 命令行参数自动覆盖配置文件
            host := cfg.GetString("host") // 来自 --host 参数或配置文件
            port := cfg.GetInt("port")    // 来自 --port 参数或配置文件
        },
    }
    
    // 定义命令行参数
    rootCmd.Flags().String("host", "", "Database host")
    rootCmd.Flags().Int("port", 5432, "Database port")
    
    rootCmd.Execute()
}
```

**优先级顺序**: 命令行参数 > 环境变量 > 配置文件 > 默认值

## 🔄 配置热重载

防抖动的智能文件监控：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

stop := cfg.WatchWithContext(ctx, func() {
    log.Println("配置文件已更新")

    var newConfig AppConfig
    if err := cfg.Unmarshal(&newConfig); err != nil {
        log.Printf("配置重载失败: %v", err)
        return
    }

    updateApplication(&newConfig)
})

// 在需要时显式停止监听
stop()
```

**热重载特性**:
- ✅ **防抖机制**: 默认 200ms 内多次变更只触发一次（可通过 `WithWatchDebounce` 调整）
- ✅ **并发安全**: 配置更新期间服务不中断
- ✅ **错误恢复**: 配置验证失败时自动回滚
- ✅ **智能监控**: 只监控实际的文件写入操作
- ✅ **可控监听**: 使用 `WatchWithContext` 可在需要时取消监听

> 需要显式关闭热重载时，可调用 `cancel := cfg.WatchWithContext(ctx, callbacks...)` 并在退出流程中执行 `cancel()`。

## ⚙️ 调优选项

```go
cfg, err := sysconf.New(
    sysconf.WithWriteDebounceDelay(500*time.Millisecond), // 调整防抖写入延迟
    sysconf.WithWatchDebounce(200*time.Millisecond),      // 调整热重载防抖时间
    sysconf.WithCacheTiming(0, 100*time.Millisecond),  // 控制缓存预热与重建延迟
)
```

- **WithWriteDebounceDelay**: 设置防抖写入延迟，delay > 0 启用防抖，delay <= 0 立即写入。
- **WithWatchDebounce**: 设置配置文件监听防抖时间，减小可提高回调灵敏度。
- **WithCacheTiming**: 配置读取缓存的预热和重建间隔，避免固定魔术数字。
- **WithEnvOptions**: 启用 SmartCase 后环境变量键会被缓存，多种大小写/前缀只需解析一次。
- **防御性写入**: 对 map/slice 自动深拷贝，外部修改不会污染内部状态，可配合示例中的 `parent.child` 演示验证。

> 将延迟设为 0 或负值可禁用等待，实时刷新缓存或直接写入文件。

## 🛡 防御性写入机制

- `Set` 操作会对 map、slice 做深拷贝，防止调用方后续修改原始数据污染内部状态。
- 嵌套结构会自动展开为扁平键，配合缓存失效保证每次读取都是一致数据。
- 示例 `examples/main.go` 展示了设置 `parent.child` 后继续修改原始 map，读取结果仍保持 "原始值"。

## 📝 配置文件格式

### YAML (推荐)

```yaml
# 应用基础配置
app:
  name: "MyApp"
  version: "1.0.0"
  env: "production"

# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080
  timeout: "30s"
  features:
    - http
    - grpc
    - websocket

# 数据库配置  
database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "secret123"  # 可通过环境变量覆盖：export APP_DATABASE_PASSWORD=newpassword
  timeout: "30s"
  max_conns: 10
  options:
    ssl_mode: "require"
    timezone: "UTC"
```

### JSON

```json
{
  "app": {
    "name": "MyApp",
    "version": "1.0.0",
    "env": "production"
  },
  "database": {
    "host": "localhost", 
    "port": 5432,
    "timeout": "30s"
  }
}
```

### TOML

```toml
[app]
name = "MyApp"
version = "1.0.0"
env = "production"

[database]
host = "localhost"
port = 5432
timeout = "30s"

[server]
features = ["http", "grpc", "websocket"]
```

### Dotenv

```bash
APP_NAME=MyApp
APP_VERSION=1.0.0
DATABASE_HOST=localhost
DATABASE_PORT=5432
```

## 📚 详细API指南

### 基础类型获取

```go
// 字符串类型（支持默认值）
host := cfg.GetString("database.host", "localhost")
host := cfg.GetStringPath("database", "host")  // 路径片段形式

// 数值类型
port := cfg.GetInt("database.port", 5432)
weight := cfg.GetFloat("metrics.weight", 0.95)

// 布尔类型（支持多种格式）
debug := cfg.GetBool("app.debug", true)
// 🆕 支持: true/false, "yes"/"no", "on"/"off", "1"/"0", 数字0/非零

// 通用类型
value := cfg.Get("any.key", "default_value")
```

### 时间和持续时间

```go
// 时间持续时间（支持 "30s", "5m", "1h" 等）
timeout := cfg.GetDuration("database.timeout")

// 时间类型
timestamp := cfg.GetTime("app.created_at")
```

### 切片类型

```go
// 字符串切片
features := cfg.GetStringSlice("server.features")

// 数值切片
ports := cfg.GetIntSlice("server.ports")  
weights := cfg.GetFloatSlice("analytics.weights")

// 布尔切片
flags := cfg.GetBoolSlice("feature.flags")
```

### 映射类型

```go
// 通用映射
options := cfg.GetStringMap("database.options")  // map[string]interface{}

// 字符串映射
params := cfg.GetStringMapString("http.headers")  // map[string]string
```

### 泛型API (推荐)

```go
// 🆕 类型安全的泛型获取
host := GetAs[string](cfg, "database.host", "localhost")
port := GetAs[int](cfg, "database.port", 5432)
timeout := GetAs[time.Duration](cfg, "timeout", 30*time.Second)

// 🆕 泛型切片
features := GetSliceAs[string](cfg, "server.features")
ports := GetSliceAs[int](cfg, "server.ports")

// 🆕 必须存在的配置（不存在则panic）
apiKey := MustGetAs[string](cfg, "api.secret_key")

// 🆕 支持多个fallback键
port := GetWithFallback[int](cfg, "server.port", "app.port", "port")
```

## 🔧 高级配置选项

### 配置选项详解

```go
cfg, err := sysconf.New(
    // 基础选项
    sysconf.WithPath("configs"),              // 配置文件目录
    sysconf.WithMode("yaml"),                 // 配置格式
    sysconf.WithName("app"),                  // 配置文件名
    
    // 默认配置
    sysconf.WithContent(defaultConfig),       // 默认配置内容
    
    // 环境变量配置
    sysconf.WithEnv("APP"),                   // 便利函数：启用智能大小写匹配
    
    // 或完整配置（高级用法）
    sysconf.WithEnvOptions(sysconf.EnvOptions{
        Prefix:    "APP",     // 环境变量前缀
        Enabled:   true,      // 启用环境变量覆盖
        SmartCase: true,      // 启用智能大小写匹配
    }),
    
    // 验证器配置
    sysconf.WithValidators(
        validation.NewDatabaseValidator(),
        validation.NewWebServerValidator(),
    ),
    
    // 加密配置
    sysconf.WithEncryption("secret-password"), // 启用加密
    
    // 命令行集成
    sysconf.WithPFlags(cmd.Flags()),           // Cobra集成
)
```

### 配置更新

```go
// 更新单个配置项（自动验证）
if err := cfg.Set("database.host", "new-host"); err != nil {
    log.Printf("配置更新失败: %v", err)
}

// 🆕 批量更新（推荐，减少验证和写入开销）
err := cfg.SetMultiple(map[string]any{
    "database.host": "new-host",
    "database.port": 5433,
    "server.debug":  true,
})
if err != nil {
    log.Printf("批量更新失败: %v", err)
}
```

**配置更新特性**:
- ✅ **批量更新**: `SetMultiple` 一次性设置多个配置项
- ✅ **3秒写入延迟**: 合并短时间内的多次更新
- ✅ **智能验证**: 字段级验证防止无效值
- ✅ **原子性写入**: 避免配置文件损坏  
- ✅ **自动备份**: 变更前自动备份原配置

### 验证器管理

```go
// 动态添加验证器
cfg.AddValidator(validation.NewEmailValidator())

// 函数式验证器
cfg.AddValidateFunc(func(config map[string]any) error {
    // 自定义验证逻辑
    if env := config["app"].(map[string]any)["env"]; env == "production" {
        // 生产环境特殊验证
        if ssl, exists := config["server"].(map[string]any)["ssl"]; !exists || ssl != true {
            return fmt.Errorf("生产环境必须启用SSL")
        }
    }
    return nil
})

// 获取当前验证器
validators := cfg.GetValidators()
fmt.Printf("当前验证器数量: %d\n", len(validators))

// 清除所有验证器
cfg.ClearValidators()
```

## 🧪 测试和调试

### 单元测试支持

```go
func TestConfig(t *testing.T) {
    // 创建测试配置（纯内存模式）
    cfg, err := sysconf.New(
        sysconf.WithContent(`
app:
  name: "TestApp"
  debug: true
database:
  host: "localhost"
  port: 5432
`),
    )
    require.NoError(t, err)
    
    // 测试配置读取
    assert.Equal(t, "TestApp", cfg.GetString("app.name"))
    assert.True(t, cfg.GetBool("app.debug"))
    assert.Equal(t, 5432, cfg.GetInt("database.port"))
    
    // 测试并发安全
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            cfg.Set(fmt.Sprintf("test.key%d", id), fmt.Sprintf("value%d", id))
            _ = cfg.GetString("app.name")
        }(i)
    }
    wg.Wait()
    
    // 验证并发操作结果
    keys := cfg.Keys()
    assert.GreaterOrEqual(t, len(keys), 100)
}
```

### 性能基准测试

```go
func BenchmarkConfig(b *testing.B) {
    cfg, _ := sysconf.New(sysconf.WithContent(testConfig))
    
    b.Run("SequentialReads", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = cfg.GetString("app.name")
        }
    })
    
    b.Run("ConcurrentReads", func(b *testing.B) {
        b.RunParallel(func(pb *testing.PB) {
            for pb.Next() {
                _ = cfg.GetString("app.name")
            }
        })
    })
    
    b.Run("MixedOperations", func(b *testing.B) {
        b.RunParallel(func(pb *testing.PB) {
            i := 0
            for pb.Next() {
                if i%10 == 0 {
                    cfg.Set(fmt.Sprintf("bench.key%d", i), i)
                } else {
                    _ = cfg.GetString("app.name")  
                }
                i++
            }
        })
    })
}
```

### 调试技巧

```go
// 启用调试日志
cfg, err := sysconf.New(
    sysconf.WithLogLevel("debug"),  // 开启详细日志
    // ... 其他选项
)

// 导出当前所有配置
allSettings := cfg.AllSettings()
fmt.Printf("当前配置: %+v\n", allSettings)

// 检查配置键是否存在
if !cfg.IsSet("some.key") {
    log.Println("配置键不存在:", "some.key")
}

// 获取所有配置键
keys := cfg.Keys()
log.Printf("配置键列表: %v", keys)
```

## 💡 最佳实践

### 1. 配置结构设计

```go
// ✅ 推荐：按模块组织配置
type Config struct {
    App      AppConfig      `config:"app"`
    Database DatabaseConfig `config:"database"`
    Server   ServerConfig   `config:"server"`
    Cache    CacheConfig    `config:"cache"`
}

// ✅ 推荐：合理使用默认值和验证
type DatabaseConfig struct {
    Host     string `config:"host" default:"localhost" validate:"hostname"`
    Port     int    `config:"port" default:"5432" validate:"min=1,max=65535"`
    Password string `config:"password" validate:"required,min=8"`  // 敏感信息无默认值
}
```

### 2. 环境区分

```go
// ✅ 推荐：按环境组织配置文件
// configs/
//   ├── base.yaml      # 基础配置
//   ├── dev.yaml       # 开发环境
//   ├── test.yaml      # 测试环境
//   └── prod.yaml      # 生产环境

env := os.Getenv("APP_ENV")
if env == "" {
    env = "dev"
}

cfg, err := sysconf.New(
    sysconf.WithName(env),
    sysconf.WithEnv(strings.ToUpper(env)),
    // ...
)
```

### 3. 错误处理

```go
// ✅ 推荐：优雅的错误处理
if err := cfg.Unmarshal(&config); err != nil {
    log.Printf("配置解析失败: %v", err)
    // 使用默认配置继续运行
    config = getDefaultConfig()
}
```

### 4. 配置热重载

```go
// ✅ 推荐：安全的热重载
type Application struct {
    config atomic.Value  // 线程安全的配置存储
}

func (app *Application) watchConfig(cfg *sysconf.Config) {
    cfg.Watch(func() {
        var newConfig AppConfig
        if err := cfg.Unmarshal(&newConfig); err != nil {
            log.Printf("热重载失败: %v", err)
            return
        }
        
        // 原子性更新配置
        app.config.Store(&newConfig)
        log.Println("配置热重载成功")
    })
}
```

### 5. 加密配置管理

```go
// ✅ 推荐：分环境的加密配置
func createConfig(env string) (*sysconf.Config, error) {
    var encryptionPassword string
    
    switch env {
    case "production":
        // 生产环境：从密钥管理系统获取
        encryptionPassword = getFromKeyVault("CONFIG_ENCRYPTION_KEY")
    case "staging":
        // 测试环境：从环境变量获取
        encryptionPassword = os.Getenv("STAGING_CONFIG_PASSWORD")
    case "development":
        // 开发环境：使用默认密码或不加密
        encryptionPassword = ""  // 开发环境不加密
    }
    
    options := []sysconf.Option{
        sysconf.WithPath("configs"),
        sysconf.WithName(fmt.Sprintf("app-%s", env)),
        sysconf.WithContent(getDefaultConfig(env)),
    }
    
    // 只在有密码时启用加密
    if encryptionPassword != "" {
        options = append(options, sysconf.WithEncryption(encryptionPassword))
    }
    
    return sysconf.New(options...)
}
```

## 🛠️ 使用指南

### 配置文件加载

```go
// 使用WithContent提供默认配置
cfg, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("app"),
    sysconf.WithContent(defaultConfig),  // 提供默认配置
)
```

### 字段级验证

```go
// 字段级验证特性
err := cfg.Set("server.port", 8080)
// 仅验证 server.port 相关规则
```

### 环境变量配置

```go
// 使用智能大小写匹配
cfg, err := sysconf.New(
    sysconf.WithEnv("APP"),  // 支持各种大小写格式
)

// 支持的环境变量格式
// app_database_host=localhost    ✅ 小写
// APP_DATABASE_HOST=localhost    ✅ 大写  
// App_Database_Host=localhost    ✅ 混合大小写
```

### 性能优化建议

```go
// 使用环境变量前缀优化性能
cfg, err := sysconf.New(
    sysconf.WithEnv("MYAPP"),  // 前缀过滤，提升性能
)

// 性能优化特性：
// - 环境变量数 > 300 时自动启用优化策略
// - 处理时间 > 100ms 时提供性能建议
// - 智能批处理和时间保护机制
```

## 🔮 路线图

### 已完成 ✅

- **线程安全**: 基于atomic.Value的并发安全架构
- **性能优化**: 微秒级读取，毫秒级写入
- **智能验证系统**: 字段级验证，支持灵活配置
- **动态验证器匹配**: 自动推断支持字段
- **企业级加密**: ChaCha20-Poly1305认证加密
- **智能环境变量**: 支持多种大小写格式
- **完整生态集成**: viper/cobra/pflag无缝兼容

## 🤝 贡献指南

我们欢迎各种形式的贡献！

### 开发环境

```bash
# 克隆代码
git clone https://github.com/darkit/sysconf.git
cd sysconf

# 安装依赖
go mod download

# 运行测试
go test ./...

# 运行性能基准测试
go test -bench=. ./...

# 运行示例
cd examples
go run .
```

### 提交规范

- 🐛 **bug**: 修复bug
- ✨ **feat**: 新功能  
- 📚 **docs**: 文档更新
- 🎨 **style**: 代码格式
- ♻️ **refactor**: 重构
- ⚡ **perf**: 性能优化
- ✅ **test**: 测试相关

## 📊 项目状态

- ✅ **线程安全**: 统一Config架构，通过并发测试
- ✅ **高性能**: 优化的读写性能，支持高并发场景
- ✅ **智能验证**: 字段级验证系统
- ✅ **动态匹配**: 验证器自动推断
- ✅ **企业级特性**: 加密、验证、热重载完整支持
- ✅ **稳定性**: 生产就绪，已通过大规模测试
- ✅ **兼容性**: Go 1.23+ 支持，向后兼容现有代码
- ✅ **文档**: 完整的API文档和最佳实践指南
- ✅ **测试**: 完善的测试覆盖，包含并发和性能基准测试

## 📄 许可证

MIT License - 查看 [LICENSE](LICENSE) 文件了解详情。

---

<div align="center">

**如果这个项目对您有帮助，请给我们一个 ⭐️**

[GitHub](https://github.com/darkit/sysconf) • [API文档](https://pkg.go.dev/github.com/darkit/sysconf) • [验证器文档](validation/README.md) • [加密示例](examples/cmd/encryption_demo/) • [反馈](https://github.com/darkit/sysconf/issues)

</div>
