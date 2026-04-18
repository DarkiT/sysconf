package sysconf

import (
	"sync"
	"testing"
	"time"
)

func TestMetricsSnapshotsAndReset(t *testing.T) {
	DisableMetrics()
	defer EnableMetrics()

	m := NewMetrics()
	m.RecordGet(2*time.Millisecond, true)
	m.RecordGet(4*time.Millisecond, false)
	m.RecordSet(3 * time.Millisecond)
	m.RecordError()
	m.RecordOperation("custom", 5*time.Millisecond)

	snap := m.GetStats()
	if snap.GetCount != 2 || snap.SetCount != 1 || snap.ErrorCount != 1 {
		t.Fatalf("stats mismatch: %+v", snap)
	}
	if snap.CacheHitRatio <= 0 || snap.AvgGetTime == 0 || snap.AvgSetTime == 0 {
		t.Fatalf("derived metrics not calculated: %+v", snap)
	}
	if snap.OperationTimes["custom"] == 0 {
		t.Fatalf("operation time missing")
	}

	// Reset 清零
	m.Reset()
	snap = m.GetStats()
	if snap.GetCount != 0 || snap.SetCount != 0 || len(snap.OperationTimes) != 0 {
		t.Fatalf("reset failed: %+v", snap)
	}
}

// TestMetricsConcurrency 测试 Metrics 的并发安全性
func TestMetricsConcurrency(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup
	iterations := 100

	// 并发执行 Get、Set、Error 和 Operation 记录
	for i := 0; i < iterations; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			m.RecordGet(time.Millisecond, true)
		}()
		go func() {
			defer wg.Done()
			m.RecordSet(time.Millisecond)
		}()
		go func() {
			defer wg.Done()
			m.RecordError()
		}()
		go func(idx int) {
			defer wg.Done()
			m.RecordOperation("op", time.Duration(idx)*time.Microsecond)
		}(i)
	}

	// 同时并发读取统计
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.GetStats()
		}()
	}

	wg.Wait()

	// 验证计数
	stats := m.GetStats()
	if stats.GetCount != int64(iterations) {
		t.Errorf("expected %d gets, got %d", iterations, stats.GetCount)
	}
	if stats.SetCount != int64(iterations) {
		t.Errorf("expected %d sets, got %d", iterations, stats.SetCount)
	}
	if stats.ErrorCount != int64(iterations) {
		t.Errorf("expected %d errors, got %d", iterations, stats.ErrorCount)
	}
	if stats.CacheHits != int64(iterations) {
		t.Errorf("expected %d cache hits, got %d", iterations, stats.CacheHits)
	}
}

// TestOperationStats 测试操作统计累积功能
func TestOperationStats(t *testing.T) {
	m := NewMetrics()

	// 记录多次操作
	m.RecordOperation("test_op", 100*time.Microsecond)
	m.RecordOperation("test_op", 200*time.Microsecond)
	m.RecordOperation("test_op", 50*time.Microsecond)
	m.RecordOperation("test_op", 300*time.Microsecond)

	stats := m.GetStats()

	// 验证操作统计
	opStats, ok := stats.OperationStats["test_op"]
	if !ok {
		t.Fatal("operation stats not found")
	}

	if opStats.Count != 4 {
		t.Errorf("expected count 4, got %d", opStats.Count)
	}

	expectedTotal := int64(650 * time.Microsecond)
	if opStats.TotalNs != expectedTotal {
		t.Errorf("expected total %d, got %d", expectedTotal, opStats.TotalNs)
	}

	expectedMin := int64(50 * time.Microsecond)
	if opStats.MinNs != expectedMin {
		t.Errorf("expected min %d, got %d", expectedMin, opStats.MinNs)
	}

	expectedMax := int64(300 * time.Microsecond)
	if opStats.MaxNs != expectedMax {
		t.Errorf("expected max %d, got %d", expectedMax, opStats.MaxNs)
	}

	expectedLast := int64(300 * time.Microsecond)
	if opStats.LastNs != expectedLast {
		t.Errorf("expected last %d, got %d", expectedLast, opStats.LastNs)
	}
}

// TestMetricsSummary 测试性能摘要输出
func TestMetricsSummary(t *testing.T) {
	m := NewMetrics()
	m.RecordGet(time.Millisecond, true)
	m.RecordSet(time.Millisecond)

	stats := m.GetStats()
	summary := stats.GetSummary()

	if summary == "" {
		t.Error("summary should not be empty")
	}
}
