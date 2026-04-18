---
name: sysconf-integrating
description: 帮助开发者在 Go 项目中集成和使用 sysconf 配置管理库。当用户需要添加配置管理、读取配置文件、环境变量集成或初始化配置系统时使用此技能。
version: "1.0.0"
---

# Sysconf 配置管理库集成

此技能帮助你在 Go 项目中快速集成和使用 sysconf 高性能配置管理库。

## 概述

Sysconf 是一个企业级 Go 配置管理库，提供以下核心功能：

- 多格式支持（YAML、JSON、TOML、Dotenv）
- 高性能并发安全访问
- 智能验证系统
- 环境变量智能匹配
- 配置热重载
- ChaCha20-Poly1305 加密

## 何时使用此技能

使用此技能当你需要：

- 在新项目中集成配置管理
- 从其他配置库迁移到 sysconf
- 设置基础配置结构
- 配置环境变量支持
- 实现配置文件读取和解析
- 集成 Cobra/PFlag 命令行工具

## 前置要求

- Go 1.24+ 已安装
- 项目已初始化 `go.mod`
- 对 Go 结构体和标签有基本了解

## 使用指南

### 步骤 1: 安装依赖

首先，在你的项目中添加 sysconf 依赖：

```bash
go get github.com/darkit/sysconf
```

如果需要使用验证功能，确保导入验证包：

```go
import "github.com/darkit/sysconf/validation"
```

### 步骤 2: 创建配置文件

参考 `templates/basic-config.yaml` 创建你的配置文件。配置文件应放在项目的 `configs/` 或 `config/` 目录中。

基础配置文件示例：

```yaml
app:
  name: "MyApp"
  version: "1.0.0"
  env: "development"

server:
  host: "localhost"
  port: 8080
  timeout: "30s"

database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "secret"
```

### 步骤 3: 定义配置结构体

在你的项目中创建配置结构体，使用 `config` 标签映射配置字段，`default` 标签设置默认值，`validate` 标签添加验证规则：

参考 `templates/struct-config.go.tmpl` 了解完整的结构体定义模式。

关键要点：

- 使用 `config` 标签指定配置键名
- 使用 `default` 标签提供默认值
- 使用 `validate` 标签添加验证规则
- 敏感信息（如密码）不设置默认值

### 步骤 4: 初始化配置实例

使用函数式选项模式创建配置实例：

```go
cfg, err := sysconf.New(
    sysconf.WithPath("configs"),           // 配置文件目录
    sysconf.WithName("app"),               // 配置文件名（不含扩展名）
    sysconf.WithMode("yaml"),              // 配置格式
    sysconf.WithContent(defaultConfig),    // 默认配置内容
    sysconf.WithEnv("APP"),                // 环境变量前缀
)
if err != nil {
    log.Fatal("配置初始化失败:", err)
}
```

### 步骤 5: 读取配置

有三种方式读取配置：

**方式 1: 结构体映射（推荐）**

```go
var config AppConfig
if err := cfg.Unmarshal(&config); err != nil {
    log.Fatal("配置解析失败:", err)
}
// 使用: config.Server.Port
```

**方式 2: 键值访问**

```go
host := cfg.GetString("server.host", "localhost")
port := cfg.GetInt("server.port", 8080)
timeout := cfg.GetDuration("server.timeout")
```

**方式 3: 泛型 API（类型安全）**

```go
host := sysconf.GetAs[string](cfg, "server.host", "localhost")
port := sysconf.GetAs[int](cfg, "server.port", 8080)
```

### 步骤 6: 动态更新配置

配置支持运行时更新：

```go
if err := cfg.Set("server.port", 9090); err != nil {
    log.Printf("配置更新失败: %v", err)
}
```

**注意**: 配置更新会自动触发验证，无效值会被拒绝。

## 模板文件

此技能提供以下模板，可直接使用或按需修改：

- `templates/basic-config.yaml` - 基础配置文件模板
- `templates/struct-config.go.tmpl` - 完整的配置结构体模板
- `templates/main.go.tmpl` - 主程序集成模板

## 示例代码

查看 `examples/` 目录获取完整示例：

- `quick-start.go` - 5 分钟快速开始示例
- `full-integration.go` - 完整的生产级集成示例
- `env-vars.go` - 环境变量配置示例

## 测试模板

参考 `tests/config_test.go.tmpl` 了解如何编写配置相关的单元测试。

## 常见配置选项

| 选项                   | 说明         | 示例                       |
| ---------------------- | ------------ | -------------------------- |
| `WithPath(path)`       | 配置文件路径 | `WithPath("configs")`      |
| `WithName(name)`       | 配置文件名   | `WithName("app")`          |
| `WithMode(mode)`       | 配置格式     | `WithMode("yaml")`         |
| `WithContent(content)` | 默认配置内容 | `WithContent(defaultYAML)` |
| `WithEnv(prefix)`      | 环境变量前缀 | `WithEnv("APP")`           |
| `WithValidators(...)`  | 添加验证器   | 见验证技能                 |
| `WithEncryption(key)`  | 启用加密     | 见加密技能                 |

## 环境变量覆盖

Sysconf 支持智能大小写匹配的环境变量：

```bash
# 以下格式都会被识别（假设前缀为 APP）
export APP_SERVER_HOST=localhost      # 标准大写格式
export app_server_host=localhost      # 小写格式
export App_Server_Host=localhost      # 混合大小写
export SERVER_PORT=8080                # 无前缀格式
```

配置优先级：**命令行参数 > 环境变量 > 配置文件 > 默认值**

## 高级功能

对于更高级的功能，请使用相应的专用技能：

- **配置验证**: 使用 `configuring-validation` 技能
- **配置加密**: 使用 `encrypting-config` 技能
- **热重载监控**: 使用 `monitoring-config-changes` 技能

## 最佳实践

1. **按模块组织配置**: 使用嵌套结构（app、server、database 等）
2. **合理使用默认值**: 非敏感配置提供默认值，敏感信息用环境变量
3. **添加验证规则**: 使用 `validate` 标签确保配置有效性
4. **环境区分**: 为不同环境（dev、test、prod）使用不同配置文件
5. **配置热重载**: 在需要时使用 `Watch` 功能实现配置动态更新

## 故障排查

**问题: 配置文件找不到**

- 检查 `WithPath` 和 `WithName` 设置是否正确
- 确认配置文件存在且有读取权限
- 使用 `WithContent` 提供默认配置作为后备

**问题: 环境变量未生效**

- 确认使用了 `WithEnv` 选项
- 检查环境变量命名格式（使用下划线分隔）
- 验证环境变量前缀是否匹配

**问题: 配置解析失败**

- 检查 YAML/JSON 格式是否正确
- 确认结构体标签是否与配置键匹配
- 检查类型转换是否兼容

## 性能提示

- Sysconf 使用原子存储，读取操作无锁且极快（微秒级）
- 配置实例可在多个 goroutine 中安全使用
- 考虑使用结构体映射方式提升访问性能
- 批量更新时使用延迟写入（默认 3 秒，可通过 `WithWriteDebounceDelay` 调整）

## 相关资源

- 项目仓库: https://github.com/darkit/sysconf
- API 文档: https://pkg.go.dev/github.com/darkit/sysconf
- README: /workspace/README.md
