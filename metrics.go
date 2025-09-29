package sysconf

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics 配置性能指标
type Metrics struct {
	mu             sync.RWMutex
	StartTime      time.Time                `json:"start_time"`
	GetCount       int64                    `json:"get_count"`
	SetCount       int64                    `json:"set_count"`
	CacheHits      int64                    `json:"cache_hits"`
	CacheMisses    int64                    `json:"cache_misses"`
	LastGetTime    time.Time                `json:"last_get_time"`
	LastSetTime    time.Time                `json:"last_set_time"`
	ErrorCount     int64                    `json:"error_count"`
	OperationTimes map[string]time.Duration `json:"operation_times"`

	// 内部计数器
	totalGetTime int64 // 累积的Get操作时间（纳秒）
	totalSetTime int64 // 累积的Set操作时间（纳秒）
}

// NewMetrics 创建新的性能指标实例
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime:      time.Now(),
		OperationTimes: make(map[string]time.Duration),
	}
}

// RecordGet 记录Get操作
func (m *Metrics) RecordGet(duration time.Duration, cacheHit bool) {
	atomic.AddInt64(&m.GetCount, 1)
	atomic.AddInt64(&m.totalGetTime, int64(duration))

	if cacheHit {
		atomic.AddInt64(&m.CacheHits, 1)
	} else {
		atomic.AddInt64(&m.CacheMisses, 1)
	}

	m.mu.Lock()
	m.LastGetTime = time.Now()
	m.mu.Unlock()
}

// RecordSet 记录Set操作
func (m *Metrics) RecordSet(duration time.Duration) {
	atomic.AddInt64(&m.SetCount, 1)
	atomic.AddInt64(&m.totalSetTime, int64(duration))

	m.mu.Lock()
	m.LastSetTime = time.Now()
	m.mu.Unlock()
}

// RecordError 记录错误
func (m *Metrics) RecordError() {
	atomic.AddInt64(&m.ErrorCount, 1)
}

// RecordOperation 记录自定义操作时间
func (m *Metrics) RecordOperation(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OperationTimes[name] = duration
}

// GetStats 获取统计信息
func (m *Metrics) GetStats() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	getCount := atomic.LoadInt64(&m.GetCount)
	setCount := atomic.LoadInt64(&m.SetCount)
	totalGetTime := atomic.LoadInt64(&m.totalGetTime)
	totalSetTime := atomic.LoadInt64(&m.totalSetTime)

	stats := MetricsSnapshot{
		StartTime:      m.StartTime,
		Uptime:         time.Since(m.StartTime),
		GetCount:       getCount,
		SetCount:       setCount,
		CacheHits:      atomic.LoadInt64(&m.CacheHits),
		CacheMisses:    atomic.LoadInt64(&m.CacheMisses),
		ErrorCount:     atomic.LoadInt64(&m.ErrorCount),
		LastGetTime:    m.LastGetTime,
		LastSetTime:    m.LastSetTime,
		OperationTimes: make(map[string]time.Duration),
	}

	// 复制操作时间
	for k, v := range m.OperationTimes {
		stats.OperationTimes[k] = v
	}

	// 计算平均时间
	if getCount > 0 {
		stats.AvgGetTime = time.Duration(totalGetTime / getCount)
	}
	if setCount > 0 {
		stats.AvgSetTime = time.Duration(totalSetTime / setCount)
	}

	// 计算缓存命中率
	totalCacheOps := stats.CacheHits + stats.CacheMisses
	if totalCacheOps > 0 {
		stats.CacheHitRatio = float64(stats.CacheHits) / float64(totalCacheOps) * 100
	}

	return stats
}

// Reset 重置统计信息
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt64(&m.GetCount, 0)
	atomic.StoreInt64(&m.SetCount, 0)
	atomic.StoreInt64(&m.CacheHits, 0)
	atomic.StoreInt64(&m.CacheMisses, 0)
	atomic.StoreInt64(&m.ErrorCount, 0)
	atomic.StoreInt64(&m.totalGetTime, 0)
	atomic.StoreInt64(&m.totalSetTime, 0)

	m.StartTime = time.Now()
	m.LastGetTime = time.Time{}
	m.LastSetTime = time.Time{}
	m.OperationTimes = make(map[string]time.Duration)
}

