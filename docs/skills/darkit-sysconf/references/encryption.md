---
name: sysconf-encrypting-config
description: 为 sysconf 配置启用 ChaCha20-Poly1305 加密功能或使用自定义加密器。当用户需要保护敏感配置数据、加密配置文件、实现配置安全存储或使用自定义加密算法时使用。
version: "1.0.0"
---

# Sysconf 配置加密功能

此技能帮助你在 sysconf 项目中启用和配置加密功能，保护敏感配置数据。

## 概述

Sysconf 提供企业级配置加密功能：
- **ChaCha20-Poly1305 认证加密**: 高性能、抗侧信道攻击
- **AEAD 认证**: 提供机密性和完整性保证
- **自定义加密器**: 支持自定义加密算法
- **自动加密/解密**: 读写时透明处理
- **加密状态检测**: 自动识别加密数据

## 何时使用此技能

使用此技能当你需要：
- 保护敏感配置信息（密码、API密钥、证书等）
- 满足合规要求（数据保护法规）
- 加密配置文件存储
- 实现配置安全传输
- 使用自定义加密算法
- 集成企业密钥管理系统

## 前置要求

- 已完成 sysconf 基础集成（参考 `integrating-sysconf` 技能）
- 了解基本的加密概念
- 准备加密密钥（生产环境从密钥管理系统获取）

## 使用指南

### 步骤 1: 启用基础加密（推荐快速开始）

使用内置 ChaCha20-Poly1305 加密：

```go
import "github.com/darkit/sysconf"

// 方式 1: 直接提供加密密钥
cfg, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("app"),
    sysconf.WithEncryption("your-secret-password-min-32-chars"),
)
if err != nil {
    log.Fatal("配置初始化失败:", err)
}

// 敏感配置会自动加密存储
cfg.Set("database.password", "super-secret-password")
cfg.Set("api.secret_key", "sk-1234567890abcdef")

// 读取时自动解密
dbPassword := cfg.GetString("database.password")
apiKey := cfg.GetString("api.secret_key")
```

**重要**: 加密密钥应该：
- 长度至少 32 个字符
- 从安全的密钥管理系统获取（生产环境）
- 不要硬编码在代码中
- 通过环境变量或密钥管理服务传递

### 步骤 2: 环境化密钥管理（生产环境推荐）

```go
func createSecureConfig(env string) (*sysconf.Config, error) {
    var encryptionKey string

    switch env {
    case "production":
        // 从密钥管理系统获取（如 AWS KMS、Azure Key Vault、HashiCorp Vault）
        encryptionKey = getFromKeyVault("CONFIG_ENCRYPTION_KEY")
        if encryptionKey == "" {
            return nil, fmt.Errorf("生产环境必须配置加密密钥")
        }

    case "staging":
        // 从环境变量获取
        encryptionKey = os.Getenv("STAGING_CONFIG_PASSWORD")
        if encryptionKey == "" {
            log.Println("警告: 未设置加密密钥，使用默认密钥（仅用于测试）")
            encryptionKey = "staging-default-key-DO-NOT-USE-IN-PROD"
        }

    case "development":
        // 开发环境可选择不加密
        log.Println("开发环境: 配置加密已禁用")
        return sysconf.New(
            sysconf.WithPath("configs"),
            sysconf.WithName(fmt.Sprintf("app-%s", env)),
        )
    }

    // 创建加密配置
    return sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName(fmt.Sprintf("app-%s", env)),
        sysconf.WithEncryption(encryptionKey),
    )
}
```

### 步骤 3: 使用自定义加密器

实现 `ConfigCrypto` 接口使用自定义加密算法：

