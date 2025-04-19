package sysconf

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Set 设置配置值
func (c *Config) Set(key string, value any) error {
	if key == "" {
		c.logger.Errorf("Attempted to set config with empty key")
		return ErrInvalidKey
	}

	c.mu.Lock()
	c.viper.Set(key, value)
	c.logger.Debugf("Set config value: %s", key)
	c.mu.Unlock()

	// 如果配置文件名称不存在则不保存文件
	if c.name == "" {
		c.logger.Debugf("Config file name not set, skipping write")
		return nil
	}
	// 用独立的互斥锁处理写入操作
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// 标记有待写入的更改
	c.pendingWrites = true

	// 如果定时器已存在，重置它
	if c.writeTimer != nil {
		c.writeTimer.Stop()
		c.logger.Debugf("Reset existing config write timer")
	}

	// 创建新的延迟写入定时器
	c.writeTimer = time.AfterFunc(3*time.Second, func() {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()

		if !c.pendingWrites {
			c.logger.Debugf("No pending changes, skipping write operation")
			return
		}

		c.logger.Infof("Writing config file")
		if err := c.viper.WriteConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if errors.As(err, &configFileNotFoundError) {
				configFile := filepath.Join(c.path, c.name+"."+c.mode)
				c.logger.Infof("Config file does not exist, creating new file: %s", configFile)
				if err := c.viper.WriteConfigAs(configFile); err != nil {
					c.logger.Errorf("Failed to create config file: %v", err)
				} else {
					c.logger.Infof("Config file created successfully")
				}
			} else {
				c.logger.Errorf("Failed to write config file: %v", err)
			}
		} else {
			c.logger.Infof("Config file written successfully")
		}

		c.pendingWrites = false
	})

	return nil
}

// SetEnvPrefix 更新环境变量选项
func (c *Config) SetEnvPrefix(prefix string) error {
	c.envOptions.Prefix = prefix
	c.envOptions.Enabled = prefix != "" // 如果有前缀就启用环境变量

	return c.initialize()
}
