package sysconf

import (
	"errors"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
)

// Set 设置配置值
func (c *Config) Set(key string, value any) error {
	if key == "" {
		return ErrInvalidKey
	}

	// 所有对 viper 的操作都需要锁保护
	c.viperMu.Lock()
	c.viper.Set(key, value)
	c.viperMu.Unlock()

	// 如果配置文件名称不存在则不保存文件
	if c.name == "" {
		return nil
	}

	// 用独立的互斥锁处理写入操作
	c.writeMu.Lock()

	// 标记有待写入的更改（使用原子操作替代布尔变量）
	if c.pendingWritesFlag == nil {
		c.pendingWritesFlag = new(int32)
	}
	atomic.StoreInt32(c.pendingWritesFlag, 1)

	// 如果定时器已存在，重置它
	if c.writeTimer != nil {
		c.writeTimer.Stop()
	}

	// 在互斥锁保护下创建定时器的本地副本
	writeTimer := time.AfterFunc(3*time.Second, func() {
		// 使用一个更严格的方法处理 viper 写入
		c.safeWriteConfig()
	})

	// 保存定时器并释放锁
	c.writeTimer = writeTimer
	c.writeMu.Unlock()

	return nil
}

// safeWriteConfig 安全地执行配置写入操作
func (c *Config) safeWriteConfig() {
	// 首先获取写入锁
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// 检查是否还有待写入的变更
	if c.pendingWritesFlag == nil || atomic.LoadInt32(c.pendingWritesFlag) == 0 {
		return
	}

	// 重置标志（使用原子操作）
	atomic.StoreInt32(c.pendingWritesFlag, 0)

	// 通过深度复制 viper 配置来避免数据竞争
	c.viperMu.Lock()
	configCopy := make(map[string]interface{})

	// 复制所有设置
	for k, v := range c.viper.AllSettings() {
		configCopy[k] = v
	}

	// 获取必要的信息然后释放主锁
	configFile := filepath.Join(c.path, c.name+"."+c.mode)
	c.viperMu.Unlock()

	// 创建临时 viper 实例进行写入操作
	tmpViper := viper.New()
	for k, v := range configCopy {
		tmpViper.Set(k, v)
	}

	// 设置必要的配置信息
	tmpViper.SetConfigFile(configFile)

	// 执行写入操作
	writeErr := tmpViper.WriteConfig()
	if writeErr != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(writeErr, &configFileNotFoundError) {
			_ = tmpViper.WriteConfigAs(configFile)
		}
	}
}

// SetEnvPrefix 更新环境变量选项
func (c *Config) SetEnvPrefix(prefix string) error {
	c.mu.Lock()
	c.envOptions.Prefix = prefix
	c.envOptions.Enabled = prefix != "" // 如果有前缀就启用环境变量
	c.mu.Unlock()

	// 在独立的互斥锁保护下初始化配置
	c.writeMu.Lock()
	// 取消任何待处理的写入操作
	if c.pendingWritesFlag != nil {
		atomic.StoreInt32(c.pendingWritesFlag, 0)
	}
	if c.writeTimer != nil {
		c.writeTimer.Stop()
	}
	c.writeMu.Unlock()

	err := c.initialize()
	return err
}
