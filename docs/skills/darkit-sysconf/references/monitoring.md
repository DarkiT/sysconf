---
name: sysconf-monitoring-config-changes
description: 实现 sysconf 配置热重载和变更监控功能。当用户需要监听配置文件变化、实现配置热更新、无重启更新配置或处理配置变更事件时使用。
version: "1.0.0"
---

# Sysconf 配置热重载与监控

此技能帮助你在 sysconf 项目中实现配置热重载和变更监控功能。

## 概述

Sysconf 提供生产级配置热重载功能：

- **防抖机制**: 默认 200ms 内多次变更只触发一次（可通过 WithWatchDebounce 调整）
- **并发安全**: 配置更新期间服务不中断
- **错误恢复**: 配置验证失败时自动回滚
- **智能监控**: 只监控实际的文件写入操作
- **可控监听**: 支持取消和暂停监听
- **原子更新**: 使用 `atomic.Value` 保证更新原子性

## 何时使用此技能

使用此技能当你需要：

- 实现配置动态更新，无需重启应用
- 监听配置文件变化并触发相应操作
- 在生产环境中调整配置参数
- 实现配置回滚和错误处理
- 集成配置中心（如 Consul、etcd）
- 监控配置变更事件

## 前置要求

- 已完成 sysconf 基础集成（参考 `integrating-sysconf` 技能）
- 了解 Go 的 context 和 goroutine
- 了解原子操作和并发安全

## 使用指南

### 步骤 1: 基础热重载（最简单方式）

```go
import (
    "log"
    "github.com/darkit/sysconf"
)

func main() {
    // 创建配置实例
    cfg, err := sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName("app"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 启动配置监听
    cfg.Watch(func() {
        log.Println("配置文件已更新！")

        // 重新读取配置
        newPort := cfg.GetInt("server.port")
        log.Printf("新的服务器端口: %d", newPort)

        // 在此处理配置变更逻辑
        updateServerConfig(newPort)
    })

    // 应用继续运行
    runApplication(cfg)
}
```

### 步骤 2: 使用 Context 控制监听（推荐）

```go
func setupConfigWatch(cfg *sysconf.Config) context.CancelFunc {
    ctx, cancel := context.WithCancel(context.Background())

    // 启动可取消的配置监听
    stopWatch := cfg.WatchWithContext(ctx, func() {
        log.Println("检测到配置变化，正在重新加载...")

        // 重新解析配置到结构体
        var newConfig AppConfig
        if err := cfg.Unmarshal(&newConfig); err != nil {
            log.Printf("配置重载失败: %v", err)
            return
        }

        // 更新应用配置
        updateApplicationConfig(&newConfig)
        log.Println("配置重载成功")
    })

    // 设置优雅关闭
    go func() {
        <-shutdownSignal()
        log.Println("停止配置监听...")
        stopWatch()  // 或者调用 cancel()
    }()

    return cancel
}
```

### 步骤 3: 线程安全的配置更新

使用 `atomic.Value` 确保配置更新的原子性：

```go
type Application struct {
    config atomic.Value  // 存储 *AppConfig
    cfg    *sysconf.Config
}

func (app *Application) StartConfigWatch() {
    app.cfg.Watch(func() {
        log.Println("配置文件变化，开始重新加载...")

        // 加载新配置
        var newConfig AppConfig
        if err := app.cfg.Unmarshal(&newConfig); err != nil {
            log.Printf("配置解析失败: %v", err)
            return
        }

        // 验证新配置
        if err := app.validateConfig(&newConfig); err != nil {
            log.Printf("配置验证失败: %v, 回滚到旧配置", err)
            return
        }

        // 原子性更新配置
        oldConfig := app.config.Load().(*AppConfig)
        app.config.Store(&newConfig)

        log.Printf("配置更新成功: 端口 %d -> %d",
            oldConfig.Server.Port,
            newConfig.Server.Port)

        // 触发配置变更处理
        app.onConfigChange(oldConfig, &newConfig)
    })
}

func (app *Application) GetConfig() *AppConfig {
    return app.config.Load().(*AppConfig)
}
```

### 步骤 4: 配置变更事件处理

实现配置变更的差异检测和处理：

