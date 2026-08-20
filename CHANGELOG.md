# Changelog

本文件记录 sysconf 配置管理库的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增 (Added)

- **分层合并与统一读路径** (`merge.go`)
  - 有效配置优先级：**显式 Set > 命令行 flag > 环境变量 > 文件/内联内容 > getter/struct 默认值**
  - `Get*` / `GetAs` / `Unmarshal` / `AllSettings` / `Load` 走同一条路径
  - 热重载后显式 `Set` 仍保留；写盘只持久化文件层 + Set，不把 env/flag 写回文件
  - `WithFiles` / `WithOverlay` 多文件叠加（后者覆盖前者）；`Watch`/`OnChange` 同时监听叠加文件
  - `Flush()` 立即落盘并返回防抖写入错误
  - `OnChange` / `OnChangeWithContext` 回调携带 prev/next 快照；校验失败时内存保持旧值
  - `WithSkipInitValidation`：跳过 New/Load/热重载的整份文档校验（Set 字段级校验仍生效）。存在验证器时 New 默认 fail-fast

- **Load / MustLoad / MustNew / UnmarshalAs** (`load.go`)
  - `Load[T](opts...)` 一次完成 New + Unmarshal，覆盖最常见的结构体启动路径
  - `MustLoad` / `MustNew` 用于 main 中的必达初始化
  - `UnmarshalAs[T](cfg, key...)` 直接返回解码后的值，避免先声明零值
  - `MustUnmarshal` 失败即 panic

- **取值默认值对齐** (`getter.go`, `generic_getter.go`)
  - `GetDuration` / `GetTime` / `Get*Slice` / `GetSliceAs` 与 `GetString` 一样支持可选默认值

- **可判断的取值错误** (`config.go`)
  - 新增 `ErrKeyNotFound` / `ErrConversion` / `ErrEmptyKey` / `ErrNilConfig`
  - `GetWithError` / `GetAsWithError` 可用 `errors.Is` 区分缺失与类型转换失败

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

### 变更 (Changed)

- **有效配置优先级** 现为 **Set > flag > env > 文件 > 默认值**。运行时 `Set` 高于环境变量与命令行；写盘不持久化 env/flag。
- **New/Load 启动校验**：只要挂了验证器，默认对整份有效配置 fail-fast（`errors.Is(err, ErrValidation)`）。预定义验证器若只用于 Set 字段拦截，请加 `WithSkipInitValidation()`。

### 性能优化 (Performance)

- **Get/Set 热路径指标采集去锁** (`metrics.go:41-95`, `getter.go:21`, `setter.go:15`, `setter.go:293`)
  - `Metrics.RecordGet/RecordSet` 改为全原子操作：`LastGetTime`/`LastSetTime`
    以 `atomic.Int64` 存储 UnixNano（`lastGetNs`/`lastSetNs`），不再持有 `mu`
  - `Get`/`Set`/`SetMultiple` 仅在 `metricsEnabled` 为真时才计时，关闭指标采集时
    零 `time.Now()` 与 defer 闭包开销
  - 实测 `BenchmarkConfigGetConcurrent`：235ns/op → ~90ns/op（并发不再因指标锁而慢于串行）

- **环境变量读路径去锁** (`config.go:101`, `config.go:761`, `config.go:1217`)
  - 新增 `envSnapshot atomic.Pointer[EnvOptions]` 只读快照，`initializeEnv` 每次
    初始化/`SetEnvPrefix` 后发布新快照
  - `lookupEnvValue` 从 `mu.RLock()` 读取 `envOptions` 改为无锁读取快照

- **fetchFromViperOrEnv 分支合并** (`config.go:1237`)
  - 合并两个重复的 `viper != nil && viperLoaded` 判断为提前返回，逻辑等价，
    消除重复条件

### 修复 (Fixed)

- **Marshal 把整份文件打成 Set 覆盖层** (`marshal.go`)
  - `deepMerge(snapshotPersistentSettings)` 再 `SetMultiple` 会钉死无关键，env/Watch 失效
  - 现只提交结构体自身字段

- **flag 快照在 New 后不再更新** (`merge.go`)
  - `Parse` 发生在 `New` 之后时 Get 仍读初始化默认值
  - 现每次取值现场收集 pflag

- **非法热重载吃掉防抖窗口** (`config.go`)
  - 先到先处理会丢掉窗口内的后续合法写入
  - 现改为尾沿防抖：安静期结束后按磁盘最终内容重载

- **校验失败后 viper 与 Get 分叉** (`config.go`)
  - 只回滚原子文件层，`cfg.Viper()` 仍是被拒绝的内容
  - 现 `restoreFileLayer` 同步回写 viper

- **Flush 写盘失败后丢失待写入状态** (`setter.go`)
  - `pendingWrites` 在 `WriteFile` 成功前就被清掉，失败后再 `Flush` 会空转
  - 现失败时恢复 pending，成功才清掉 `lastWriteErr`

- **Close 在落盘失败后仍标记已关闭** (`config.go`)
  - 待写入无法持久化时仍关闭资源，无法重试
  - 现先在锁内标记关闭以拒绝并发 Set，落盘失败再恢复未关闭

- **校验失败回滚用 viper.Set 钉死文件键** (`config.go`)
  - 合法热重载后 `Get` 已更新，`Viper()` 仍停在被拒绝的旧值
  - 现用 `ReadConfig` 重写 viper 文件层，保留显式 Set 覆盖

