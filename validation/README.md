# Sysconf 验证器系统使用指南

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/dl/)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/sysconf/blob/master/LICENSE)

**Sysconf 验证器系统** 是一个企业级的配置验证框架，提供30+种内置验证规则、预定义验证器和灵活的自定义验证机制，确保配置数据的正确性和安全性。

## 🎯 核心特性

### ✨ 验证器类型
- **🔧 预定义验证器**: 针对常见场景的即用型验证器
- **📝 规则验证器**: 基于规则引擎的灵活验证
- **🔗 复合验证器**: 组合多个验证器的强大验证
- **⚡ 函数式验证器**: 支持自定义验证逻辑

### 🛡️ 验证规则
- **📊 数据类型**: string, number, boolean, array, object
- **🌐 网络相关**: email, url, ipv4, ipv6, hostname, port
- **📐 数值范围**: range, min, max, length
- **🔒 格式验证**: uuid, json, base64, regex, alphanum
- **📅 时间相关**: datetime, timezone
- **💳 业务规则**: creditcard, phonenumber
- **🎚️ 枚举验证**: enum, oneof

### 🚀 高级功能
- **动态验证器管理**: 运行时添加/移除验证器
- **嵌套配置验证**: 支持深层次配置结构验证
- **错误聚合**: 收集并报告所有验证错误
- **性能优化**: 高效的验证执行引擎

## 📦 快速开始

### 基础使用

```go
package main

import (
    "log"
    "github.com/darkit/sysconf"
    "github.com/darkit/sysconf/validation"
)

func main() {
    // 创建配置实例，集成验证器
    cfg, err := sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName("app"),
        sysconf.WithMode("yaml"),
        // 🆕 添加预定义验证器
        sysconf.WithValidators(
            validation.NewDatabaseValidator(),  // 数据库配置验证
            validation.NewWebServerValidator(), // Web服务器配置验证
            validation.NewRedisValidator(),     // Redis配置验证
        ),
    )
    if err != nil {
        log.Fatal("创建配置失败:", err)
    }

    // 配置值会自动进行验证
    cfg.Set("server.port", 8080)     // ✅ 有效端口
    cfg.Set("server.port", 70000)    // ❌ 被验证器拦截
    cfg.Set("database.host", "localhost") // ✅ 有效主机名
}
```

## 🔧 预定义验证器

### 1. 数据库验证器 (DatabaseValidator)

验证数据库连接相关的配置项：

```go
validator := validation.NewDatabaseValidator()

// 验证的配置项:
// - database.host      : 主机名验证
// - database.port      : 端口范围验证 (1-65535)
// - database.username  : 必填用户名
// - database.password  : 必填密码
// - database.type      : 数据库类型 (mysql,postgresql,sqlite,mongodb)
// - database.database  : 数据库名称
// - database.max_conns : 连接数范围 (1-100)
// - database.timeout   : 超时设置验证
```

**示例配置:**
```yaml
database:
  host: "localhost"          # hostname验证
  port: 5432                 # port验证 (1-65535)
  username: "postgres"       # required验证
  password: "secret123"      # required验证
  database: "myapp"          # required验证
  type: "postgresql"         # enum验证
  max_conns: 10             # range验证 (1-100)
  timeout: "30s"            # 超时格式验证
```

### 2. Web服务器验证器 (WebServerValidator)

验证Web服务器相关配置：

```go
validator := validation.NewWebServerValidator()

// 验证的配置项:
// - server.host    : 主机名验证
// - server.port    : 端口范围验证
// - server.mode    : 运行模式 (development,production,testing)
// - server.timeout : 超时设置验证
// - server.ssl.*   : SSL配置验证
```

**示例配置:**
```yaml
server:
  host: "0.0.0.0"           # hostname验证
  port: 8080                # port验证
  mode: "production"        # enum验证
  timeout: "30s"            # 超时验证
  ssl:
    enabled: true           # boolean验证
    cert_file: "/path/to/cert.pem"  # required验证
    key_file: "/path/to/key.pem"    # required验证
```

