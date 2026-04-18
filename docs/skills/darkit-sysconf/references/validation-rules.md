# Sysconf 验证规则完整参考

本文档提供 sysconf 验证系统的完整规则参考。

## 网络相关验证规则

### email - 邮箱格式验证

验证字符串是否为有效的邮箱地址。

**示例**:

```go
validator.AddStringRule("user.email", "email")
```

**有效值**: `user@example.com`, `admin@company.co.uk`
**无效值**: `invalid-email`, `user@`, `@example.com`

---

### url - URL 格式验证

验证字符串是否为有效的 URL。

**示例**:

```go
validator.AddStringRule("api.endpoint", "url")
```

**有效值**: `https://api.example.com`, `http://localhost:8080/api`
**无效值**: `not-a-url`, `htp://invalid`, `//example.com`

---

### ipv4 - IPv4 地址验证

验证字符串是否为有效的 IPv4 地址。

**示例**:

```go
validator.AddStringRule("server.ip", "ipv4")
```

**有效值**: `192.168.1.1`, `10.0.0.1`, `127.0.0.1`
**无效值**: `999.999.999.999`, `192.168.1`, `2001:db8::1`

---

### ipv6 - IPv6 地址验证

验证字符串是否为有效的 IPv6 地址。

**示例**:

```go
validator.AddStringRule("server.ipv6", "ipv6")
```

**有效值**: `2001:0db8:85a3:0000:0000:8a2e:0370:7334`, `::1`
**无效值**: `192.168.1.1`, `invalid:ipv6`

---

### hostname - 主机名验证

验证字符串是否为有效的主机名。

**示例**:

```go
validator.AddStringRule("database.host", "hostname")
```

**有效值**: `localhost`, `api.example.com`, `db-server-01`
**无效值**: `invalid_hostname!`, `-invalid`, `host name with spaces`

---

### port - 端口号验证

验证数值是否为有效的端口号 (1-65535)。

**示例**:

```go
validator.AddStringRule("server.port", "port")
```

**有效值**: `80`, `8080`, `443`, `65535`
**无效值**: `0`, `-1`, `70000`, `abc`

---

## 数据格式验证规则

### uuid - UUID 格式验证

验证字符串是否为有效的 UUID (v1-v5)。

**示例**:

```go
validator.AddStringRule("request.id", "uuid")
```

**有效值**: `123e4567-e89b-12d3-a456-426614174000`
**无效值**: `not-a-uuid`, `123e4567e89b12d3a456426614174000`

---

### json - JSON 格式验证

验证字符串是否为有效的 JSON。

**示例**:

```go
validator.AddStringRule("config.data", "json")
```

**有效值**: `{"key": "value"}`, `[1, 2, 3]`, `"string"`
**无效值**: `{invalid json}`, `{key: value}`

---

### base64 - Base64 编码验证

验证字符串是否为有效的 Base64 编码。

**示例**:

```go
validator.AddStringRule("auth.token", "base64")
```

**有效值**: `SGVsbG8gV29ybGQ=`, `YWJjMTIz`
**无效值**: `not base64!`, `Invalid===`

---

### alphanum - 字母数字验证

验证字符串是否只包含字母和数字。

**示例**:

```go
validator.AddStringRule("user.username", "alphanum")
```

**有效值**: `abc123`, `User01`, `Test`
**无效值**: `user-name`, `email@example.com`, `has spaces`

---

### phonenumber - 电话号码验证

验证字符串是否为有效的电话号码格式。

**示例**:

```go
validator.AddStringRule("contact.phone", "phonenumber")
```

**有效值**: `+1234567890`, `1234567890`, `+86-123-4567-8900`
**无效值**: `abc`, `12-ab-34`

---

## 数值范围验证规则

### range:min,max - 范围验证

验证数值是否在指定范围内（包含边界）。

**示例**:

```go
validator.AddStringRule("user.age", "range:1,120")
validator.AddStringRule("score", "range:0,100")
```

**有效值**: `50` (范围 1-120), `100` (范围 0-100)
**无效值**: `0` (范围 1-120), `150` (范围 1-120)

---

### length:min,max - 长度验证

验证字符串或切片的长度是否在指定范围内。

**示例**:

```go
validator.AddStringRule("user.password", "length:8,32")
validator.AddStringRule("tags", "length:1,10")
```

**有效值**: `"password123"` (长度 8-32), `["tag1", "tag2"]` (长度 1-10)
**无效值**: `"short"` (长度 8-32), `[]` (长度 1-10)

---

### min:value - 最小值验证

验证数值是否大于等于最小值。

**示例**:

```go
validator.AddStringRule("database.max_conns", "min:1")
validator.AddStringRule("timeout", "min:0")
```

**有效值**: `10`, `1`, `100`
**无效值**: `0` (min:1), `-1` (min:0)