- **叠加层用 viper.Set，删除的键无法回退** (`config.go`)
  - overlay 去掉的键 `Get` 回到主文件，`Viper()` 仍是旧叠加值
  - 现按主文件+叠加独立合并后整体替换 viper 文件层

- **New 之后才 Changed 的 flag 未 BindPFlag** (`config.go`)
  - `Get`/`AllSettings` 能看到 `Parse`/`FlagSet.Set`，`Viper()` 仍读文件
  - 现 `Viper()` 在返回前按当前 Changed 状态补绑

- **空值 env-only 键进不了快照** (`config.go`)
  - `PREFIX_TOKEN=` 时 `Get`/`IsSet` 为真，`AllSettings`/`Unmarshal` 没有该键
  - 现扫描前缀环境变量时保留空值

- **Set 提交后、标记 pending 前 Close 会漏写** (`setter.go`)
  - `storeData` 已对 Get 可见，`pendingWrites` 仍在锁外 `scheduleWrite` 才置位
  - Close 认为无待写，文件停在旧值；现与提交同锁标记 pending

- **Close 与 Flush 锁顺序相反导致死锁** (`config.go`)
  - Close 持 `writeMu` 再取 `mu`，Flush/写盘定时器相反
  - 现写盘结束后再改 pending/closed；已关闭时防抖写盘直接退出，避免把 `ErrAlreadyClosed` 记成写失败

- **toml 热重载把 YAML 喂给 toml Viper** (`config_file.go`)
  - hydrate 序列化不含 toml，回退 YAML 后 `ReadConfig` 失败，Get 回滚而 Viper 已是新值
  - 现补齐 toml/dotenv 序列化，hydrate 不再跨格式回退

- **叠加文件变更不会触发热重载** (`config.go`)
  - viper 只 Watch 主文件，`WithFiles`/`WithOverlay` 后写的文件改了内存不更新
  - 现对叠加文件所在目录做 fsnotify，尾沿防抖后重读全部叠加层

- **AllSettings/Keys 看不到仅存在于 env 的键** (`merge.go`)
  - `Get`/`Load` 能读到 `PREFIX_TOKEN`，快照却没有，来源分叉
  - 现把前缀 env 与 SmartCase 别名并进有效扁平快照

- **小写 snake_case 误打到点号嵌套键** (`merge.go`)
  - `Get("max_conns")` 会命中 `max.conns`
  - 现仅对全大写环境变量风格键做 `_` → `.` 回退

- **文件层被 viper.AllSettings 污染** (`config.go`)
  - `AutomaticEnv` / `BindPFlag` 会使 `AllSettings()` 带上 env/flag，写盘会把密钥写进配置文件
  - 现从文件/内容独立解析文件层，与有效配置分离

- **SmartCase 混合大小写前缀** (`config.go`)
  - 离开 viper.Get 后，`MyApp_Cache_Timeout` 这类无法由 `deriveEnvKeys` 穷举的名字会丢失
  - 现把 `bindSmartCaseEnvVars` 扫到的真实 env 名纳入 `lookupEnvValue`

- **GetAs 绕过统一读路径** (`generic_getter.go`)
  - 原先先读 `readCache`，缓存命中时不会走 `getRaw`
  - 现与 `Get*` 共用 `getRaw`，优先级一致

- **Unmarshal/AllSettings 看不到环境变量** (`config.go`)
  - `snapshotAllSettings` 只读原子存储，`Unmarshal` 与 `Get*` 对同一键会读到不同来源
  - 现对扁平快照叠加 env，结构体解码与 AllSettings 与 Get* 一致

- **默认验证器误判 email.smtp.host** (`validation/validator.go`)
  - `validateEmail` 用 `strings.Contains(key, "email")`，把 SMTP 主机当成邮箱
  - 现只匹配叶子键 `email`/`mail`/`*_email`/`*_mail`

- **Marshal 非原子写入与 O(N²) 开销** (`marshal.go:18-48`)
  - 原实现 `setMapToViper` 对每个叶子键调用一次 `Set`，N 个键即触发 N 次全量
    深拷贝（O(N²)），且中途某个键校验失败会留下部分写入的脏数据
  - 改为 `flattenForSet` 递归拍平为点分隔键后，一次性调用 `SetMultiple`
    批量提交：O(N) 开销，单次校验 + 单次写盘，失败可整体回滚

- **Unmarshal 潜在死锁** (`unmarshal.go:57-65`)
  - 移除多余的外层 `c.mu.RLock()`：该锁会在内部 `getRaw → fetchFromViperOrEnv`
    处重入 `sync.RWMutex`（Go 的 RWMutex 不可重入），与并发 `Set` 竞争时可能死锁
  - `getRaw`/`snapshotAllSettings` 自身已保证并发安全，函数其余部分只操作
    调用方传入的 `obj`，无需外层加锁

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

- **WithEnv 空前缀不再启用环境变量** (`options.go`)
  - `WithEnv("")` 过去会无前缀盲查系统变量，`Get("path")` 被 `PATH` 静默覆盖
  - 现空前缀视为未启用；确需无前缀请显式 `WithEnvOptions`
  - 无前缀启用时 `initializeEnv` 会打警告

- **Watch 返回 error** (`config.go`)
  - `Watch()` 在实例已关闭时返回 `ErrAlreadyClosed`，不再静默丢弃失败
  - `WatchWithContext` 签名不变

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