### 3. Redis验证器 (RedisValidator)

验证Redis缓存配置：

```go
validator := validation.NewRedisValidator()

// 验证的配置项:
// - redis.host      : 主机名验证
// - redis.port      : 端口验证
// - redis.db        : 数据库索引 (0-15)
// - redis.password  : 密码验证（可选）
// - redis.addresses : 地址列表验证
// - redis.timeout   : 超时验证
```

### 4. 日志验证器 (LogValidator)

验证日志配置：

```go
validator := validation.NewLogValidator()

// 验证的配置项:
// - logging.level  : 日志级别 (debug,info,warn,error,fatal)
// - logging.format : 日志格式 (json,text)
// - logging.path   : 日志路径验证
```

### 5. 邮件验证器 (EmailValidator)

验证邮件发送配置：

```go
validator := validation.NewEmailValidator()

// 验证的配置项:
// - email.smtp.host     : SMTP主机名验证
// - email.smtp.port     : SMTP端口验证
// - email.smtp.username : 邮箱格式验证
// - email.smtp.password : 必填密码验证
// - email.from          : 发件人邮箱验证
```

### 6. API验证器 (APIValidator)

验证API接口配置：

```go
validator := validation.NewAPIValidator()

// 验证的配置项:
// - api.base_url                    : URL格式验证
// - api.timeout                     : 超时范围 (1-300秒)
// - api.rate_limit.enabled          : 布尔值验证
// - api.rate_limit.requests_per_minute : 范围验证 (1-10000)
// - api.auth.api_key                : 必填API密钥
// - api.auth.jwt.*                  : JWT配置验证
```

## 📝 规则验证器

### 创建自定义规则验证器

```go
// 创建业务逻辑验证器
businessValidator := validation.NewRuleValidator("业务配置验证器")

// 添加结构化规则
businessValidator.AddRule("company.name", validation.Required("公司名称不能为空"))
businessValidator.AddRule("company.tax_id", validation.Pattern(`^\d{18}$`, "税务登记号必须是18位数字"))

// 添加字符串规则
businessValidator.AddStringRule("company.industry", "enum:technology,finance,healthcare,education")
businessValidator.AddStringRule("company.employee_count", "range:1,10000")
businessValidator.AddStringRule("company.email", "email")
businessValidator.AddStringRule("company.website", "url")

// 应用验证器
cfg.AddValidator(businessValidator)
```

### 支持的字符串规则

#### 基础验证
```go
"required"              // 必填字段
"string"                // 字符串类型
"number"                // 数字类型
```

#### 网络相关
```go
"email"                 // 邮箱格式
"url"                   // URL格式
"ipv4"                  // IPv4地址
"ipv6"                  // IPv6地址
"hostname"              // 主机名
"port"                  // 端口号 (1-65535)
```

#### 数值范围
```go
"range:1,100"           // 数值范围
"length:5,20"           // 字符串长度范围
```

#### 格式验证
```go
"regex:^[A-Z][a-z]+$"   // 正则表达式
"enum:apple,banana,orange" // 枚举值
"alphanum"              // 字母数字
"uuid"                  // UUID格式
"json"                  // JSON格式
"base64"                // Base64编码
```

#### 时间相关
```go
"datetime"              // 日期时间格式
"timezone"              // 时区验证
```

#### 业务规则
```go
"creditcard"            // 信用卡号
"phonenumber"           // 电话号码
```

### 结构化规则API

```go
validator := validation.NewRuleValidator("结构化规则示例")

// 基础规则
validator.AddRule("user.name", validation.Required("用户名不能为空"))
validator.AddRule("user.age", validation.Range("18", "65", "年龄必须在18-65岁之间"))
validator.AddRule("user.email", validation.Pattern(`^[^@]+@[^@]+\.[^@]+$`, "邮箱格式不正确"))

// 枚举规则
validator.AddRule("user.role", validation.Enum("admin,user,guest", "角色必须是admin、user或guest之一"))

// 长度规则
validator.AddRule("user.password", validation.Length("8", "密码长度必须是8位"))
```

