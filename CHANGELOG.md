# Changelog

本文件记录 sysconf 配置管理库的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.1.1] - 2026-06-16

### 新增 (Added)

- **缓存防抖调度** (`cache.go`)
  - 引入防抖定时器，合并短时间内多次缓存更新
  - 减少频繁写入场景下的锁竞争与文件 IO 抖动
  - 修复 `Close` 时未停止缓存定时器导致的阻塞

- **直接内容解析** (`config.go`)
  - 新增直接内容解析路径，跳过 viper 直接加载 YAML/JSON
  - 显著降低初次加载与热重载的延迟

- **隐藏配置文件加载** (`config_file.go`)
  - 支持通过精确文件名加载隐藏配置文件（如 `.env`）

### 变更 (Changed)

- **Go 版本要求** 升级至 1.25，配套更新相关依赖
- 使用 `maps.Copy`、`slices.Contains` 等标准库函数简化代码
- `config.go` 重构内部数据通路以适配新的直接解析路径

## [1.1.0] - 2026-06-15

### ⚠ 破坏性变更 (Breaking Changes)

- **默认构造与配置访问 API 调整**
  - 默认构造行为与配置访问入口有调整，使用方需按迁移指南更新调用
  - 详见 `MIGRATION.md`

## [1.0.0] - 2024-XX-XX

### 初始版本

- 高性能配置管理
- 线程安全的并发访问
- 多格式支持 (YAML, JSON, TOML, Dotenv)
- 智能验证系统
- ChaCha20-Poly1305 加密
- 环境变量集成
- 热重载支持
- Cobra/PFlag 集成
