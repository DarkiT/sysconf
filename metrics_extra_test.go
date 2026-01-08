package sysconf

import (
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