```go
// 自定义加密器接口
type ConfigCrypto interface {
    Encrypt(data []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    IsEncrypted(data []byte) bool
}

// 示例: AES-GCM 自定义加密器
type AESGCMCrypto struct {
    key []byte
}

func (c *AESGCMCrypto) Encrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(c.key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    return gcm.Seal(nonce, nonce, data, nil), nil
}

func (c *AESGCMCrypto) Decrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(c.key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, errors.New("数据太短")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    return gcm.Open(nil, nonce, ciphertext, nil)
}

func (c *AESGCMCrypto) IsEncrypted(data []byte) bool {
    // 简单检查：加密数据长度应该大于 nonce 大小
    return len(data) > 12
}

// 使用自定义加密器
customCrypto := &AESGCMCrypto{key: []byte("your-32-byte-aes-gcm-key-here!!")}
cfg, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("app"),
    sysconf.WithEncryptionCrypto(customCrypto),
)
```

### 步骤 4: 加密配置文件持久化

```go
// 创建加密配置
cfg, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("secure-config"),
    sysconf.WithEncryption(encryptionKey),
    sysconf.WithContent(defaultConfig),
)
if err != nil {
    log.Fatal(err)
}

// 设置敏感数据
cfg.Set("database.password", "production-db-password")
cfg.Set("api.private_key", "-----BEGIN PRIVATE KEY-----...")
cfg.Set("encryption.master_key", "master-encryption-key")

// 配置会自动加密并写入文件
// 文件内容是加密的，无法直接读取

// 下次启动时，使用相同密钥可以解密读取
cfg2, err := sysconf.New(
    sysconf.WithPath("configs"),
    sysconf.WithName("secure-config"),
    sysconf.WithEncryption(encryptionKey),  // 必须使用相同密钥
)
password := cfg2.GetString("database.password")  // 自动解密
```

### 步骤 5: 选择性加密字段

并非所有配置都需要加密。建议策略：

```go
// ✅ 需要加密的配置
cfg.Set("database.password", secretPassword)       // 数据库密码
cfg.Set("api.secret_key", apiSecret)               // API 密钥
cfg.Set("jwt.signing_key", jwtKey)                 // JWT 签名密钥
cfg.Set("encryption.master_key", masterKey)        // 加密主密钥
cfg.Set("oauth.client_secret", oauthSecret)        // OAuth 密钥
cfg.Set("smtp.password", smtpPassword)             // SMTP 密码

// ❌ 不需要加密的配置
cfg.Set("app.name", "MyApp")                       // 应用名称
cfg.Set("server.host", "localhost")                // 服务器地址
cfg.Set("server.port", 8080)                       // 端口号
cfg.Set("logging.level", "info")                   // 日志级别
```

**注意**: Sysconf 的加密是文件级的，整个配置文件会被加密。建议将敏感配置和普通配置分开存储。

### 步骤 6: 密钥轮换

实现密钥轮换策略：

```go
func rotateEncryptionKey(oldKey, newKey string) error {
    // 1. 使用旧密钥读取配置
    oldCfg, err := sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName("app"),
        sysconf.WithEncryption(oldKey),
    )
    if err != nil {
        return fmt.Errorf("无法使用旧密钥读取配置: %w", err)
    }

    // 2. 读取所有配置
    allSettings := oldCfg.AllSettings()

    // 3. 使用新密钥创建配置
    newCfg, err := sysconf.New(
        sysconf.WithPath("configs"),
        sysconf.WithName("app-new"),
        sysconf.WithEncryption(newKey),
    )
    if err != nil {
        return fmt.Errorf("无法创建新配置: %w", err)
    }

    // 4. 写入所有配置（使用新密钥加密）
    for key, value := range flattenMap(allSettings) {
        if err := newCfg.Set(key, value); err != nil {
            return fmt.Errorf("密钥轮换失败: %w", err)
        }
    }

    log.Println("密钥轮换成功")
    return nil
}
```

## 模板和示例

### 模板文件

- `templates/encrypted-config.go.tmpl` - 加密配置集成模板

### 示例代码

- `examples/basic-encryption.go` - 基础加密使用示例
- `examples/custom-crypto.go` - 自定义加密器示例

## 加密算法比较