---

### max:value - 最大值验证

验证数值是否小于等于最大值。

**示例**:

```go
validator.AddStringRule("database.max_conns", "max:100")
validator.AddStringRule("retry_count", "max:5")
```

**有效值**: `50`, `100`, `1`
**无效值**: `101` (max:100), `10` (max:5)

---

## 业务规则验证

### enum:value1,value2,value3 - 枚举值验证

验证值是否在枚举列表中。

**示例**:

```go
validator.AddStringRule("app.env", "enum:development,test,production")
validator.AddStringRule("log.level", "enum:debug,info,warn,error")
```

**有效值**: `"development"`, `"production"`
**无效值**: `"staging"`, `"dev"`

---

### required - 必填验证

验证字段是否存在且非空。

**示例**:

```go
validator.AddRule("user.name", validation.Required("用户名不能为空"))
validator.AddRule("database.password", validation.Required("密码不能为空"))
```

**有效值**: 任何非空值
**无效值**: `nil`, `""`, `0` (对于字符串), 空切片

**注意**: 字段级验证会跳过 required 检查，避免级联失败。

---

### pattern:regex - 正则表达式验证

使用正则表达式验证字符串格式。

**示例**:

```go
validator.AddRule("company.tax_id", validation.Pattern(`^\d{18}$`, "税号必须是18位数字"))
validator.AddRule("product.code", validation.Pattern(`^[A-Z]{2}\d{6}$`, "产品代码格式: 两个大写字母+6位数字"))
```

**有效值**: `"123456789012345678"` (税号), `"AB123456"` (产品代码)
**无效值**: `"12345"` (税号), `"ab123456"` (产品代码)

---

### creditcard - 信用卡号验证

验证字符串是否为有效的信用卡号（使用 Luhn 算法）。

**示例**:

```go
validator.AddStringRule("payment.card_number", "creditcard")
```

**有效值**: 有效的信用卡号
**无效值**: 无效的信用卡号格式

---

### datetime - 日期时间验证

验证字符串是否为有效的日期时间格式。

**示例**:

```go
validator.AddStringRule("event.start_time", "datetime")
```

**有效值**: `"2024-01-01T00:00:00Z"`, `"2024-01-01"`
**无效值**: `"invalid-date"`, `"13/32/2024"`

---

### timezone - 时区验证

验证字符串是否为有效的时区标识符。

**示例**:

```go
validator.AddStringRule("app.timezone", "timezone")
```

**有效值**: `"UTC"`, `"Asia/Shanghai"`, `"America/New_York"`
**无效值**: `"Invalid/Timezone"`, `"GMT+8"`

---

## 预定义验证器支持的字段

### DatabaseValidator

验证数据库配置相关字段。

**支持字段**:

- `database.host` - 数据库主机名（hostname 验证）
- `database.port` - 数据库端口（1-65535）
- `database.username` - 数据库用户名（必填，非空）
- `database.password` - 数据库密码（必填，非空）
- `database.database` - 数据库名称（必填，非空）
- `database.type` - 数据库类型（枚举: postgresql, mysql, sqlite, mongodb）
- `database.max_conns` - 最大连接数（1-100）
- `database.timeout` - 连接超时（必填）

---

### WebServerValidator

验证 Web 服务器配置相关字段。

**支持字段**:

- `server.host` - 服务器主机名（hostname 验证）
- `server.port` - 服务器端口（1-65535）
- `server.timeout` - 请求超时（必填）
- `server.read_timeout` - 读取超时
- `server.write_timeout` - 写入超时

---

### RedisValidator

验证 Redis 配置相关字段。

**支持字段**:

- `redis.host` - Redis 主机名（hostname 验证）
- `redis.port` - Redis 端口（1-65535）
- `redis.password` - Redis 密码（可选）
- `redis.db` - 数据库索引（0-15）
- `redis.timeout` - 连接超时（必填）
- `redis.addresses` - 集群地址列表（必填，每个地址必填）

---

### LogValidator

验证日志配置相关字段。

**支持字段**:

- `logging.level` - 日志级别（枚举: debug, info, warn, error）
- `logging.format` - 日志格式（枚举: json, text）
- `logging.path` - 日志文件路径（必填）
- `logging.max_size` - 最大文件大小（1-1000 MB）
- `logging.max_backups` - 最大备份数（0-100）
- `logging.max_age` - 最大保留天数（1-365）

---

### EmailValidator

验证邮件配置相关字段。

**支持字段**:

- `email.smtp.host` - SMTP 服务器主机名（必填，hostname 验证）
- `email.smtp.port` - SMTP 端口（1-65535）
- `email.smtp.username` - SMTP 用户名（必填，email 格式）
- `email.smtp.password` - SMTP 密码（必填，非空）
- `email.from` - 发件人地址（必填，email 格式）

