package cache

import (
	"sync"
)

// Manager 缓存管理器
type Manager struct {
	camelToSnakeCache map[string]string
	snakeToCamelCache map[string]string
	maxCacheSize      int
	mu                sync.RWMutex
}

// NewManager 创建新的缓存管理器
func NewManager(maxSize int) *Manager {
	return &Manager{
		camelToSnakeCache: make(map[string]string),
		snakeToCamelCache: make(map[string]string),
		maxCacheSize:      maxSize,
	}
}

// 全局缓存管理器实例
var (
	GlobalManager = NewManager(1000) // 默认最大缓存1000个条目
)

// CleanCache 清理缓存（当缓存过大时）
func (m *Manager) CleanCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.camelToSnakeCache) > m.maxCacheSize {
		// 清理一半的缓存条目
		newCache := make(map[string]string)
		count := 0
		for k, v := range m.camelToSnakeCache {
			if count < m.maxCacheSize/2 {
				newCache[k] = v
				count++
			} else {
				break
			}
		}
		m.camelToSnakeCache = newCache
	}

	if len(m.snakeToCamelCache) > m.maxCacheSize {
		// 清理一半的缓存条目
		newCache := make(map[string]string)
		count := 0
		for k, v := range m.snakeToCamelCache {
			if count < m.maxCacheSize/2 {
				newCache[k] = v
				count++
			} else {
				break
			}
		}
		m.snakeToCamelCache = newCache
	}
}

// SetCamelToSnake 设置camelCase到snake_case的缓存（带清理检查）
func (m *Manager) SetCamelToSnake(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.camelToSnakeCache) >= m.maxCacheSize {
		m.cleanCacheUnsafe() // 内部调用，不加锁
	}
	m.camelToSnakeCache[key] = value
}

// SetSnakeToCamel 设置snake_case到camelCase的缓存
func (m *Manager) SetSnakeToCamel(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.snakeToCamelCache) >= m.maxCacheSize {
		m.cleanCacheUnsafe() // 内部调用，不加锁
	}
	m.snakeToCamelCache[key] = value
}

// GetCamelToSnake 获取camelCase到snake_case的缓存
func (m *Manager) GetCamelToSnake(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.camelToSnakeCache[key]
	return value, exists
}

// GetSnakeToCamel 获取snake_case到camelCase的缓存
func (m *Manager) GetSnakeToCamel(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.snakeToCamelCache[key]
	return value, exists
}

// cleanCacheUnsafe 内部清理缓存方法（不加锁，调用者需要确保已加锁）
func (m *Manager) cleanCacheUnsafe() {
	if len(m.camelToSnakeCache) > m.maxCacheSize {
		// 清理一半的缓存条目
		newCache := make(map[string]string)
		count := 0
		for k, v := range m.camelToSnakeCache {
			if count < m.maxCacheSize/2 {
				newCache[k] = v
				count++
			} else {
				break
			}
		}
		m.camelToSnakeCache = newCache
	}

	if len(m.snakeToCamelCache) > m.maxCacheSize {
		// 清理一半的缓存条目
		newCache := make(map[string]string)
		count := 0
		for k, v := range m.snakeToCamelCache {
			if count < m.maxCacheSize/2 {
				newCache[k] = v
				count++
			} else {
				break
			}
		}
		m.snakeToCamelCache = newCache
	}
}

// ClearAll 清空所有缓存
func (m *Manager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.camelToSnakeCache = make(map[string]string)
	m.snakeToCamelCache = make(map[string]string)
}

// Stats 返回缓存统计信息
func (m *Manager) Stats() (camelToSnakeCount, snakeToCamelCount int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.camelToSnakeCache), len(m.snakeToCamelCache)
}
