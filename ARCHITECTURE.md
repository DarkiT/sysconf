# 架构设计文档

本文档描述 sysconf 配置管理库的整体架构设计，包括核心组件、数据流、并发模型和扩展机制。

## 📋 目录

- [架构概览](#架构概览)
- [核心组件](#核心组件)
- [并发模型](#并发模型)
- [数据流](#数据流)
- [扩展机制](#扩展机制)
- [性能设计](#性能设计)

## 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                         Sysconf 架构                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Getter    │  │   Setter    │  │  Unmarshal  │              │
│  │   API 层    │  │   API 层    │  │   API 层    │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                      │
│         └────────────────┼────────────────┘                      │
│                          │                                       │
│  ┌───────────────────────┴───────────────────────┐              │
│  │              Config 核心结构体                 │              │
│  │  ┌─────────────────────────────────────────┐  │              │
│  │  │  data atomic.Value (map[string]any)    │  │              │
│  │  │  mu sync.RWMutex                        │  │              │
│  │  │  readCache atomic.Value (读取缓存)      │  │              │
│  │  │  envSnapshot atomic.Pointer[EnvOptions] │  │              │
│  │  │  validators []ConfigValidator          │  │              │
│  │  │  crypto ConfigCrypto                   │  │              │
│  │  └─────────────────────────────────────────┘  │              │
│  └───────────────────────┬───────────────────────┘              │
│                          │                                       │
│         ┌────────────────┼────────────────┐                      │
│         │                │                │                      │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐              │
│  │    Cache    │  │ Validation  │  │   Crypto    │              │
│  │    模块     │  │    模块     │  │    模块     │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  底层依赖层                              │    │
│  │  fsnotify │ mapstructure │ yaml.v3 │ cast │ pflag       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. Config 结构体

Config 是 sysconf 的核心，采用**原子存储架构**实现线程安全：

```go
type Config struct {
    // 核心数据存储 - 使用 atomic.Value 实现无锁读取
    data atomic.Value // 存储 map[string]any
    
    // 并发控制
    mu sync.RWMutex // 保护元数据和写操作
    wg sync.WaitGroup // Go 1.25+ WaitGroup.Go()，等待后台 goroutine 退出
    
    // 读取缓存（直接内嵌在 Config 上，无独立 Cache 类型）
    readCache    atomic.Value // map[string]any，无锁读
    cacheEnabled atomic.Bool
    cacheTimer   *time.Timer  // 防抖重建定时器
    
    // 环境变量无锁读路径
    envSnapshot atomic.Pointer[EnvOptions] // 只读快照，供读路径免锁访问
    
    // 功能模块
    validators  []ConfigValidator   // 验证器链
    crypto      ConfigCrypto        // 加密器
    
    // 配置选项
    path        string
    name        string
    mode        string
    // ... 更多选项
}
```

**设计要点**：
- `atomic.Value` 实现**无锁读取**，微秒级性能
- `sync.RWMutex` 保护**元数据**和**写操作**；`envSnapshot` 等高频读字段则用独立的
  `atomic.Pointer`/`atomic.Int64` 只读快照，连 `RWMutex.RLock()` 都不占，进一步降低热路径争抢
- 数据与元数据分离，优化并发访问模式

### 2. Getter API 层

提供类型安全的配置读取：

```go
// 基础类型获取
func (c *Config) GetString(key string, def ...string) string
func (c *Config) GetInt(key string, def ...int) int
func (c *Config) GetBool(key string, def ...bool) bool

// 泛型 API（Go 1.24+）
func GetAs[T any](c *Config, key string, def ...T) T
func MustGetAs[T any](c *Config, key string) T

// 内部实现
func (c *Config) loadData() map[string]any {
    if data := c.data.Load(); data != nil {
        return data.(map[string]any)
    }
    return make(map[string]any)
}
```

**性能优化**：
- 读取操作**无锁**，直接访问 `atomic.Value`
- 缓存热点数据，减少重复查找
- 支持嵌套键访问（如 `database.host`）

### 3. Setter API 层

提供线程安全的配置写入：

```go
// 单键设置
func (c *Config) Set(key string, value any) error

// 批量设置（原子性）
func (c *Config) SetMultiple(values map[string]any) error
```

**写入流程**：
1. 获取写锁 (`mu.Lock()`)
2. 深拷贝当前数据（防御性复制）
3. 应用变更
4. 验证新值（如果配置了验证器）
5. 原子性更新 `data.Store(newData)`
6. 释放写锁
7. 异步持久化（防抖写入）

### 4. 验证系统 (Validation)

分层验证架构：

```
┌─────────────────────────────────────┐
│         验证器接口                   │
│  type ConfigValidator interface {   │
│      Validate(map[string]any) error │
│      GetName() string               │
│  }                                  │
└─────────────────────────────────────┘
                   │
       ┌───────────┼───────────┐
       │           │           │
┌──────▼─────┐ ┌───▼────┐ ┌────▼─────┐
│ 预定义验证器 │ │规则验证器│ │函数验证器 │
└────────────┘ └────────┘ └──────────┘
```

**验证器类型**：
- **预定义验证器**：针对特定场景（数据库、Redis、Web服务器等）
- **规则验证器**：基于规则引擎的灵活验证（30+ 内置规则）
- **函数验证器**：自定义验证逻辑

**字段级验证优化**：
```go
func (c *Config) validateSingleField(key string, value any) error {
    // 只验证相关的验证器和字段
    for _, validator := range c.validators {
        if !c.validatorSupportsField(validator, key) {
            continue // 跳过不相关的验证器
        }
        // 执行单字段验证
        if err := c.validateField(validator, key, value); err != nil {
            return err
        }
    }
    return nil
}
```

### 5. 加密模块 (Crypto)

ChaCha20-Poly1305 认证加密：

```
明文配置 ──► Argon2id 密钥派生 ──► ChaCha20-Poly1305 加密 ──► 密文存储
                │
                └── 参数: time=1, memory=64MB, threads=4
```

**安全特性**：
- **Argon2id**：内存硬函数，抵抗暴力破解
- **ChaCha20-Poly1305**：AEAD 认证加密
- **常量时间比较**：防止时序攻击
- **密文格式**：包含版本、盐值、Nonce、密文、认证标签

### 6. 缓存模块 (Cache)

缓存字段直接内嵌在 `Config` 结构体上（`cache.go`），没有独立的 `Cache` 类型：

```go
// Config 中与缓存相关的字段（节选）
readCache         atomic.Value // 只读缓存，map[string]any，无锁读
cacheEnabled      atomic.Bool  // 缓存开关
cacheVersion      int64        // 缓存版本号
cacheMu           sync.Mutex   // 缓存更新互斥锁
cacheBuildMu      sync.Mutex   // 防止并发 updateReadCache 访问 viper map
cacheTimer        *time.Timer  // 防抖重建定时器
cacheWarmupDelay  time.Duration
cacheRebuildDelay time.Duration
```

**缓存策略**：
- **延迟预热** (`cacheWarmupDelay`)：`enableReadCache` 在创建时调度一次异步预热，避免启动时大量缓存未命中
- **延迟重建** (`cacheRebuildDelay`)：`invalidateCache` 先清空缓存，再用 `time.AfterFunc` 防抖调度重建，批量吸收短时间内的多次写入
- **原子更新**：`updateReadCache` 通过 `snapshotAllSettings` 取得一致快照后，用 `atomic.Value.Store` 整体替换缓存，读路径永远拿到完整一致的 map
- **优雅退出**：重建任务纳入 `Config.wg`（`wg.Go(...)`），`Close()` 的 `wg.Wait()` 能等到最后一次重建完成再返回

## 并发模型

### 读写分离架构

```
         读取操作                    写入操作
            │                          │
            ▼                          ▼
   ┌─────────────────┐        ┌─────────────────┐
   │  atomic.Value   │        │  sync.RWMutex   │
   │   (无锁读取)     │        │   (写保护)       │
   └─────────────────┘        └─────────────────┘
            │                          │
            ▼                          ▼
   ┌─────────────────┐        ┌─────────────────┐
   │   微秒级延迟     │        │   毫秒级延迟     │
   │   高并发支持     │        │   数据一致性     │
   └─────────────────┘        └─────────────────┘
```

### 并发安全保证

1. **读取路径**（无锁）：
   ```go
   func (c *Config) GetString(key string, def ...string) string {
       data := c.loadData() // atomic.Value.Load() - 无锁
       value, exists := data[key]
       // ...
   }
   ```

2. **写入路径**（加锁）：
   ```go
   func (c *Config) Set(key string, value any) error {
       c.mu.Lock()
       defer c.mu.Unlock()
       
       oldData := c.loadData()
       newData := deepCopy(oldData) // 防御性复制
       newData[key] = value
       
       c.data.Store(newData) // 原子更新
       // ...
   }
   ```

3. **验证路径**（无锁读取）：
   ```go
   func (c *Config) validate(data map[string]any) error {
       // 读取 validators 切片（创建时确定，无需加锁）
       for _, v := range c.validators {
           if err := v.Validate(data); err != nil {
               return err
           }
       }
       return nil
   }
   ```

4. **统计与环境变量读取路径**（全原子，连 RLock 都不占）：
   ```go
   // Metrics.RecordGet/RecordSet 全程走 atomic，无 mu.Lock()
   func (m *Metrics) RecordGet(duration time.Duration, cacheHit bool) {
       atomic.AddInt64(&m.GetCount, 1)
       m.lastGetNs.Store(time.Now().UnixNano()) // 而非 mu.Lock() 后赋值 time.Time
   }

   // lookupEnvValue 读取 envSnapshot 快照，而非对 envOptions 加 mu.RLock()
   func (c *Config) lookupEnvValue(key string) (any, bool) {
       snapshot := c.envSnapshot.Load()
       if snapshot == nil || !snapshot.Enabled {
           return nil, false
       }
       // ...
   }
   ```
   Get/Set 默认开启指标采集，是访问频率最高的两个入口；一旦这里出现锁竞争，
   前面 `atomic.Value` 带来的无锁读优势会被直接抵消。因此这两条路径不容许出现
   `sync.Mutex`/`sync.RWMutex`，只能用 `atomic.*`。

### 竞态检测

项目通过 `go test -race` 进行全面竞态检测：

```go
func TestConcurrentSafety(t *testing.T) {
    cfg, _ := sysconf.New(/* 选项 */)
    
    var wg sync.WaitGroup
    // 并发写入
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            cfg.Set(fmt.Sprintf("key%d", id), id)
        }(i)
    }
    // 并发读取
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = cfg.GetString("key0")
        }()
    }
    wg.Wait()
}
```

## 数据流

### 配置加载流程

```
配置文件 ──► 解析 (yaml/json/toml) ──► 解密 (如启用) ──► 验证 ──► 存储
                                              │
                                              ▼
默认值 ──────────────────────────────────────► 合并
```

### 配置写入流程

```
Set(key, value) ──► 深拷贝数据 ──► 应用变更 ──► 验证 ──► 原子存储 ──► 异步持久化
                      │                                               │
                      │                                               ▼
                      └────────────────────────────────────────► 防抖写入文件
```

### 热重载流程

```
文件变更 ──► fsnotify 监听 ──► 防抖 (200ms) ──► 重新加载 ──► 验证 ──► 原子更新 ──► 回调通知
```

## 扩展机制

### 1. 选项模式 (Functional Options)

```go
type Option func(*Config) error

func WithPath(path string) Option {
    return func(c *Config) error {
        c.path = path
        return nil
    }
}

func WithEncryption(password string) Option {
    return func(c *Config) error {
        c.crypto = NewChaCha20Crypto(password)
        return nil
    }
}

// 使用
cfg, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithEncryption("secret"),
    sysconf.WithValidators(v1, v2),
)
```

### 2. 验证器接口

```go
type ConfigValidator interface {
    Validate(config map[string]any) error
    GetName() string
}

// 自定义验证器
type MyValidator struct{}

func (v *MyValidator) Validate(config map[string]any) error {
    // 自定义验证逻辑
    return nil
}

func (v *MyValidator) GetName() string {
    return "MyValidator"
}
```

### 3. 加密器接口

```go
type ConfigCrypto interface {
    Encrypt(data []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    IsEncrypted(data []byte) bool
}
```

## 性能设计

### 读取性能

| 操作 | 实现方式 | 延迟 |
|------|----------|------|
| `GetString` | atomic.Value.Load() | ~100ns |
| `GetInt` | 类型转换 + 缓存 | ~200ns |
| `GetAs[T]` | 泛型 + 缓存 | ~300ns |
| 嵌套键 | 路径解析 + 缓存 | ~500ns |

### 写入性能

| 操作 | 实现方式 | 延迟 |
|------|----------|------|
| `Set` | 深拷贝 + 验证 + 原子存储 | ~1-5ms |
| `SetMultiple` | 批量处理 + 单次验证 | ~5-10ms |
| 持久化 | 防抖写入（3秒延迟） | 异步 |

### 内存优化

1. **写时复制 (COW)**：写入时深拷贝数据，读取共享不变数据
2. **缓存预热**：延迟填充缓存，避免冷启动开销
3. **对象池**：复用临时对象，减少 GC 压力

### 基准测试结果

```
BenchmarkConfig/SequentialReads-8     10000000    105 ns/op
BenchmarkConfig/ConcurrentReads-8     10000000    112 ns/op
BenchmarkConfig/MixedOperations-8      5000000    285 ns/op
```

## 设计决策记录

### 1. 为什么选择 atomic.Value 而不是 sync.Map？

- **atomic.Value** 适合**读多写少**场景，读取性能更高
- 配置数据通常是整体替换，而非细粒度更新
- 实现更简单，无需处理 map 的并发安全

### 2. 为什么写入需要深拷贝？

- 防止外部修改影响内部状态
- 保证原子性更新（替换整个 map）
- 支持配置回滚（保存历史版本）

### 3. 为什么使用防抖写入？

- 减少磁盘 I/O 次数
- 合并短时间内的多次更新
- 避免配置文件频繁变动导致的热重载抖动

### 4. 为什么 Metrics 与环境变量读取要从"加锁"改成"全原子"？

- `Get` 默认开启指标采集，`RecordGet` 曾经用 `mu.Lock()` 写 `LastGetTime`；
  高并发读场景下这把锁会成为瓶颈，实测 `BenchmarkConfigGetConcurrent`
  并发吞吐反而低于串行（235ns/op vs 209ns/op 串行）
- 改为 `atomic.Int64` 存储 UnixNano、`GetStats()` 时再转换回 `time.Time` 后，
  同一基准并发降到 ~90ns/op，锁竞争完全消失
- `envOptions` 同理：写入频率极低（仅初始化/`SetEnvPrefix` 时变更），
  读取频率极高（每次 `Get` 未命中原子存储都会探测环境变量），
  用 `atomic.Pointer[EnvOptions]` 发布只读快照，读路径不再需要 `mu.RLock()`
- 代价：`Reset()`/`SetEnvPrefix()` 等低频路径要多维护一份原子字段，
  但热路径的收益远大于这点复杂度

---

**文档版本**: 1.1
**最后更新**: 2026-07-04
**维护者**: darkit/sysconf 团队
