---
name: sysconf-configuring-validation
description: 配置 sysconf 的智能验证系统，包括 30+ 内置验证规则和 6 种预定义验证器。当用户需要添加配置验证、创建自定义验证器、使用字段级验证或实现业务规则验证时使用。
version: "1.0.0"
---

# Sysconf 配置验证系统

此技能帮助你在 sysconf 项目中配置和使用强大的验证系统。

## 概述

Sysconf 提供企业级配置验证功能：

- **30+ 内置验证规则**: 网络、格式、数值、业务规则等
- **6 种预定义验证器**: 数据库、Web 服务器、Redis、邮件、API、日志
- **字段级智能验证**: 只验证相关字段，避免级联失败
- **动态验证器匹配**: 自动推断验证器支持的字段范围
- **自定义验证器**: 灵活的验证器接口

## 何时使用此技能

使用此技能当你需要：

- 确保配置值的有效性和合法性
- 防止无效配置导致应用崩溃
- 实现业务规则验证
- 创建自定义验证逻辑
- 使用预定义验证器快速开始
- 实现字段级精细验证

## 前置要求

- 已完成 sysconf 基础集成（参考 `integrating-sysconf` 技能）
- 导入验证包: `import "github.com/darkit/sysconf/validation"`

## 使用指南

### 步骤 1: 使用预定义验证器（推荐快速开始）

Sysconf 提供 6 种开箱即用的验证器：

```go
import (
    "github.com/darkit/sysconf"
    "github.com/darkit/sysconf/validation"
)

cfg, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("app"),
    sysconf.WithValidators(
        validation.NewDatabaseValidator(),  // 数据库配置验证
        validation.NewWebServerValidator(), // Web 服务器验证
        validation.NewRedisValidator(),     // Redis 配置验证
        validation.NewLogValidator(),       // 日志配置验证
        validation.NewEmailValidator(),     // 邮件配置验证
        validation.NewAPIValidator(),       // API 配置验证
    ),
)
```

**预定义验证器覆盖的字段**：

- `DatabaseValidator`: database.host, database.port, database.username, database.password, database.type 等
- `WebServerValidator`: server.host, server.port, server.timeout 等
- `RedisValidator`: redis.host, redis.port, redis.db, redis.addresses 等
- `LogValidator`: logging.level, logging.format, logging.path 等
- `EmailValidator`: email.smtp.host, email.smtp.port, email.from 等
- `APIValidator`: api.base_url, api.timeout, api.auth.api_key 等

### 步骤 2: 创建自定义验证器

使用 `RuleValidator` 创建业务特定的验证器：

```go
// 创建自定义业务验证器
businessValidator := validation.NewRuleValidator("业务配置验证器")

// 添加验证规则
businessValidator.AddRule("company.name", validation.Required("公司名称不能为空"))
businessValidator.AddRule("company.tax_id", validation.Pattern(`^\d{18}$`, "税号必须是18位数字"))
businessValidator.AddStringRule("company.industry", "enum:technology,finance,healthcare,education")
businessValidator.AddStringRule("company.employee_count", "range:1,10000")

// 添加到配置
cfg.AddValidator(businessValidator)
```

### 步骤 3: 使用 30+ 内置验证规则

**网络相关验证**：

```go
validator.AddStringRule("network.email", "email")           // 邮箱格式
validator.AddStringRule("network.url", "url")               // URL 格式
validator.AddStringRule("network.ipv4", "ipv4")             // IPv4 地址
validator.AddStringRule("network.ipv6", "ipv6")             // IPv6 地址
validator.AddStringRule("network.hostname", "hostname")      // 主机名
validator.AddStringRule("network.port", "port")             // 端口号 (1-65535)
```

**数据格式验证**：

```go
validator.AddStringRule("format.uuid", "uuid")              // UUID 格式
validator.AddStringRule("format.json", "json")              // JSON 格式
validator.AddStringRule("format.base64", "base64")          // Base64 编码
validator.AddStringRule("format.alphanum", "alphanum")      // 字母数字
validator.AddStringRule("format.phone", "phonenumber")      // 电话号码
```

**数值范围验证**：

```go
validator.AddStringRule("numbers.age", "range:1,120")       // 范围验证
validator.AddStringRule("numbers.length", "length:5,20")    // 长度验证
validator.AddStringRule("numbers.min_value", "min:1")       // 最小值
validator.AddStringRule("numbers.max_value", "max:100")     // 最大值
```

**业务规则验证**：

```go
validator.AddStringRule("payment.card", "creditcard")       // 信用卡号
validator.AddStringRule("date.timezone", "timezone")        // 时区
validator.AddStringRule("date.datetime", "datetime")        // 日期时间
validator.AddRule("config.required_field", validation.Required("必填字段"))
```

完整规则列表请参考 `REFERENCE.md`。

### 步骤 4: 使用字段级验证

字段级验证只验证相关字段，避免全量验证的性能开销：

```go
// 设置配置时自动触发字段级验证
err := cfg.Set("server.port", 8080)     // ✅ 有效端口
if err != nil {
    log.Printf("验证失败: %v", err)
}

err = cfg.Set("server.port", 70000)     // ❌ 无效端口，被验证器拦截
if err != nil {
    log.Printf("验证失败: %v", err)      // 只验证 server.port 相关规则
}
```

**字段级验证的优势**：

- 只验证相关验证器和字段
- 跳过 required 检查，避免级联失败
- 更快的验证速度
- 更精确的错误信息

### 步骤 5: 创建复合验证器

组合多个验证器为一个验证器：