| 算法 | 性能 | 安全性 | 适用场景 |
|------|------|--------|----------|
| ChaCha20-Poly1305 (默认) | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 通用场景，移动设备，无硬件加速 |
| AES-GCM | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 有硬件加速（AES-NI）的服务器 |
| AES-CBC | ⭐⭐⭐ | ⭐⭐⭐⭐ | 传统场景，需要兼容性 |

**默认选择 ChaCha20-Poly1305 的原因**：
- ✅ 软件实现性能优于 AES（无硬件加速时）
- ✅ 抗侧信道攻击
- ✅ 移动设备友好
- ✅ AEAD 提供认证加密
- ✅ 现代加密算法，安全性高

## 最佳实践

### 1. 密钥管理

```go
// ✅ 推荐：从密钥管理服务获取
encryptionKey := getFromKeyVault("CONFIG_ENCRYPTION_KEY")

// ✅ 推荐：从环境变量获取
encryptionKey := os.Getenv("CONFIG_ENCRYPTION_KEY")

// ❌ 避免：硬编码在代码中
encryptionKey := "hardcoded-key-is-insecure"  // 绝对不要这样做！
```

### 2. 密钥长度

```go
// ✅ 推荐：32 字节或更长
encryptionKey := generateRandomKey(32)  // ChaCha20-Poly1305 推荐

// ❌ 避免：过短的密钥
encryptionKey := "short"  // 不安全！
```

### 3. 分环境配置

```go
// ✅ 推荐：不同环境使用不同密钥
prodKey := getKeyForEnvironment("production")
stagingKey := getKeyForEnvironment("staging")
devKey := ""  // 开发环境可以不加密

// ❌ 避免：所有环境使用相同密钥
```

### 4. 配置分离

```go
// ✅ 推荐：敏感配置单独存储
sensitiveCfg, _ := sysconf.New(
    sysconf.WithName("secrets"),
    sysconf.WithEncryption(key),
)

normalCfg, _ := sysconf.New(
    sysconf.WithName("app"),
    // 无需加密
)

// ❌ 避免：所有配置都加密（性能损失）
```

### 5. 错误处理

```go
cfg, err := sysconf.New(
    sysconf.WithEncryption(key),
)
if err != nil {
    // ✅ 推荐：明确的错误处理
    if errors.Is(err, sysconf.ErrInvalidKey) {
        log.Fatal("加密密钥无效或过短")
    } else if errors.Is(err, sysconf.ErrDecryptionFailed) {
        log.Fatal("解密失败，可能密钥不正确")
    }
}
```

## 故障排查

**问题: 解密失败**
- 确认使用的密钥与加密时相同
- 检查密钥长度是否满足要求（ChaCha20-Poly1305 需要 32 字节）
- 确认配置文件未被篡改

**问题: 性能问题**
- 考虑只加密敏感配置，而不是所有配置
- 使用硬件加速（AES-NI）时选择 AES-GCM
- 减少配置文件大小

**问题: 密钥管理复杂**
- 使用企业密钥管理系统（AWS KMS、Azure Key Vault、HashiCorp Vault）
- 实现自动化密钥轮换
- 使用环境变量传递密钥

## 安全建议

1. **永远不要在代码中硬编码密钥**
2. **使用密钥管理服务**（生产环境必须）
3. **定期轮换密钥**（建议每 90 天）
4. **限制密钥访问权限**（最小权限原则）
5. **审计密钥使用**（记录谁在何时访问密钥）
6. **备份加密配置**（确保有密钥的备份）
7. **测试密钥恢复流程**（确保能够从备份恢复）

## 性能提示

- 加密/解密操作的性能开销约 5-10%
- ChaCha20-Poly1305 在无硬件加速时比 AES 快 2-3 倍
- 配置文件大小对性能影响不大（1KB 和 100KB 差异很小）
- 建议将敏感配置单独存储，减少加密范围

## 相关资源

- ChaCha20-Poly1305 RFC: https://tools.ietf.org/html/rfc7539
- 加密实现源码: /workspace/crypto.go
- 主集成技能: `integrating-sysconf`
- 验证技能: `configuring-validation`
