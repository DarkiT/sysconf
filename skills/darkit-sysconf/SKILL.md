---
name: darkit-sysconf
description: "Use this skill whenever code integrates, configures, or debugs `github.com/darkit/sysconf` — a Go configuration library built on viper with functional options, layered sources (file/env/pflags/content), typed getters (Get/GetString/GetInt/GetAs[T]), setters with debounced write-back, hot reload/Watch, struct Marshal/Unmarshal, pluggable validators, and AEAD-encrypted config files. Triggers on: loading app config in Go, `sysconf.New`/`WithPath`/`WithEnv`/`WithValidator`/`WithEncryption`/`WithWatchDebounce`, `Config.Get*`/`GetAs[T]`/`Set`/`Watch`/`Unmarshal`/`Marshal`, env-var override, config validation rules, encrypted config, or config hot-reload. Use it even when the user just says 'load a YAML/TOML config in Go with env override', 'typed config getter', 'encrypted config file', 'watch config changes', or names the module without spelling out the API."
license: MIT
user-invocable: true
disable-model-invocation: false
---

# darkit-sysconf

本 Skill 是 `github.com/darkit/sysconf` 的 **接入入口法阵**，不是全文 API 手册：

- 先帮你判断任务落在哪条能力链
- 只加载当前任务最小必要的 `references/` 文档
- 直接给可复制的上手模板（`assets/examples/`）
- 提前挡掉几个真实存在、极易踩的坑（无前缀 env 撞系统变量、GetAs 类型支持、加密恢复语义）

模块定位：基于 viper 封装的 **Go 配置管理库**。函数式选项、多来源分层（文件/env/pflags/内容）、类型安全 getter（含泛型 `GetAs[T]`）、防抖回写、热重载、struct 编解码、可插拔校验器、AEAD 加密配置文件。

## 何时触发

任务涉及以下任一场景即使用：

- 基于 `github.com/darkit/sysconf` 加载/管理应用配置
- `sysconf.New(opts...)` + `WithPath/WithName/WithMode/WithEnv/WithContent/WithBindPFlags`
- 类型 getter：`Get/GetString/GetInt/GetBool/GetFloat/GetDuration/GetTime/GetAs[T]`
- setter：`Set/SetMultiple`，防抖写回，`WithWriteDebounceDelay`
- 热重载：`Watch/WatchWithContext`，`WithWatchDebounce`
- struct 编解码：`Unmarshal/Marshal`
- 配置校验：`WithValidator/WithValidateFunc/AddValidator`，`validation` 子包规则
- 加密配置：`WithEncryption/WithCrypto`，`NewDefaultCrypto/NewChaCha20Crypto`
- env 覆盖、嵌套 key、配置漂移排查
- 为该模块写示例、文档、测试

## 默认工作方式

1. 先从**调用方视角**理解任务，而不是从内部实现展开
2. 优先读 `references/` 里最贴近任务的**一个**文件
3. 优先复用 `assets/examples/` 的模板
4. 需要核对真实行为时，再到当前 workspace 搜 live code 与测试

## 最小心智模型

```
app, cfg, err := sysconf.Load[App](opts...)   // 推荐：New + Unmarshal
cfg, err := sysconf.New(opts...)              // *Config, error
   cfg.Get(key, def...) / cfg.GetString/GetInt/GetDuration(key, def...)
   sysconf.GetAs[T](cfg, key, def...)         // 包级泛型，与 Get* 同路径
   cfg.Set(key, value) / cfg.SetMultiple(map)
   cfg.Flush()
   cfg.Unmarshal(&struct{}) / sysconf.UnmarshalAs[T](cfg)
   cfg.Watch(callback) / cfg.OnChange(handler)
   cfg.Close()
```

- **优先级**：显式 `Set` > 命令行 flag > 环境变量 > 文件/内联内容 > getter/struct 默认值。`Get*` / `GetAs` / `Unmarshal` / `AllSettings` / `Load` 同一路径。写盘不持久化 env/flag。
- **来源分层**：`WithPath`/`WithFiles`/`WithOverlay`(文件) / `WithContent`(内联) / `WithEnv`(env 前缀) / `WithBindPFlags`(命令行)。
- **泛型 getter**：`sysconf.GetAs[T](cfg, key, def...)` 是包级泛型函数（不是方法），类型安全。
- **热重载**：`Watch` 简单回调；`OnChange(prev, next, err)` 带快照。校验失败内存不切换。叠加文件变更也会重载。
- **Flush**：`cfg.Flush()` 立即落盘并返回防抖写入错误。

## 30 秒上手

```go
package main

import (
	"log"
	"github.com/darkit/sysconf"
)

func main() {
	app, cfg, err := sysconf.Load[struct {
		Server struct {
			Host string `config:"host" default:"0.0.0.0"`
			Port int    `config:"port" default:"8080"`
		} `config:"server"`
	}](
		sysconf.WithPath("config.yaml"),
		sysconf.WithEnv("MYAPP"), // 用 MYAPP_ 前缀覆盖（见坑 #1）
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()
	log.Printf("listen %s:%d", app.Server.Host, app.Server.Port)
}
```

完整可运行版本见 `assets/examples/basic.go`。

## 能力路由：先读哪个 reference

| 任务 | 读这个 |
|---|---|
| 全部选项、来源分层、env、pflags、getter/setter、Unmarshal/Marshal | `references/config.md` |
| 校验器与 validation 子包规则（含 boolean/range/enum 等） | `references/validation.md` |
| 加密配置文件、Crypto、密钥、热重载与 Watch | `references/encryption-and-watch.md` |

## 五个最容易踩的坑（务必先看）

1. **启用 env 一定要带前缀**：`WithEnv("MYAPP")` 用 `MYAPP_XXX` 覆盖。若不带前缀启用 env，`Get("path")`/`Get("home")`/`Get("user")` 会去查系统的 `PATH`/`HOME`/`USER`，配置被系统变量无声覆盖。**始终用前缀。** 显式 `Set` 现在优先于 env。

2. **`GetAs[T]` 是包级泛型函数**：`sysconf.GetAs[T](cfg, key, def...)`，不是 `cfg.GetAs`。支持 string/int/int8..int64/uint..uint64/float/bool/Duration/Time；类型不匹配返回默认值（`MustGetAs` 则 panic）。

3. **返回的 map/slice 是拷贝**：`Get`/`GetAs`/`GetWithError` 返回深拷贝，改动不影响内部存储（这是并发安全的保证，别指望改返回值能改配置——用 `Set`）。

4. **加密恢复会覆盖配置**：接了 `ErrorRecovery` 时，解密/格式失败可能触发用默认内容重建配置文件。密钥错误或密文被篡改都会走此路径——生产环境评估是否要启用自动恢复，认证失败应视为需人工介入。

5. **struct tag 用 `config`/`sysconf`/`json` 等**：`Unmarshal`/`Marshal` 识别多种 tag；`default` tag 支持默认值填充（含 uint 字段，如 `Port uint16 \`default:"8080"\``）。

## 核对真实行为

```bash
grep -rn 'func New(opts\|func (c \*Config)' *.go   # 构造与方法
grep -rn 'func With' options.go                    # 所有选项
```

源码即真理源，示例/文档漂移时以 live code 为准。