## 🔗 复合验证器

### 创建复合验证器

```go
// 创建企业级应用验证器
enterpriseValidator := validation.NewCompositeValidator("企业级应用验证器",
    validation.NewDatabaseValidator(),  // 数据库验证
    validation.NewWebServerValidator(), // Web服务器验证
    validation.NewRedisValidator(),     // Redis验证
    validation.NewEmailValidator(),     // 邮件验证
    validation.NewAPIValidator(),       // API验证
)

// 添加到配置
cfg.AddValidator(enterpriseValidator)

// 获取子验证器信息
subValidators := enterpriseValidator.GetValidators()
fmt.Printf("包含 %d 个子验证器\n", len(subValidators))
```

### 预定义复合验证器

```go
// 通用验证器（包含常用验证器）
commonValidator := validation.NewCommonValidator()

// 最小化验证器（仅包含基础验证器）
minimalValidator := validation.NewMinimalValidator()
```

## ⚡ 函数式验证器

### 创建函数式验证器

```go
// 自定义业务逻辑验证
cfg.AddValidateFunc(func(config map[string]any) error {
    // 检查数据库连接配置的一致性
    if dbConfig, exists := config["database"].(map[string]any); exists {
        if dbType, ok := dbConfig["type"].(string); ok && dbType == "mysql" {
            if port, ok := dbConfig["port"].(int); ok && port != 3306 {
                return fmt.Errorf("MySQL数据库应使用默认端口3306")
            }
        }
    }
    return nil
})

// 环境特定验证
cfg.AddValidateFunc(func(config map[string]any) error {
    if appConfig, exists := config["app"].(map[string]any); exists {
        if env, ok := appConfig["env"].(string); ok && env == "production" {
            // 生产环境必须启用SSL
            if serverConfig, exists := config["server"].(map[string]any); exists {
                if sslConfig, exists := serverConfig["ssl"].(map[string]any); exists {
                    if enabled, ok := sslConfig["enabled"].(bool); !ok || !enabled {
                        return fmt.Errorf("生产环境必须启用SSL")
                    }
                }
            }
        }
    }
    return nil
})
```

## 🎛️ 动态验证器管理

### 运行时管理验证器

```go
// 获取当前验证器列表
validators := cfg.GetValidators()
fmt.Printf("当前验证器数量: %d\n", len(validators))

// 动态添加验证器
tempValidator := validation.NewRuleValidator("临时验证器")
tempValidator.AddStringRule("temp.value", "required")
cfg.AddValidator(tempValidator)

// 清除所有验证器
cfg.ClearValidators()

// 重新添加必要的验证器
cfg.AddValidator(validation.NewDatabaseValidator())
cfg.AddValidator(validation.NewWebServerValidator())
```

### 条件验证器

```go
// 根据环境动态添加验证器
env := cfg.GetString("app.env", "development")
switch env {
case "production":
    cfg.AddValidator(validation.NewEmailValidator())  // 生产环境需要邮件配置
    cfg.AddValidator(validation.NewAPIValidator())    // 生产环境需要API配置
case "development":
    cfg.AddValidator(validation.NewMinimalValidator()) // 开发环境使用最小验证
}
```

## 🔧 高级用法

### 1. 嵌套配置验证

```go
// 验证深层嵌套的配置
validator := validation.NewRuleValidator("嵌套配置验证器")

// 支持点号分隔的深层路径
validator.AddStringRule("app.security.jwt.secret", "required")
validator.AddStringRule("app.security.jwt.expires_in", "required")
validator.AddStringRule("app.database.connections.read.host", "hostname")
validator.AddStringRule("app.database.connections.write.host", "hostname")

cfg.AddValidator(validator)
```

