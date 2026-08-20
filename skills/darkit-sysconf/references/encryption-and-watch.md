# 加密配置与热重载

## 加密配置文件

```go
cfg, _ := sysconf.New(
	sysconf.WithPath("config.enc"),
	sysconf.WithEncryption("your-strong-key"),   // 启用 AEAD 加密
)
// 或注入自定义 crypto
crypto, _ := sysconf.NewChaCha20Crypto("your-key")
cfg, _ := sysconf.New(sysconf.WithEncryptionCrypto(crypto))
```

Crypto 构造：

```go
sysconf.NewDefaultCrypto(key)             // (*DefaultCrypto, error)，AES-GCM
sysconf.NewChaCha20Crypto(key)            // ChaCha20-Poly1305
sysconf.NewDefaultCryptoFromKey(encoded)  // 从已编码密钥
```

实现要点（了解即可）：AEAD（AES-GCM / ChaCha20-Poly1305）+ **每次加密随机 salt/nonce** + `crypto/rand` + Argon2id KDF，无 ECB/nonce 复用/弱随机。

查询状态：

```go
cfg.IsEncryptionEnabled() bool
cfg.GetCryptoType() string
cfg.GetEncryptionKey() string   // 谨慎：勿打日志
```

## ⚠️ 加密恢复语义

若接了 `sysconf.NewErrorRecovery(cfg)`，解密/格式失败可能触发用**默认内容重建**配置文件（覆盖原文件，覆盖前会尝试备份）。但认证失败（密钥错误或**密文被篡改**）与"文件不存在"走同一恢复分支——这意味着一次篡改可能把线上配置打回默认值。生产建议：

- 认证失败视为需人工介入，不自动重建；
- 或用独立开关控制自动恢复；
- 确保备份成功才允许覆盖。

## 热重载 Watch

```go
cfg.Watch(func() {
	// 配置文件变更后回调；用 cfg.Get* 读新值
})

// 带 prev/next 快照；校验失败时 err != nil，内存保持旧配置
_ = cfg.OnChange(func(prev, next map[string]any, err error) {
	if err != nil {
		return
	}
	_ = next
})

// 带 context 的可取消版本
cancel := cfg.WatchWithContext(ctx, callback)
defer cancel()
```

配合 `WithWatchDebounce(delay)` 合并抖动的连续变更。存在验证器时，非法重载会拒绝切换并回传 `ErrValidation`。

> 并发提示：`Watch` 启用后，在高并发读写下对底层 viper 的访问存在已知的并发敏感面。稳妥用法是：Watch 回调里只读快照（`cfg.Get*`/`AllSettings`），避免在回调内与其它 goroutine 并发 `Set`；或用 `WatchWithContext` 明确生命周期，`Close` 时 cancel。
