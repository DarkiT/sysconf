package sysconf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// readConfigFile 读取配置文件（支持解密）- 线程安全版本
func (c *Config) readConfigFile() error {
	if c.name == "" {
		return nil // 内存模式，不需要读取文件
	}

	configFile := filepath.Join(c.path, c.name+"."+c.mode)

	// 检查文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return err // 直接返回原始错误，让调用者处理
	}

	// 读取文件内容（锁外执行 I/O）
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	// 如果启用了加密，尝试解密（锁外执行解密）
	if c.cryptoOptions.Enabled && c.crypto != nil {
		if c.crypto.IsEncrypted(data) {
			c.logger.Debugf("Decrypting config file")
			decryptedData, err := c.crypto.Decrypt(data)
			if err != nil {
				return fmt.Errorf("decrypt config file: %w", err)
			}
			data = decryptedData
			c.logger.Infof("Config file decrypted successfully")
		} else {
			c.logger.Debugf("Config file is not encrypted")
		}
	}

	// 使用 viper 解析配置内容（需要锁保护，锁顺序：cacheBuildMu -> writeMu）
	c.cacheBuildMu.Lock()
	c.writeMu.Lock()
	err = c.viper.ReadConfig(strings.NewReader(string(data)))
	c.writeMu.Unlock()
	c.cacheBuildMu.Unlock()

	if err != nil {
		return fmt.Errorf("parse config content: %w", err)
	}

	return nil
}

// readConfigFileUnsafe 读取配置文件 - 调用者已持锁版本（供 initialize 等内部方法使用）
func (c *Config) readConfigFileUnsafe() error {
	if c.name == "" {
		return nil
	}

	configFile := filepath.Join(c.path, c.name+"."+c.mode)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if c.cryptoOptions.Enabled && c.crypto != nil {
		if c.crypto.IsEncrypted(data) {
			c.logger.Debugf("Decrypting config file")
			decryptedData, err := c.crypto.Decrypt(data)
			if err != nil {
				return fmt.Errorf("decrypt config file: %w", err)
			}
			data = decryptedData
			c.logger.Infof("Config file decrypted successfully")
		} else {
			c.logger.Debugf("Config file is not encrypted")
		}
	}

	if err := c.viper.ReadConfig(strings.NewReader(string(data))); err != nil {
		return fmt.Errorf("parse config content: %w", err)
	}

	return nil
}

// writeConfigFile 写入配置文件（支持加密）
func (c *Config) writeConfigFile() error {
	if c.name == "" {
		return nil // 内存模式，不需要写入文件
	}

	configFile := filepath.Join(c.path, c.name+"."+c.mode)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// 将 viper 配置序列化为字节数组
	data, err := c.marshalConfig()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// 如果启用了加密，加密数据
	if c.cryptoOptions.Enabled && c.crypto != nil {
		c.logger.Debugf("Encrypting config file")
		encryptedData, err := c.crypto.Encrypt(data)
		if err != nil {
			return fmt.Errorf("encrypt config: %w", err)
		}
		data = encryptedData
		c.logger.Infof("Config file encrypted successfully")
	}

	// 写入文件
	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	c.logger.Infof("Config file written: %s", configFile)
	return nil
}

// marshalConfig 将viper配置序列化为指定格式的字节数组
func (c *Config) marshalConfig() ([]byte, error) {
	allSettings := c.snapshotAllSettings()

	switch c.mode {
	case "yaml", "yml":
		return yaml.Marshal(allSettings)
	case "json":
		return json.MarshalIndent(allSettings, "", "  ")
	case "ini":
		// 对于INI格式，我们需要特殊处理
		return c.marshalToINI(allSettings)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", c.mode)
	}
}

// writeConfigFileWithData 使用传入的配置数据写入文件（支持加密）
// 调用者需确保 settingsData 已在锁外安全获取，避免自死锁
func (c *Config) writeConfigFileWithData(settingsData map[string]any) error {
	if settingsData == nil {
		return fmt.Errorf("settingsData cannot be nil")
	}

	if c.name == "" {
		return nil // 内存模式，不需要写入文件
	}

	configFile := filepath.Join(c.path, c.name+"."+c.mode)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// 使用传入的数据进行序列化，避免再次调用 snapshotAllSettings()
	data, err := c.marshalConfigWithData(settingsData)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// 如果启用了加密，加密数据
	if c.cryptoOptions.Enabled && c.crypto != nil {
		c.logger.Debugf("Encrypting config file")
		encryptedData, err := c.crypto.Encrypt(data)
		if err != nil {
			return fmt.Errorf("encrypt config: %w", err)
		}
		data = encryptedData
		c.logger.Infof("Config file encrypted successfully")
	}

	// 写入文件
	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	c.logger.Infof("Config file written: %s", configFile)
	return nil
}

// marshalConfigWithData 使用传入的配置数据序列化为指定格式的字节数组
// 不调用 snapshotAllSettings()，由调用者提供数据以避免锁竞争
func (c *Config) marshalConfigWithData(settings map[string]any) ([]byte, error) {
	switch c.mode {
	case "yaml", "yml":
		return yaml.Marshal(settings)
	case "json":
		return json.MarshalIndent(settings, "", "  ")
	case "ini":
		// 对于INI格式，我们需要特殊处理
		return c.marshalToINI(settings)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", c.mode)
	}
}

// marshalToINI 将配置转换为INI格式
func (c *Config) marshalToINI(settings map[string]any) ([]byte, error) {
	var buf bytes.Buffer

	// 写入顶级键值对
	for key, value := range settings {
		switch v := value.(type) {
		case map[string]any:
			// 写入section
			fmt.Fprintf(&buf, "\n[%s]\n", key)
			for k, val := range v {
				fmt.Fprintf(&buf, "%s = %v\n", k, val)
			}
		default:
			// 写入顶级键值对
			fmt.Fprintf(&buf, "%s = %v\n", key, v)
		}
	}

	return buf.Bytes(), nil
}

// GetEncryptionKey 获取当前使用的加密密钥（如果适用）
func (c *Config) GetEncryptionKey() string {
	if !c.cryptoOptions.Enabled || c.crypto == nil {
		return ""
	}

	// 支持默认加密器和自定义加密器
	switch crypto := c.crypto.(type) {
	case *DefaultCrypto:
		return crypto.GetKey()
	default:
		// 对于自定义加密器，尝试通过接口方法获取密钥
		// 如果自定义加密器没有提供GetKey方法，返回空字符串
		if keyGetter, ok := crypto.(interface{ GetKey() string }); ok {
			return keyGetter.GetKey()
		}
		return ""
	}
}

// GetCryptoType 获取当前使用的加密类型
func (c *Config) GetCryptoType() string {
	if !c.cryptoOptions.Enabled || c.crypto == nil {
		return "none"
	}

	switch c.crypto.(type) {
	case *DefaultCrypto:
		return "ChaCha20-Poly1305"
	default:
		return "custom"
	}
}

// IsEncryptionEnabled 检查是否启用了加密
func (c *Config) IsEncryptionEnabled() bool {
	return c.cryptoOptions.Enabled && c.crypto != nil
}
