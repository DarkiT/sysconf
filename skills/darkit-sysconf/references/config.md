# 配置：选项、来源、读写

`sysconf.Load[T](opts...)` 一次完成 New + Unmarshal，是生产应用推荐入口。
`sysconf.New(opts ...Option)` 返回 `(*Config, error)`。`sysconf.Default(opts...)` 带默认值。
`sysconf.MustNew` / `MustLoad` 失败即 panic。

## 来源与选项

```go
sysconf.WithPath("config.yaml")        // 配置文件路径（按扩展名识别 yaml/toml/json...）
sysconf.WithFiles("base.yaml", "local.yaml") // 多文件，后者覆盖前者；第一个作 Watch/写回主文件
sysconf.WithOverlay("override.yaml")   // 在主文件上叠加
sysconf.WithName("config")             // 文件名（不含扩展）
sysconf.WithMode("yaml")               // 显式格式
sysconf.WithContent(yamlString)        // 内联内容（不读文件）
sysconf.WithEnv("MYAPP")               // 启用 env 覆盖，前缀 MYAPP_（务必带前缀）
sysconf.WithEnvSmartCase("MYAPP", true)
sysconf.WithBindPFlags(flagSet)        // 绑定 spf13/pflag 命令行
sysconf.WithSkipInitValidation()       // 跳过 New/Load/热重载整份校验；Set 字段校验仍生效
sysconf.WithLogger(logger)             // 自定义日志
sysconf.WithWriteDebounceDelay(500*ms) // Set 后回写文件的防抖
sysconf.WithWatchDebounce(300*ms)      // 文件变更热重载的防抖
sysconf.WithCacheTiming(warmup, rebuild)
```

优先级：**显式 Set > flag > env > 文件/默认**。`Load`/`Get*`/`GetAs`/`Unmarshal`/`AllSettings` 同一路径。写盘只含文件层与 Set。

## 类型 getter（方法）

```go
cfg.Get(key, def...) any
cfg.GetString(key, def...) / GetInt / GetBool / GetFloat
cfg.GetDuration(key) / GetTime(key)
cfg.GetStringSlice/GetIntSlice/GetBoolSlice/GetFloatSlice(key)
cfg.GetStringMap/GetStringMapString(key)
cfg.GetStringPath(...parts) / GetIntPath / GetBoolPath / GetFloatPath   // 变参路径
cfg.GetWithError(key) (any, error)
cfg.IsSet(key) / Keys() / AllSettings()
```

返回的 map/slice 均为**深拷贝**，改动不污染内部存储。

## 泛型 getter（包级函数）

```go
v := sysconf.GetAs[int](cfg, "server.port", 8080)     // 类型安全
v := sysconf.MustGetAs[uint16](cfg, "server.port")    // 失败 panic
```

支持 string / int / int8..int64 / uint..uint64 / float32/64 / bool / time.Duration / time.Time。类型不匹配时返回默认值/零值。

## setter

```go
cfg.Set(key, value) error
cfg.SetMultiple(map[string]any) error
```

`WithWriteDebounceDelay > 0` 时写盘是异步防抖的：`Set` 立即返回、稍后落盘。**延迟写失败仅记日志、不回滚**，视为最终一致；需要同步持久化保证时调用 `cfg.Flush()`，或把防抖设为 0。

## 编解码

```go
cfg.Unmarshal(&myStruct)          // 整份解析到 struct
cfg.Unmarshal(&sub, "server")     // 子树解析
cfg.Marshal(myStruct)             // struct 写回配置
cfg.Marshal(myStruct, "server")   // 写到指定前缀
```

struct tag 识别 `config`/`sysconf`/`json`/`toml` 等；`default` tag 填充默认值（含 uint 字段）。

## 环境变量陷阱

启用 env **必须带前缀**。无前缀时会对任意 key 盲查系统 env，`Get("path")→PATH`、`Get("home")→HOME` 等被系统变量无声覆盖。始终 `WithEnv("YOURPREFIX")`。显式 `Set` 优先于 env。