### 2. 条件验证

```go
// 基于配置值的条件验证
cfg.AddValidateFunc(func(config map[string]any) error {
    // 如果启用了缓存，必须配置Redis
    if cacheEnabled := getNestedValue(config, "app.cache.enabled"); cacheEnabled == true {
        if redisHost := getNestedValue(config, "redis.host"); redisHost == nil || redisHost == "" {
            return fmt.Errorf("启用缓存时必须配置Redis主机地址")
        }
    }
    return nil
})
```

### 3. 批量验证

```go
// 批量设置配置并验证
updates := map[string]interface{}{
    "server.host": "api.example.com",
    "server.port": 443,
    "database.host": "db.example.com",
    "database.port": 5432,
    "redis.host": "cache.example.com",
}

// 所有更新都会经过验证器验证
for key, value := range updates {
    if err := cfg.Set(key, value); err != nil {
        log.Printf("设置 %s 失败: %v", key, err)
    }
}
```

### 4. 自定义验证规则

```go
// 注册自定义验证规则
validation.RegisterValidator("chinese_phone", func(value any, params string) (bool, string) {
    phone, ok := value.(string)
    if !ok {
        return false, "手机号必须是字符串类型"
    }
    
    // 中国手机号验证
    matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
    if !matched {
        return false, "无效的中国手机号格式"
    }
    return true, ""
})

// 使用自定义规则
validator := validation.NewRuleValidator("中国本地化验证器")
validator.AddStringRule("user.phone", "chinese_phone")
cfg.AddValidator(validator)
```

## 🧪 测试支持

### 验证器单元测试

```go
func TestValidators(t *testing.T) {
    // 创建测试配置
    cfg, err := sysconf.New(
        sysconf.WithContent(`
database:
  host: "localhost"
  port: 5432
  username: "test"
  password: "test123"
  type: "postgresql"
`),
        sysconf.WithValidators(
            validation.NewDatabaseValidator(),
        ),
    )
    require.NoError(t, err)

    // 测试有效配置
    err = cfg.Set("database.port", 3306)
    assert.NoError(t, err)

    // 测试无效配置
    err = cfg.Set("database.port", 70000)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "port")

    // 测试配置解析
    var config struct {
        Database struct {
            Host string `config:"host"`
            Port int    `config:"port"`
        } `config:"database"`
    }
    
    err = cfg.Unmarshal(&config)
    assert.NoError(t, err)
    assert.Equal(t, "localhost", config.Database.Host)
}
```

### 验证器性能测试

```go
func BenchmarkValidation(b *testing.B) {
    validator := validation.NewDatabaseValidator()
    config := map[string]any{
        "database": map[string]any{
            "host":     "localhost",
            "port":     5432,
            "username": "test",
            "password": "test123",
            "type":     "postgresql",
        },
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        err := validator.Validate(config)
        if err != nil {
            b.Fatalf("验证失败: %v", err)
        }
    }
}
```

## 🚨 错误处理

### 验证错误类型

```go
// 尝试设置无效配置
if err := cfg.Set("server.port", 70000); err != nil {
    // 验证错误包含详细信息
    fmt.Printf("验证失败: %v\n", err)
    // 输出: validator 'Web Server Configuration Validator' - field 'server.port': port number must be between 1-65535
}

// 结构体验证错误
var config AppConfig
if err := cfg.Unmarshal(&config); err != nil {
    fmt.Printf("配置解析失败: %v\n", err)
}
```

### 错误聚合

验证器会聚合多个验证错误，提供完整的错误信息：

```go
// 同时设置多个无效值
cfg.Set("server.port", 70000)      // 端口错误
cfg.Set("database.host", "")       // 主机名错误
cfg.Set("redis.db", 20)           // Redis DB索引错误

// 获取所有验证错误
errors := cfg.GetValidationErrors() // 自定义方法，返回所有错误
for _, err := range errors {
    fmt.Printf("验证错误: %v\n", err)
}
```