```go
func (app *Application) onConfigChange(old, new *AppConfig) {
    // 检测服务器配置变化
    if old.Server.Port != new.Server.Port {
        log.Printf("服务器端口变化: %d -> %d", old.Server.Port, new.Server.Port)
        app.restartServer(new.Server.Port)
    }

    // 检测日志级别变化
    if old.Logging.Level != new.Logging.Level {
        log.Printf("日志级别变化: %s -> %s", old.Logging.Level, new.Logging.Level)
        app.updateLogLevel(new.Logging.Level)
    }

    // 检测数据库连接池变化
    if old.Database.MaxConns != new.Database.MaxConns {
        log.Printf("数据库连接池大小变化: %d -> %d",
            old.Database.MaxConns,
            new.Database.MaxConns)
        app.resizeDatabasePool(new.Database.MaxConns)
    }

    // 发送配置变更通知
    app.notifyConfigChange(old, new)
}
```

### 步骤 5: 错误处理和回滚

实现配置验证失败时的回滚机制：

```go
func (app *Application) SafeConfigReload() {
    // 保存当前配置作为备份
    backup := app.config.Load().(*AppConfig)

    app.cfg.Watch(func() {
        log.Println("尝试重新加载配置...")

        // 加载新配置
        var newConfig AppConfig
        if err := app.cfg.Unmarshal(&newConfig); err != nil {
            log.Printf("❌ 配置解析失败: %v", err)
            return
        }

        // 验证新配置
        if err := app.validateConfig(&newConfig); err != nil {
            log.Printf("❌ 配置验证失败: %v", err)
            log.Println("保持使用旧配置")
            return
        }

        // 尝试应用新配置
        if err := app.applyConfig(&newConfig); err != nil {
            log.Printf("❌ 配置应用失败: %v", err)
            log.Println("回滚到旧配置...")

            // 回滚到备份配置
            app.applyConfig(backup)
            app.config.Store(backup)
            return
        }

        // 更新成功
        app.config.Store(&newConfig)
        log.Println("✅ 配置更新成功")
    })
}

func (app *Application) validateConfig(cfg *AppConfig) error {
    // 自定义配置验证逻辑
    if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
        return fmt.Errorf("无效的服务器端口: %d", cfg.Server.Port)
    }

    if cfg.Database.MaxConns < 1 {
        return fmt.Errorf("数据库连接池大小必须大于 0")
    }

    // 更多验证...
    return nil
}
```

### 步骤 6: 防抖机制和批量更新

利用 sysconf 内置的防抖机制：

```go
func main() {
    cfg, _ := sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName("app"),
    )

    // sysconf 内置防抖机制：
    // 默认 200ms 内多次配置变更只会触发一次回调（可通过 WithWatchDebounce 调整）
    cfg.Watch(func() {
        log.Println("配置更新（已防抖）")

        // 批量读取所有变更
        allSettings := cfg.AllSettings()
        log.Printf("更新后的完整配置: %+v", allSettings)

        // 批量应用配置
        applyAllSettings(allSettings)
    })

    // 模拟频繁的配置文件修改
    // 只会触发一次回调（默认 200ms 内）
    editConfigFileMultipleTimes()
}
```

### 步骤 7: 定时取消监听

```go
func setupTimedConfigWatch(cfg *sysconf.Config, duration time.Duration) {
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()

    cfg.WatchWithContext(ctx, func() {
        log.Println("配置已更新")
    })

    log.Printf("配置监听将在 %v 后自动停止", duration)

    // 等待超时或手动取消
    <-ctx.Done()
    log.Println("配置监听已停止")
}

// 使用示例
setupTimedConfigWatch(cfg, 1*time.Hour)  // 1小时后停止监听
```

## 模板和示例

### 模板文件

- `templates/hot-reload.go.tmpl` - 热重载集成模板

### 示例代码

- `examples/watch-config.go` - 配置监听示例
- `examples/atomic-update.go` - 原子更新示例

## 热重载工作原理

```
文件系统事件 → fsnotify 监听 → 防抖过滤 (默认 200ms) → 验证新配置 → 原子更新 → 触发回调
                                    ↓
                                失败回滚
```

**关键特性**：

1. **防抖**: 默认 200ms 内多次变更合并为一次
2. **验证**: 加载前验证配置有效性
3. **原子**: 使用 `atomic.Value` 保证无锁读取
4. **回滚**: 验证失败自动保持旧配置
5. **事件**: 触发用户定义的回调函数

## 最佳实践

### 1. 使用原子存储

