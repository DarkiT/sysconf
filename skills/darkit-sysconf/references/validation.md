# 配置校验

两种接入：整体校验函数，或字段级规则（`validation` 子包）。

## 挂载校验器

```go
cfg, _ := sysconf.New(
	sysconf.WithValidateFunc(func(all map[string]any) error {
		if all["server"] == nil { return errors.New("server 段缺失") }
		return nil
	}),
	sysconf.WithValidator(myValidator),       // 实现 ConfigValidator 接口
	sysconf.WithValidators(v1, v2),
)

// 运行时增删
cfg.AddValidator(v)
cfg.AddValidateFunc(fn)
cfg.ClearValidators()
```

校验在加载/Set 时触发；失败返回 error。

## validation 子包规则

`github.com/darkit/sysconf/validation` 提供字段级规则。内置规则名（`ValidateValue(value, "rule:params")`）：

```
required, string, number, boolean(bool), email, url, range, length, regex,
enum, ipv4, ipv6, port, hostname, alphanum, uuid, json, base64,
datetime, timezone, creditcard, phonenumber
```

- `range:1,100` / `length:3,20` / `enum:a,b,c` / `regex:^\d+$`
- `boolean`/`bool` 兼容 `true/false/1/0/yes/no/on/off`（含字符串形态，适配 env 覆盖）。
- 预定义组合：`NewWebServerValidator()`、`NewAPIValidator()`（`validation/predefined.go`）。

## 结构体校验

```go
validation.ValidateStruct(myStruct, rules)   // rules: map[fieldName][]ValidationRule
```

未导出字段自动跳过；非结构体输入安全返回。

## 注意

- 校验短路：首个失败即返回（一次一个错），不聚合。
- 自定义规则：`validation.RegisterValidator("myrule", fn)`（并发安全）。
- validator 实例在**构建完成后不应再并发增删规则**（实例字段无锁）；注册表层是并发安全的。
