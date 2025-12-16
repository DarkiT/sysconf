package sysconf

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darkit/sysconf/internal/testutil"
)

func TestScheduleCacheUpdateDelay(t *testing.T) {
	cfg := newTestConfig(t)
	testutil.Cleanup(t, cfg.Close)

	cfg.cacheEnabled.Store(true)
	cfg.cacheRebuildDelay = 5 * time.Millisecond
	startVer := atomic.LoadInt64(&cfg.cacheVersion)
	cfg.scheduleCacheUpdate(cfg.cacheRebuildDelay)
	time.Sleep(15 * time.Millisecond)
	if atomic.LoadInt64(&cfg.cacheVersion) == startVer {
		t.Fatalf("cache version should increase after delayed update")
	}
}

func TestLoadReadCacheDisabled(t *testing.T) {
	cfg := newTestConfig(t)
	testutil.Cleanup(t, cfg.Close)
	cfg.cacheEnabled.Store(false)
	if cfg.loadReadCache() != nil {
		t.Fatalf("disabled cache should return nil")
	}
}

func TestFlattenMapToCacheInterfaceKeys(t *testing.T) {
	cfg := newTestConfig(t)
	testutil.Cleanup(t, cfg.Close)
	cache := make(map[string]any)
	m := map[interface{}]any{
		"level1": map[interface{}]any{
			"level2": "val",
		},
	}
	cfg.flattenMapToCache("root", m, cache)
	found := false
	for k, v := range cache {
		if strings.HasSuffix(k, "level2") && v == "val" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected flattened value present")
	}
}