// MetricsSnapshot 性能指标快照
type MetricsSnapshot struct {
	StartTime      time.Time                `json:"start_time"`
	Uptime         time.Duration            `json:"uptime"`
	GetCount       int64                    `json:"get_count"`
	SetCount       int64                    `json:"set_count"`
	CacheHits      int64                    `json:"cache_hits"`
	CacheMisses    int64                    `json:"cache_misses"`
	CacheHitRatio  float64                  `json:"cache_hit_ratio"`
	ErrorCount     int64                    `json:"error_count"`
	AvgGetTime     time.Duration            `json:"avg_get_time"`
	AvgSetTime     time.Duration            `json:"avg_set_time"`
	LastGetTime    time.Time                `json:"last_get_time"`
	LastSetTime    time.Time                `json:"last_set_time"`
	OperationTimes map[string]time.Duration `json:"operation_times"`
}

// GetSummary 获取性能摘要字符串
func (s MetricsSnapshot) GetSummary() string {
	return fmt.Sprintf(
		"Config Performance Summary:\n"+
			"  Uptime: %v\n"+
			"  Operations: %d gets, %d sets\n"+
			"  Cache: %.2f%% hit ratio (%d hits, %d misses)\n"+
			"  Avg Times: %v get, %v set\n"+
			"  Errors: %d\n",
		s.Uptime,
		s.GetCount, s.SetCount,
		s.CacheHitRatio, s.CacheHits, s.CacheMisses,
		s.AvgGetTime, s.AvgSetTime,
		s.ErrorCount,
	)
}

// GetMetrics 获取配置的性能指标（使用全局监控器）
func (c *Config) GetMetrics() MetricsSnapshot {
	return GetGlobalMetrics()
}

// ResetMetrics 重置性能指标（使用全局监控器）
func (c *Config) ResetMetrics() {
	ResetGlobalMetrics()
}

// 在Config结构体中添加metrics字段需要更新config.go
// 这里为了不修改现有结构，使用全局监控器

var (
	globalMetrics     *Metrics
	globalMetricsOnce sync.Once
)

// getGlobalMetrics 获取全局性能监控器
func getGlobalMetrics() *Metrics {
	globalMetricsOnce.Do(func() {
		globalMetrics = NewMetrics()
	})
	return globalMetrics
}

// GetGlobalMetrics 获取全局性能指标
func GetGlobalMetrics() MetricsSnapshot {
	return getGlobalMetrics().GetStats()
}

// ResetGlobalMetrics 重置全局性能指标
func ResetGlobalMetrics() {
	getGlobalMetrics().Reset()
}

// recordGetOperation 记录Get操作（内部使用）
func recordGetOperation(duration time.Duration, cacheHit bool) {
	getGlobalMetrics().RecordGet(duration, cacheHit)
}

// recordSetOperation 记录Set操作（内部使用）
func recordSetOperation(duration time.Duration) {
	getGlobalMetrics().RecordSet(duration)
}

// recordErrorOperation 记录错误操作（内部使用）
func recordErrorOperation() {
	getGlobalMetrics().RecordError()
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	config *Config
	ticker *time.Ticker
	done   chan struct{}
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(config *Config, interval time.Duration) *PerformanceMonitor {
	return &PerformanceMonitor{
		config: config,
		ticker: time.NewTicker(interval),
		done:   make(chan struct{}),
	}
}

// Start 启动性能监控
func (pm *PerformanceMonitor) Start() {
	go func() {
		for {
			select {
			case <-pm.ticker.C:
				stats := GetGlobalMetrics()
				pm.config.logger.Infof("Performance Stats: Gets=%d, Sets=%d, Cache Hit=%.1f%%, Errors=%d",
					stats.GetCount, stats.SetCount, stats.CacheHitRatio, stats.ErrorCount)

				// 检查性能警告
				if stats.CacheHitRatio < 50 && stats.CacheHits+stats.CacheMisses > 100 {
					pm.config.logger.Warnf("Low cache hit ratio: %.1f%%", stats.CacheHitRatio)
				}

				if stats.AvgGetTime > 10*time.Millisecond {
					pm.config.logger.Warnf("Slow get operations: avg %v", stats.AvgGetTime)
				}

			case <-pm.done:
				return
			}
		}
	}()
}

// Stop 停止性能监控
func (pm *PerformanceMonitor) Stop() {
	pm.ticker.Stop()
	close(pm.done)
}