---

### APIValidator

验证 API 配置相关字段。

**支持字段**:

- `api.base_url` - API 基础 URL（必填，URL 格式）
- `api.timeout` - API 超时（1-300 秒）
- `api.rate_limit.enabled` - 是否启用限流
- `api.rate_limit.requests_per_minute` - 每分钟请求数（1-10000）
- `api.auth.api_key` - API 密钥（必填，最小长度 10）

---

## 验证器组合使用

### 复合验证器示例

```go
// 创建复合验证器
composite := validation.NewCompositeValidator(
    "企业级应用验证器",
    validation.NewDatabaseValidator(),
    validation.NewWebServerValidator(),
    validation.NewRedisValidator(),
    validation.NewLogValidator(),
    validation.NewEmailValidator(),
    validation.NewAPIValidator(),
)

cfg.AddValidator(composite)
```

### 自定义 + 预定义验证器

```go
// 创建自定义业务验证器
businessValidator := validation.NewRuleValidator("业务验证器")
businessValidator.AddRule("company.name", validation.Required("公司名称必填"))
businessValidator.AddStringRule("company.industry", "enum:tech,finance,retail")

// 组合使用
cfg, _ := sysconf.New(
    sysconf.WithValidators(
        validation.NewDatabaseValidator(),  // 预定义验证器
        validation.NewWebServerValidator(), // 预定义验证器
        businessValidator,                  // 自定义验证器
    ),
)
```

---

## 验证规则字符串语法

### 单规则

```go
"email"           // 邮箱验证
"url"             // URL 验证
"required"        // 必填（通过 Required() 函数）
```

### 带参数规则

```go
"range:1,100"     // 范围: 1 到 100
"length:5,20"     // 长度: 5 到 20
"min:1"           // 最小值: 1
"max:100"         // 最大值: 100
"enum:dev,test,prod"  // 枚举值
```

### 多规则组合（使用逗号分隔）

```go
"required,email"           // 必填且为邮箱格式
"required,min:1,max:100"   // 必填且在 1-100 范围
```

**注意**: `required` 规则需要使用 `validation.Required()` 函数创建，不能用字符串形式。

---

## 错误信息自定义

```go
// 使用 Required 函数自定义错误信息
validator.AddRule("user.name", validation.Required("用户名不能为空"))

// 使用 Pattern 函数自定义错误信息
validator.AddRule("phone", validation.Pattern(`^\d{11}$`, "手机号必须是11位数字"))

// 内置规则使用默认错误信息
validator.AddStringRule("email", "email")  // 默认: "无效的邮箱格式"
```

---

## 性能注意事项

1. **字段级验证**: 只验证相关字段，性能优于全量验证
2. **预定义验证器**: 会缓存支持字段列表，首次推断后性能极高
3. **正则表达式**: 复杂正则会影响性能，建议简化或缓存编译结果
4. **验证器数量**: 多个验证器会依次执行，建议合理组织

---

## 完整示例

```go
package main

import (
    "github.com/darkit/sysconf"
    "github.com/darkit/sysconf/validation"
)

func main() {
    // 创建自定义验证器
    businessValidator := validation.NewRuleValidator("业务验证器")

    // 网络验证
    businessValidator.AddStringRule("api.url", "url")
    businessValidator.AddStringRule("admin.email", "email")

    // 数值验证
    businessValidator.AddStringRule("user.age", "range:1,150")
    businessValidator.AddStringRule("password", "length:8,32")

    // 枚举验证
    businessValidator.AddStringRule("role", "enum:admin,user,guest")

    // 必填验证
    businessValidator.AddRule("company.name", validation.Required("公司名称必填"))

    // 正则验证
    businessValidator.AddRule("tax_id",
        validation.Pattern(`^\d{15}|\d{18}$`, "税号必须是15或18位数字"))

    // 创建配置实例，使用预定义+自定义验证器
    cfg, _ := sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName("app"),
        sysconf.WithValidators(
            validation.NewDatabaseValidator(),
            validation.NewWebServerValidator(),
            businessValidator,
        ),
    )

    // 测试验证
    cfg.Set("api.url", "https://api.example.com")       // ✅ 通过
    cfg.Set("user.age", 25)                             // ✅ 通过
    cfg.Set("role", "admin")                            // ✅ 通过

    cfg.Set("api.url", "invalid-url")                   // ❌ 失败
    cfg.Set("user.age", 200)                            // ❌ 失败
    cfg.Set("role", "superadmin")                       // ❌ 失败
}
```

---

## 相关资源

- 验证器源码: /workspace/validation/
- 规则实现: /workspace/validation/rules.go
- 预定义验证器: /workspace/validation/predefined.go
- 主验证技能: `configuring-validation`