```go
// ✅ 推荐：使用 atomic.Value
type App struct {
    config atomic.Value  // *AppConfig
}

func (app *App) UpdateConfig(new *AppConfig) {
    app.config.Store(new)
}

func (app *App) GetConfig() *AppConfig {
    return app.config.Load().(*AppConfig)
}

// ❌ 避免：使用锁保护读取
type App struct {
    mu     sync.RWMutex
    config *AppConfig
}
```

### 2. 配置验证

```go
// ✅ 推荐：先验证再应用
cfg.Watch(func() {
    var newConfig AppConfig
    if err := cfg.Unmarshal(&newConfig); err != nil {
        log.Printf("解析失败: %v", err)
        return  // 不应用无效配置
    }

    if err := validateConfig(&newConfig); err != nil {
        log.Printf("验证失败: %v", err)
        return  // 不应用无效配置
    }

    applyConfig(&newConfig)
})

// ❌ 避免：直接应用未验证的配置
```

### 3. 错误处理

```go
// ✅ 推荐：完善的错误处理
cfg.Watch(func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("配置重载 panic: %v", r)
        }
    }()

    if err := reloadConfig(); err != nil {
        log.Printf("配置重载失败: %v", err)
        rollbackConfig()
    }
})

// ❌ 避免：忽略错误
cfg.Watch(func() {
    reloadConfig()  // 可能失败但不处理
})
```

### 4. 优雅关闭

```go
// ✅ 推荐：使用 context 控制生命周期
ctx, cancel := context.WithCancel(context.Background())
stopWatch := cfg.WatchWithContext(ctx, callback)

// 在关闭时停止监听
func shutdown() {
    log.Println("停止配置监听...")
    stopWatch()
    // 或 cancel()
}

// ❌ 避免：无法停止的监听
cfg.Watch(callback)  // 无法控制何时停止
```

### 5. 配置变更通知

```go
// ✅ 推荐：使用事件系统
type ConfigChangeEvent struct {
    Old *AppConfig
    New *AppConfig
}

func (app *App) notifyConfigChange(old, new *AppConfig) {
    event := ConfigChangeEvent{Old: old, New: new}
    app.eventBus.Publish("config.changed", event)
}

// 订阅者可以监听配置变更
app.eventBus.Subscribe("config.changed", func(e ConfigChangeEvent) {
    // 处理配置变更
})
```

## 故障排查

**问题: 配置监听未触发**

- 确认配置文件路径正确
- 检查文件是否有写入权限
- 验证文件系统支持 fsnotify（某些网络文件系统不支持）
- 检查是否使用了 Docker 卷挂载（可能需要特殊配置）

**问题: 配置更新后应用未生效**

- 确认回调函数被正确执行
- 检查配置验证是否失败
- 验证是否使用了原子存储（`atomic.Value`）
- 检查日志是否有错误信息

**问题: 频繁触发配置重载**

- sysconf 已内置默认 200ms 防抖，可通过 WithWatchDebounce 调整
- 检查编辑器是否创建临时文件（部分编辑器会）
- 考虑增加防抖时间

**问题: 配置回滚失败**

- 确保备份配置在验证通过后才被替换
- 检查回滚逻辑是否正确实现
- 验证配置验证逻辑的完整性

## 性能提示

- 配置监听使用 fsnotify，性能开销极小（<1% CPU）
- 防抖机制避免频繁回调，提升性能
- `atomic.Value` 读取无锁，性能优于 `RWMutex`
- 建议配置文件大小控制在 1MB 以下
- 避免在回调中执行耗时操作（使用 goroutine）

## 生产环境建议

1. **监控配置变更**

   - 记录所有配置变更事件
   - 监控配置重载成功率
   - 设置配置变更告警

2. **测试配置更新**

   - 在测试环境先验证配置变更
   - 准备回滚方案
   - 分批次更新生产环境

3. **配置审计**

   - 记录谁在何时修改了配置
   - 保留配置变更历史
   - 实现配置版本控制

4. **错误恢复**
   - 实现自动回滚机制
   - 设置配置验证规则
   - 准备配置备份

## 相关资源

- fsnotify 文档: https://github.com/fsnotify/fsnotify
- atomic.Value 文档: https://pkg.go.dev/sync/atomic#Value
- 热重载实现: /workspace/config.go (Watch 方法)
- 主集成技能: `integrating-sysconf`
- 验证技能: `configuring-validation`
