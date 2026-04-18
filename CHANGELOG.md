# Changelog

本文件记录 sysconf 配置管理库的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增 (Added)

- **SetMultiple 批量设置方法** (`setter.go:280-387`)
  - 支持一次性设置多个配置项
  - 减少重复验证和文件写入开销
  - 原子性操作：任一键验证失败则全部回滚

- **GetBool 增强** (`getter.go:100-150`)
  - 支持 `"yes"/"no"` 布尔表示
  - 支持 `"on"/"off"` 布尔表示
  - 支持数字类型 `0/非零` 转换为布尔值

- **Metrics 累积统计** (`metrics.go:10-17`)
  - 新增 `OperationStats` 结构体记录操作统计
  - 包含 Count、TotalNs、MinNs、MaxNs、LastNs 字段
  - 向后兼容：保留 `OperationTimes` 最后一次操作时间

- **MatchName 缓存** (`unmarshal.go:16-28`)
  - 使用 `sync.Map` 缓存字段名匹配结果
  - 避免重复计算驼峰/蛇形转换
  - 提升 Unmarshal 性能

- **Metrics 并发测试** (`metrics_extra_test.go:36-88`)
  - 新增 `TestMetricsConcurrency` 并发安全测试
  - 新增 `TestOperationStats` 累积统计测试

### 修复 (Fixed)

- **marshal.go 并发安全** (`marshal.go:19-35`)
  - 修复 `Marshal` 方法未获取读锁的问题
  - 防止并发读写导致的数据竞争

- **validation/rules.go 线程安全** (`validation/rules.go:42-60`)
  - 新增 `validatorsMu sync.RWMutex` 保护全局 validators map
  - `RegisterValidator` 使用写锁
  - `ValidateValue` 使用读锁

- **crypto.go 时序攻击漏洞** (`crypto.go:175-179`)
  - 使用 `crypto/subtle.ConstantTimeCompare` 替代字节比较
  - 防止通过时序分析推断加密前缀

- **IPv6 验证** (`validation/rules.go:257-270`)
  - 使用 `net.ParseIP()` 替代正则表达式
  - 正确支持压缩形式（如 `::1`、`fe80::1`）

- **Email 验证** (`validation/rules.go:106-130`)
  - 添加连续点号检查
  - 禁止以点号开头或结尾的邮箱地址

- **Base64 验证** (`validation/rules.go:345-360`)
  - 使用 `base64.StdEncoding.DecodeString()` 验证
  - 正确验证长度和填充

- **Hostname 验证 ReDoS** (`validation/rules.go:293-310`)
  - 添加 253 字符长度限制
  - 预编译正则表达式避免 ReDoS 风险

### 变更 (Changed)

- **密钥导出函数** (`crypto.go:65-140`)
	- 使用 Argon2id 替代直接 SHA256
	- 参数：time=1, memory=64MB, threads=4, keyLen=32
	- 密文格式包含版本与盐值，避免密钥不可复现

- **deepMerge 优化** (`marshal.go:93-122`)
  - 原地修改 m1 替代创建新 Map
  - 减少内存分配和 GC 压力

- **错误消息改进** (`unmarshal.go:116-130`)
  - 区分类型转换错误、字段缺失错误
  - 提供更精确的错误分类

### 安全 (Security)

- **Argon2id 密钥导出**: 使用内存硬函数防止暴力破解
- **常量时间比较**: 防止时序攻击
- **ReDoS 防护**: 主机名验证添加长度限制

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