```go
composite := validation.NewCompositeValidator(
    "企业级应用验证器",
    validation.NewDatabaseValidator(),
    validation.NewWebServerValidator(),
    validation.NewRedisValidator(),
    customBusinessValidator,
)

cfg.AddValidator(composite)
```

### 步骤 6: 动态验证器管理

运行时添加或移除验证器：

```go
// 添加验证器
cfg.AddValidator(validation.NewEmailValidator())

// 添加函数式验证器
cfg.AddValidateFunc(func(config map[string]any) error {
    // 自定义验证逻辑
    if env := config["app"].(map[string]any)["env"]; env == "production" {
        // 生产环境特殊验证
        return validateProductionConfig(config)
    }
    return nil
})

// 获取当前验证器列表
validators := cfg.GetValidators()
log.Printf("当前验证器数量: %d", len(validators))

// 清除所有验证器
cfg.ClearValidators()
```

## 模板和示例

### 模板文件

- `templates/validator.go.tmpl` - 自定义验证器模板

### 示例代码

- `examples/predefined-validators.go` - 使用预定义验证器
- `examples/custom-validators.go` - 创建自定义验证器

## 验证规则参考

### 常用验证规则快速查询

| 规则             | 说明             | 示例                             |
| ---------------- | ---------------- | -------------------------------- |
| `required`       | 必填字段         | `Required("不能为空")`           |
| `email`          | 邮箱格式         | `"email"`                        |
| `url`            | URL 格式         | `"url"`                          |
| `port`           | 端口号 (1-65535) | `"port"`                         |
| `ipv4`           | IPv4 地址        | `"ipv4"`                         |
| `hostname`       | 主机名           | `"hostname"`                     |
| `range:min,max`  | 数值范围         | `"range:1,100"`                  |
| `length:min,max` | 长度范围         | `"length:5,20"`                  |
| `min:value`      | 最小值           | `"min:1"`                        |
| `max:value`      | 最大值           | `"max:100"`                      |
| `enum:v1,v2,v3`  | 枚举值           | `"enum:dev,test,prod"`           |
| `pattern:regex`  | 正则匹配         | `Pattern("^\d+$", "只能是数字")` |
| `uuid`           | UUID 格式        | `"uuid"`                         |
| `json`           | JSON 格式        | `"json"`                         |
| `base64`         | Base64 编码      | `"base64"`                       |
| `alphanum`       | 字母数字         | `"alphanum"`                     |

完整规则列表和详细说明请参考 `REFERENCE.md`。

## 最佳实践

### 1. 分层验证策略

```go
// 基础验证：使用预定义验证器
cfg, _ := sysconf.New(
    sysconf.WithValidators(
        validation.NewDatabaseValidator(),
        validation.NewWebServerValidator(),
    ),
)

// 业务验证：添加自定义验证器
cfg.AddValidator(customBusinessValidator)

// 环境验证：添加特定环境的验证
if env == "production" {
    cfg.AddValidator(productionValidator)
}
```

### 2. 验证器命名规范

```go
// ✅ 推荐：描述性名称
validator := validation.NewRuleValidator("用户认证配置验证器")
validator := validation.NewRuleValidator("支付系统配置验证")

// ❌ 避免：模糊名称
validator := validation.NewRuleValidator("validator1")
```

### 3. 错误处理

```go
// 设置配置时捕获验证错误
if err := cfg.Set("server.port", invalidPort); err != nil {
    // 记录详细的验证错误
    log.Printf("配置验证失败 - 字段: server.port, 错误: %v", err)
    // 提供用户友好的错误提示
    return fmt.Errorf("服务器端口配置无效，请检查端口范围 (1-65535)")
}
```

### 4. 性能优化

- 使用字段级验证，避免全量验证
- 预定义验证器会自动推断支持字段，无需手动配置
- 复合验证器会自动去重，避免重复验证

### 5. 验证规则组织

```go
// ✅ 推荐：按模块组织验证规则
serverValidator := validation.NewRuleValidator("服务器验证器")
serverValidator.AddStringRule("server.host", "hostname")
serverValidator.AddStringRule("server.port", "port")
serverValidator.AddStringRule("server.timeout", "required")

dbValidator := validation.NewRuleValidator("数据库验证器")
dbValidator.AddStringRule("database.host", "hostname")
dbValidator.AddStringRule("database.port", "port")
dbValidator.AddStringRule("database.username", "required")
```

## 故障排查

**问题: 验证器没有生效**

- 确认在 `New()` 时使用了 `WithValidators()` 或使用 `AddValidator()` 添加
- 检查验证器的字段前缀是否与配置键匹配
- 验证规则字符串格式是否正确

**问题: 验证规则报错**

- 检查规则语法是否正确 (例如: `range:1,100` 不是 `range:1-100`)
- 确认枚举值使用逗号分隔 (例如: `enum:dev,test,prod`)
- 正则表达式需要使用 `Pattern()` 函数，不能直接用字符串

**问题: 字段级验证未生效**

- 确认使用的是预定义验证器或 `StructuredValidator`
- 检查字段前缀是否在验证器的支持范围内
- 使用 `GetSupportedFields()` 查看验证器支持的字段

## 性能提示

- 预定义验证器会缓存支持字段列表，首次推断后性能极高
- 字段级验证比全量验证快 10-100 倍
- 复合验证器会自动优化，避免重复验证
- 验证器推断算法复杂度为 O(1)，不影响配置读写性能

## 相关资源

- 完整验证规则参考: `REFERENCE.md`
- 预定义验证器源码: /workspace/validation/predefined.go
- 验证规则实现: /workspace/validation/rules.go
- 主集成技能: `integrating-sysconf`