## 🎯 最佳实践

### 1. 验证器组织

```go
// ✅ 推荐：按模块组织验证器
func setupValidators(cfg *sysconf.Config, env string) {
    // 基础验证器（所有环境）
    cfg.AddValidator(validation.NewDatabaseValidator())
    cfg.AddValidator(validation.NewWebServerValidator())
    
    // 环境特定验证器
    switch env {
    case "production":
        cfg.AddValidator(validation.NewEmailValidator())
        cfg.AddValidator(validation.NewAPIValidator())
    case "development":
        cfg.AddValidator(validation.NewLogValidator())
    }
    
    // 业务逻辑验证器
    cfg.AddValidator(createBusinessValidator())
}
```

### 2. 错误处理策略

```go
// ✅ 推荐：优雅的错误处理
func loadConfig() (*AppConfig, error) {
    cfg, err := sysconf.New(/* 选项 */)
    if err != nil {
        return nil, fmt.Errorf("创建配置失败: %w", err)
    }
    
    var config AppConfig
    if err := cfg.Unmarshal(&config); err != nil {
        // 配置验证失败，使用默认配置
        log.Printf("配置验证失败，使用默认配置: %v", err)
        return getDefaultConfig(), nil
    }
    
    return &config, nil
}
```

### 3. 性能优化

```go
// ✅ 推荐：验证器复用
var (
    commonValidators = []validation.Validator{
        validation.NewDatabaseValidator(),
        validation.NewWebServerValidator(),
        validation.NewRedisValidator(),
    }
)

func createConfig() *sysconf.Config {
    cfg, _ := sysconf.New(
        sysconf.WithValidators(commonValidators...),
    )
    return cfg
}
```

### 4. 配置分层验证

```go
// ✅ 推荐：分层验证策略
func createLayeredValidation() *validation.CompositeValidator {
    return validation.NewCompositeValidator("分层验证器",
        // 第一层：基础格式验证
        validation.NewRuleValidator("格式验证器").
            AddStringRule("*.port", "port").
            AddStringRule("*.host", "hostname").
            AddStringRule("*.email", "email"),
        
        // 第二层：业务逻辑验证
        createBusinessValidator(),
        
        // 第三层：环境特定验证
        createEnvironmentValidator(),
    )
}
```

## 📚 API 参考

### 验证器接口

```go
type Validator interface {
    Validate(config map[string]any) error
    GetName() string
}
```

### 主要类型

```go
// 规则验证器
type StructuredValidator struct {
    // 添加规则
    AddRule(key string, rule ValidationRule) *StructuredValidator
    AddStringRule(key string, rule string) *StructuredValidator
    AddRules(key string, rules ...ValidationRule) *StructuredValidator
    AddStringRules(key string, rules ...string) *StructuredValidator
}

// 复合验证器
type CompositeValidator struct {
    // 添加验证器
    AddValidator(validator Validator) *CompositeValidator
    GetValidators() []Validator
}

// 函数式验证器
type ValidatorFunc func(config map[string]any) error
```

### 配置方法

```go
// 验证器管理
cfg.AddValidator(validator Validator)
cfg.AddValidateFunc(func(config map[string]any) error)
cfg.GetValidators() []Validator
cfg.ClearValidators()

// 验证相关配置选项
sysconf.WithValidators(validators ...Validator)
```

## 🔗 相关资源

- **主文档**: [README.md](../README.md)
- **API文档**: [pkg.go.dev](https://pkg.go.dev/github.com/darkit/sysconf)
- **示例代码**: [examples/](../examples/)
- **测试用例**: [*_test.go](../validation/)

---

<div align="center">

**验证器系统让您的配置更安全、更可靠**

[🏠 返回主文档](README.md) • [🐛 问题反馈](https://github.com/darkit/sysconf/issues) • [💡 功能建议](https://github.com/darkit/sysconf/discussions)

</div> 