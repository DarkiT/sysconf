package utils

import (
	"sync"
)

// CacheManager 缓存管理器
type CacheManager struct {
	camelToSnakeCache map[string]string
	snakeToCamelCache map[string]string
	maxCacheSize      int
	mu                sync.RWMutex
}

// NewCacheManager 创建新的缓存管理器
func NewCacheManager(maxSize int) *CacheManager {
	return &CacheManager{
		camelToSnakeCache: make(map[string]string),
		snakeToCamelCache: make(map[string]string),
		maxCacheSize:      maxSize,
	}
}

// CleanCache 清理缓存（当缓存过大时）
func (cm *CacheManager) CleanCache() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.camelToSnakeCache) > cm.maxCacheSize {
		// 清理一半的缓存条目
		newCache := make(map[string]string)
		count := 0
		for k, v := range cm.camelToSnakeCache {
			if count < cm.maxCacheSize/2 {
				newCache[k] = v
				count++
			} else {
				break
			}
		}
		cm.camelToSnakeCache = newCache
	}

	if len(cm.snakeToCamelCache) > cm.maxCacheSize {
		// 清理一半的缓存条目
		newCache := make(map[string]string)
		count := 0
		for k, v := range cm.snakeToCamelCache {
			if count < cm.maxCacheSize/2 {
				newCache[k] = v
				count++
			} else {
				break
			}
		}
		cm.snakeToCamelCache = newCache
	}
}

// SetCamelToSnake 设置camelCase到snake_case的缓存
func (cm *CacheManager) SetCamelToSnake(key, value string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.camelToSnakeCache) >= cm.maxCacheSize {
		cm.CleanCache()
	}
	cm.camelToSnakeCache[key] = value
}

// SetSnakeToCamel 设置snake_case到camelCase的缓存
func (cm *CacheManager) SetSnakeToCamel(key, value string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.snakeToCamelCache) >= cm.maxCacheSize {
		cm.CleanCache()
	}
	cm.snakeToCamelCache[key] = value
}

// GetCamelToSnake 获取camelCase到snake_case的缓存
func (cm *CacheManager) GetCamelToSnake(key string) (string, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	value, exists := cm.camelToSnakeCache[key]
	return value, exists
}

// GetSnakeToCamel 获取snake_case到camelCase的缓存
func (cm *CacheManager) GetSnakeToCamel(key string) (string, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	value, exists := cm.snakeToCamelCache[key]
	return value, exists
}
