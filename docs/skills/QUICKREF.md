# Sysconf Skills 快速参考指南

本文档提供 sysconf Claude Code Skills 的快速参考。

## 快速导航

- [5分钟快速开始](#5分钟快速开始)
- [常见场景](#常见场景)
- [API 速查表](#api-速查表)
- [故障排查](#故障排查)

## 5分钟快速开始

### 1. 安装依赖

```bash
go get github.com/darkit/sysconf
```

### 2. 创建配置文件 `configs/app.yaml`

```yaml
app:
  name: "MyApp"
  port: 8080
database:
  host: "localhost"
  port: 5432
```

### 3. 编写代码 `main.go`

```go
package main

import (
    "log"
    "github.com/darkit/sysconf"
)

func main() {
    cfg, _ := sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName("app"),
    )

    port := cfg.GetInt("app.port")
    log.Printf("应用端口: %d", port)
}
```

### 4. 运行

```bash
go run main.go
```

## 常见场景

### 场景 1: 基础配置集成

**需求**: 在项目中添加配置管理

**使用 SKILL**: `integrating-sysconf`

**步骤**:
1. 复制 `templates/basic-config.yaml` 到项目
2. 复制 `templates/struct-config.go.tmpl` 并调整字段
3. 参考 `examples/quick-start.go` 创建主程序

**时间**: 10-15 分钟

---

### 场景 2: 添加配置验证

**需求**: 确保配置值有效，防止错误配置

**使用 SKILL**: `configuring-validation`

**步骤**:
1. 选择预定义验证器或创建自定义验证器
2. 在 `New()` 时使用 `WithValidators()` 添加
3. 参考 `examples/predefined-validators.go`

**时间**: 5-10 分钟

---

### 场景 3: 保护敏感配置

**需求**: 加密数据库密码、API 密钥等敏感信息

**使用 SKILL**: `encrypting-config`

**步骤**:
1. 准备加密密钥（从环境变量或密钥管理服务）
2. 使用 `WithEncryption(key)` 启用加密
3. 参考 `examples/basic-encryption.go`

**时间**: 10-15 分钟

---

### 场景 4: 配置热重载

**需求**: 无需重启应用即可更新配置

**使用 SKILL**: `monitoring-config-changes`

**步骤**:
1. 使用 `atomic.Value` 存储配置
2. 调用 `cfg.Watch()` 或 `cfg.WatchWithContext()`
3. 在回调中重新加载配置
4. 参考 `examples/watch-config.go`

**时间**: 15-20 分钟

## API 速查表

### 配置创建

```go
// 基础配置
cfg, _ := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("app"),
    sysconf.WithMode("yaml"),
)

// 使用默认内容
cfg, _ := sysconf.New(
    sysconf.WithContent(defaultYAML),
)

// 环境变量支持
cfg, _ := sysconf.New(
    sysconf.WithEnv("APP"),
)

// 验证器
cfg, _ := sysconf.New(
    sysconf.WithValidators(
        validation.NewDatabaseValidator(),
    ),
)

// 加密
cfg, _ := sysconf.New(
    sysconf.WithEncryption(key),
)
```

### 配置读取

```go
// 基础类型
str := cfg.GetString("key", "default")
num := cfg.GetInt("key", 0)
flag := cfg.GetBool("key", false)
duration := cfg.GetDuration("key")

// 切片
strs := cfg.GetStringSlice("key")
nums := cfg.GetIntSlice("key")

// 映射
m := cfg.GetStringMap("key")
sm := cfg.GetStringMapString("key")

// 泛型（类型安全）
value := sysconf.GetAs[string](cfg, "key", "default")

// 结构体
var config AppConfig
cfg.Unmarshal(&config)
```

### 配置写入

```go
// 设置值
cfg.Set("key", value)

// 批量设置
for k, v := range updates {
    cfg.Set(k, v)
}
```

### 配置监听

```go
// 基础监听
cfg.Watch(func() {
    // 配置变化时执行
})

// Context 控制
ctx, cancel := context.WithCancel(context.Background())
stop := cfg.WatchWithContext(ctx, func() {
    // 配置变化时执行
})
// 停止监听
stop()
```

## 故障排查

### 问题: 配置文件找不到

**症状**: `open configs/app.yaml: no such file or directory`

**解决方案**:
1. 检查 `WithPath()` 和 `WithName()` 设置
2. 确认文件存在于指定路径
3. 使用 `WithContent()` 提供默认配置

---

### 问题: 环境变量未生效

**症状**: 环境变量设置了但配置未更新

**解决方案**:
1. 确认使用了 `WithEnv("PREFIX")`
2. 检查环境变量命名: `PREFIX_SECTION_KEY`
3. 环境变量使用下划线，配置键使用点号

---

### 问题: 验证器不工作

**症状**: 无效配置未被拦截

**解决方案**:
1. 确认在 `New()` 时添加了验证器
2. 检查字段前缀是否匹配
3. 使用 `cfg.GetValidators()` 查看已注册验证器

---

### 问题: 配置热重载未触发

**症状**: 修改配置文件后回调未执行

**解决方案**:
1. 确认调用了 `Watch()` 或 `WatchWithContext()`
2. 检查文件系统是否支持 fsnotify
3. Docker 卷挂载可能需要特殊配置

---

### 问题: 解密失败

**症状**: `decryption failed` 或 `invalid key`

**解决方案**:
1. 确认使用的密钥与加密时相同
2. 检查密钥长度（至少 32 字节）
3. 确认配置文件未被损坏

## 性能提示

| 操作 | 性能 | 说明 |
|------|------|------|
| 配置读取 | ~1μs | 使用原子存储，无锁读取 |
| 配置写入 | ~1ms | 包含验证和持久化 |
| 热重载触发 | ~100ms | 防抖后的配置更新 |
| 验证执行 | ~100μs | 字段级验证 |
| 加密/解密 | ~50μs | ChaCha20-Poly1305 |

## 最佳实践速查

✅ **推荐做法**:
- 使用结构体映射访问配置
- 为所有环境变量设置前缀
- 敏感信息使用环境变量
- 启用配置验证
- 使用 `atomic.Value` 存储配置
- 定期轮换加密密钥

❌ **避免做法**:
- 硬编码配置值
- 密钥硬编码在代码中
- 忽略验证错误
- 在回调中执行耗时操作
- 混用多个配置实例

## 代码模板索引

### 主集成
- 基础配置文件: `integrating-sysconf/templates/basic-config.yaml`
- 配置结构体: `integrating-sysconf/templates/struct-config.go.tmpl`
- 主程序: `integrating-sysconf/templates/main.go.tmpl`
- 测试: `integrating-sysconf/tests/config_test.go.tmpl`

### 验证
- 验证器: `configuring-validation/templates/validator.go.tmpl`

### 加密
- 加密配置: `encrypting-config/templates/encrypted-config.go.tmpl`

### 热重载
- 热重载: `monitoring-config-changes/templates/hot-reload.go.tmpl`

## 完整示例索引

### 快速开始
- 5分钟示例: `integrating-sysconf/examples/quick-start.go`
- 完整集成: `integrating-sysconf/examples/full-integration.go`
- 环境变量: `integrating-sysconf/examples/env-vars.go`

### 验证
- 预定义验证器: `configuring-validation/examples/predefined-validators.go`
- 自定义验证器: `configuring-validation/examples/custom-validators.go`

### 加密
- 基础加密: `encrypting-config/examples/basic-encryption.go`

### 热重载
- 配置监听: `monitoring-config-changes/examples/watch-config.go`

## 相关链接

- 主 README: `README.md`
- 验证规则参考: `configuring-validation/REFERENCE.md`
- 项目 README: `/workspace/README.md`
- API 文档: https://pkg.go.dev/github.com/darkit/sysconf

---

**提示**: 这是快速参考指南。详细文档请查看各个 SKILL 的 `SKILL.md` 文件。
