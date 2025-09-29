# sysconf 配置加密功能

本文档展示了 sysconf 配置管理库的加密功能，包括内置的 ChaCha20-Poly1305 加密器和自定义加密器接口。

## 🔒 设计理念

### 为什么选择 ChaCha20-Poly1305？

我们经过性能测试和安全性分析，最终选择 **ChaCha20-Poly1305** 作为唯一的内置加密算法：

1. **高性能**: 在各种硬件平台上都有优秀的性能表现
2. **现代密码学**: 被广泛认可的现代 AEAD (认证加密) 算法
3. **安全性**: 同时提供机密性和完整性保护
4. **抗侧信道攻击**: 相比 AES 在软件实现中更安全
5. **移动友好**: 在 ARM 处理器上性能特别出色
6. **频繁写入友好**: 相比 AES-256-GCM 快约 2-3 倍，适合频繁的配置更新

### 接口驱动的设计

虽然我们只提供一种内置加密算法，但通过 `ConfigCrypto` 接口，用户可以轻松实现自定义加密器：

```go
type ConfigCrypto interface {
    Encrypt(data []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    IsEncrypted(data []byte) bool
}
```

## 🚀 基本使用

### 启用默认加密

```go
config, err := sysconf.New(
    sysconf.WithEncryption("my_encryption_key"),
    sysconf.WithPath("./configs"),
    sysconf.WithName("app"),
)
```

### 自动生成密钥

```go
config, err := sysconf.New(
    sysconf.WithEncryption(""), // 空字符串自动生成密钥
    sysconf.WithPath("./configs"),
    sysconf.WithName("app"),
)

// 获取生成的密钥用于备份
key := config.GetEncryptionKey()
```

### 向后兼容的函数

所有这些函数都使用相同的 ChaCha20-Poly1305 算法：

```go
sysconf.WithEncryption("key")           // 默认加密
sysconf.WithFastEncryption("key")       // 快速加密
sysconf.WithLightweightEncryption("key") // 轻量级加密
sysconf.WithSecureEncryption("key")     // 安全加密
sysconf.WithChaCha20Encryption("key")   // ChaCha20 加密
```

## 🛠️ 自定义加密器

如果您需要更高的性能或特殊的加密需求，可以实现自定义加密器：

```go
type CustomCrypto struct {
    key []byte
}

func (c *CustomCrypto) Encrypt(data []byte) ([]byte, error) {
    // 实现您的加密逻辑
    return encryptedData, nil
}

func (c *CustomCrypto) Decrypt(data []byte) ([]byte, error) {
    // 实现您的解密逻辑
    return decryptedData, nil
}

func (c *CustomCrypto) IsEncrypted(data []byte) bool {
    // 检查数据是否已加密
    return true
}

// 使用自定义加密器
customCrypto := &CustomCrypto{key: myKey}
config, err := sysconf.New(
    sysconf.WithEncryptionCrypto(customCrypto),
    // 其他选项...
)
```

## 📊 性能对比

基于实际测试（1000次加密/解密操作）：

| 加密算法 | 耗时 | 相对性能 | 安全性 | 适用场景 |
|---------|------|----------|--------|----------|
| ChaCha20-Poly1305 | ~1.9ms | 1.0x | 高 | **推荐默认选择** |
| 自定义 XOR | ~0.7ms | 2.8x | 低 | 极高频更新场景 |

💡 **建议**: 对于大多数应用场景，默认的 ChaCha20-Poly1305 已经足够快且安全。只有在极高频的配置更新场景下，才考虑使用自定义的轻量级加密器。

## 🔑 密钥管理

### 从 base64 密钥恢复

```go
// 从已保存的密钥恢复加密器
crypto, err := sysconf.NewDefaultCryptoFromKey("SfHvn3Es9KwVMwYjxILh5gjavijaQlOVF2Pwc7oyiqM=")
config, err := sysconf.New(
    sysconf.WithEncryptionCrypto(crypto),
)
```

### 获取加密信息

```go
// 获取加密类型
cryptoType := config.GetCryptoType() // "ChaCha20-Poly1305" 或 "custom"

// 获取加密密钥（如果支持）
key := config.GetEncryptionKey()

// 检查是否启用加密
enabled := config.IsEncryptionEnabled()
```

## 🛡️ 安全特性

### ChaCha20-Poly1305 的安全保证

1. **认证加密 (AEAD)**: 同时提供机密性和完整性
2. **随机 Nonce**: 每次加密使用不同的随机数，防止重放攻击
3. **防篡改**: 任何对密文的修改都会被检测到
4. **前缀识别**: 使用 `SYSCONF_CRYPTO:` 前缀，支持自动检测

### 数据格式

加密后的配置文件格式：
```
SYSCONF_CRYPTO:<base64_encoded_nonce_and_ciphertext>
```

## 📁 文件兼容性

- ✅ **向前兼容**: 可以读取未加密的配置文件
- ✅ **向后兼容**: 所有加密选项函数都指向同一实现
- ✅ **自动检测**: 自动识别文件是否已加密
- ✅ **平滑迁移**: 可以将现有配置文件加密化

## 🚀 运行演示

```bash
cd examples/encryption_demo
go run main.go
```

演示程序包含：
1. 基本加密功能演示
2. 自定义加密器示例
3. 性能对比测试
4. 向后兼容性验证

## 💡 最佳实践

1. **生产环境**: 使用固定的强密钥，不要依赖自动生成
2. **密钥存储**: 将加密密钥存储在环境变量或密钥管理系统中
3. **性能考虑**: 大多数场景下默认加密已足够，无需自定义
4. **备份策略**: 确保密钥的安全备份，丢失密钥将无法恢复数据
5. **迁移策略**: 可以逐步将现有配置文件加密化，无需停机