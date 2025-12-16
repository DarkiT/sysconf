package sysconf

import (
	"sync/atomic"
	"time"
)

// enableReadCache 启用读取缓存（默认启用）
func (c *Config) enableReadCache() {
	c.cacheEnabled.Store(true)
	delay := c.cacheWarmupDelay
	if delay <= 0 {
		c.scheduleCacheUpdate(0)
		return
	}
	c.scheduleCacheUpdate(delay)
}

// disableReadCache 禁用读取缓存
func (c *Config) disableReadCache() {
	c.cacheEnabled.Store(false)
	emptyCache := make(map[string]any)
	c.readCache.Store(emptyCache)
}

// loadReadCache 加载只读缓存
func (c *Config) loadReadCache() map[string]any {
	if !c.cacheEnabled.Load() {
		return nil
	}

	if cache := c.readCache.Load(); cache != nil {
		return cache.(map[string]any)
	}
	return nil
}

// updateReadCache 更新只读缓存
func (c *Config) updateReadCache() {
	if c == nil {
		return
	}
	if !c.cacheEnabled.Load() {
		return
	}

	// 通过 snapshotAllSettings 获取安全快照（内部已按 cacheBuildMu -> mu -> writeMu 顺序加锁）
	safeSettings := c.snapshotAllSettings()

	// 然后更新缓存
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	// 创建缓存的深拷贝，同时构建嵌套键缓存
	newCache := make(map[string]any)
	flatCache := make(map[string]any)

	for key, value := range safeSettings {
		newCache[key] = value
		// 扁平化嵌套结构，构建完整的键路径缓存
		c.flattenMapToCache(key, value, flatCache)
	}

	// 合并原始缓存和扁平化缓存
	for k, v := range flatCache {
		newCache[k] = v
	}

	// 原子更新缓存
	c.readCache.Store(newCache)
	atomic.AddInt64(&c.cacheVersion, 1)

	c.logger.Debugf("Read cache updated, version: %d, keys: %d, flat keys: %d",
		atomic.LoadInt64(&c.cacheVersion), len(safeSettings), len(flatCache))
}

// flattenMapToCache 递归扁平化map结构，生成完整的键路径
func (c *Config) flattenMapToCache(prefix string, value any, cache map[string]any) {
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			fullKey := prefix + "." + key
			cache[fullKey] = val
			// 递归处理嵌套结构
			c.flattenMapToCache(fullKey, val, cache)
		}
	case map[interface{}]any:
		for key, val := range v {
			if keyStr, ok := key.(string); ok {
				fullKey := prefix + "." + keyStr
				cache[fullKey] = val
				// 递归处理嵌套结构
				c.flattenMapToCache(fullKey, val, cache)
			}
		}
	default:
		// 叶子节点，已经在上层添加到缓存中
	}
}

// getCachedValue 从缓存获取值，如果缓存未命中则从viper获取
func (c *Config) getCachedValue(key string) (any, bool) {
	// 简化：只从缓存读取，避免复杂的锁逻辑
	if cache := c.loadReadCache(); cache != nil {
		// 首先尝试直接匹配
		if value, exists := cache[key]; exists {
			return value, true
		}

		// 然后尝试嵌套键查找
		if value := c.getNestedValue(cache, key); value != nil {
			return value, true
		}
	}

	// 如果缓存未命中，返回false，让调用者使用传统路径
	return nil, false
}

// getNestedValue 从缓存中获取嵌套键的值
// 委托给 getNestedValueFromData 实现，避免重复逻辑
func (c *Config) getNestedValue(cache map[string]any, key string) any {
	// 如果直接存在，返回
	if value, exists := cache[key]; exists {
		return value
	}

	// 委托给统一的嵌套查找实现
	if value, found := c.getNestedValueFromData(cache, key); found {
		return value
	}
	return nil
}

// invalidateCache 使缓存失效（在配置更新时调用）
func (c *Config) invalidateCache() {
	if c.cacheEnabled.Load() {
		// 存储空的map而不是nil，避免atomic.Value的nil限制
		emptyCache := make(map[string]any)
		c.readCache.Store(emptyCache)
		atomic.AddInt64(&c.cacheVersion, 1)

		// 异步重建缓存，但不阻塞当前操作
		delay := c.cacheRebuildDelay
		if delay <= 0 {
			c.scheduleCacheUpdate(0)
			return
		}
		c.scheduleCacheUpdate(delay)
	}
}

// scheduleCacheUpdate 调度缓存更新，支持停止信号
func (c *Config) scheduleCacheUpdate(delay time.Duration) {
	c.wg.Add(1)
	if delay <= 0 {
		go func() {
			defer c.wg.Done()
			select {
			case <-c.stopChan:
				return
			default:
				c.updateReadCache()
			}
		}()
		return
	}

	time.AfterFunc(delay, func() {
		defer c.wg.Done()
		select {
		case <-c.stopChan:
			return
		default:
			c.updateReadCache()
		}
	})
}
