package sysconf

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// ConfigCrypto 配置加密接口
// 用户可以实现此接口来提供自定义的加密算法
type ConfigCrypto interface {
	// Encrypt 加密配置数据
	// data: 要加密的配置数据（通常是YAML/JSON等格式的字节数组）
	// 返回: 加密后的数据和错误
	Encrypt(data []byte) ([]byte, error)

	// Decrypt 解密配置数据
	// data: 要解密的数据
	// 返回: 解密后的原始配置数据和错误
	Decrypt(data []byte) ([]byte, error)

	// IsEncrypted 检查数据是否已加密
	// data: 要检查的数据
	// 返回: 如果数据已加密返回true，否则返回false
	IsEncrypted(data []byte) bool
}

// CryptoOptions 加密配置选项
type CryptoOptions struct {
	Enabled bool         // 是否启用加密
	Crypto  ConfigCrypto // 加密实现，如果为nil则使用默认ChaCha20加密
	Key     string       // 加密密钥，如果为空则生成随机密钥
}

// DefaultCrypto 默认加密实现 - 使用 ChaCha20-Poly1305
//
// ChaCha20-Poly1305 的优势：
// - 高性能：在各种硬件平台上都有优秀的性能表现
// - 现代密码学：被广泛认可的现代AEAD算法
// - 安全性：提供认证加密，同时保证机密性和完整性
// - 抗侧信道攻击：相比AES在软件实现中更安全
// - 移动友好：在ARM处理器上性能特别出色
type DefaultCrypto struct {
	key    []byte // 256位密钥
	prefix string // 加密数据前缀标识
}

// NewDefaultCrypto 创建新的默认加密器
// key: 加密密钥，如果为空则生成随机密钥
func NewDefaultCrypto(key string) (*DefaultCrypto, error) {
	var keyBytes []byte

	if key == "" {
		// 生成随机32字节密钥
		keyBytes = make([]byte, 32)
		if _, err := rand.Read(keyBytes); err != nil {
			return nil, fmt.Errorf("生成随机密钥失败: %w", err)
		}
	} else {
		// 使用SHA256哈希用户提供的密钥以确保长度为32字节
		hash := sha256.Sum256([]byte(key))
		keyBytes = hash[:]
	}

	return &DefaultCrypto{
		key:    keyBytes,
		prefix: "SYSCONF_CRYPTO:",
	}, nil
}

// Encrypt 实现ConfigCrypto接口的加密方法
func (d *DefaultCrypto) Encrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("加密数据不能为空")
	}

	// 创建ChaCha20-Poly1305 AEAD
	aead, err := chacha20poly1305.New(d.key)
	if err != nil {
		return nil, fmt.Errorf("创建ChaCha20-Poly1305失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密数据
	ciphertext := aead.Seal(nonce, nonce, data, nil)

	// 添加前缀并编码为base64
	result := append([]byte(d.prefix), ciphertext...)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(result)))
	base64.StdEncoding.Encode(encoded, result)

	return encoded, nil
}

// Decrypt 实现ConfigCrypto接口的解密方法
func (d *DefaultCrypto) Decrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("解密数据不能为空")
	}

	// 检查是否为加密数据
	if !d.IsEncrypted(data) {
		return nil, errors.New("数据不是有效的加密格式")
	}

	// base64解码
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		return nil, fmt.Errorf("base64解码失败: %w", err)
	}
	decoded = decoded[:n]

	// 移除前缀
	prefixLen := len(d.prefix)
	if len(decoded) < prefixLen {
		return nil, errors.New("加密数据格式无效")
	}
	ciphertext := decoded[prefixLen:]

	// 创建ChaCha20-Poly1305 AEAD
	aead, err := chacha20poly1305.New(d.key)
	if err != nil {
		return nil, fmt.Errorf("创建ChaCha20-Poly1305失败: %w", err)
	}

	// 检查数据长度
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("加密数据太短")
	}

	// 提取nonce和密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密数据
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}

	return plaintext, nil
}

// IsEncrypted 检查数据是否已加密
func (d *DefaultCrypto) IsEncrypted(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// 尝试base64解码
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		return false
	}
	decoded = decoded[:n]

	// 检查前缀
	prefixBytes := []byte(d.prefix)
	if len(decoded) < len(prefixBytes) {
		return false
	}

	for i, b := range prefixBytes {
		if decoded[i] != b {
			return false
		}
	}

	return true
}

// GetKey 获取加密密钥的base64编码（用于保存和恢复）
func (d *DefaultCrypto) GetKey() string {
	return base64.StdEncoding.EncodeToString(d.key)
}

// GetKeyBytes 获取原始密钥字节（用于高级用途）
func (d *DefaultCrypto) GetKeyBytes() []byte {
	// 返回副本以避免外部修改
	key := make([]byte, len(d.key))
	copy(key, d.key)
	return key
}

// NewDefaultCryptoFromKey 从base64编码的密钥创建默认加密器
func NewDefaultCryptoFromKey(encodedKey string) (*DefaultCrypto, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("解码密钥失败: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, errors.New("密钥长度必须为32字节")
	}

	return &DefaultCrypto{
		key:    keyBytes,
		prefix: "SYSCONF_CRYPTO:",
	}, nil
}

// =============================================================================
// 便利函数和向后兼容性
// =============================================================================

// NewCrypto 创建默认加密器的便利函数（向后兼容）
func NewCrypto(key string) (ConfigCrypto, error) {
	return NewDefaultCrypto(key)
}

// 以下是为了向后兼容而保留的别名函数

// NewChaCha20Crypto 别名函数，指向默认加密器
func NewChaCha20Crypto(key string) (*DefaultCrypto, error) {
	return NewDefaultCrypto(key)
}

// NewChaCha20CryptoFromKey 别名函数，指向默认加密器
func NewChaCha20CryptoFromKey(encodedKey string) (*DefaultCrypto, error) {
	return NewDefaultCryptoFromKey(encodedKey)
}
