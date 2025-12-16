package sysconf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupConfig 创建一个默认配置供测试复用
func setupConfig(t *testing.T) *Config {
	t.Helper()
	cfg, err := New()
	require.NoError(t, err)
	return cfg
}

func TestGetAsWithError(t *testing.T) {
	cfg := setupConfig(t)
	cfg.Set("port", 8080)

	val, err := GetAsWithError[int](cfg, "port")
	assert.NoError(t, err)
	assert.Equal(t, 8080, val)

	_, err = GetAsWithError[int](cfg, "missing")
	assert.Error(t, err)

	cfg.Set("bad", "oops")
	_, err = GetAsWithError[int](cfg, "bad")
	assert.Error(t, err)
}

func TestMustGetAs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := setupConfig(t)
		cfg.Set("port", 8080)

		port := MustGetAs[int](cfg, "port")
		assert.Equal(t, 8080, port)
	})

	t.Run("panic on missing key", func(t *testing.T) {
		cfg := setupConfig(t)
		assert.Panics(t, func() {
			_ = MustGetAs[string](cfg, "missing_key")
		})
	})

	t.Run("panic on type conversion failure", func(t *testing.T) {
		cfg := setupConfig(t)
		cfg.Set("port", "not_a_number")

		assert.Panics(t, func() {
			_ = MustGetAs[int](cfg, "port")
		})
	})
}

func TestMustGetAsVariousTypes(t *testing.T) {
	cfg := setupConfig(t)
	cfg.Set("str", "text")
	cfg.Set("num", 42)
	cfg.Set("flt", 3.14)
	cfg.Set("flag", true)
	cfg.Set("dur", "5s")

	assert.Equal(t, "text", MustGetAs[string](cfg, "str"))
	assert.Equal(t, 42, MustGetAs[int](cfg, "num"))
	assert.InDelta(t, 3.14, MustGetAs[float64](cfg, "flt"), 0.0001)
	assert.True(t, MustGetAs[bool](cfg, "flag"))
	assert.Equal(t, 5*time.Second, MustGetAs[time.Duration](cfg, "dur"))
}

func TestGetAsCacheDefaultAndFailure(t *testing.T) {
	cfg := setupConfig(t)

	// cache 命中路径
	cfg.cacheEnabled.Store(true)
	cfg.readCache.Store(map[string]any{"cached": "123"})
	assert.Equal(t, 123, GetAs[int](cfg, "cached"))

	// 空键返回默认或零值
	assert.Equal(t, "d", GetAs(cfg, "", "d"))
	assert.Equal(t, 0, GetAs[int](cfg, ""))

	// 转换失败使用默认值
	cfg.Set("bad", "oops")
	assert.Equal(t, 7, GetAs(cfg, "bad", 7))
	// 无默认值返回零值并记录 warn（不可观测，仅验证返回值）
	assert.Equal(t, 0, GetAs[int](cfg, "bad"))
}

func TestGetSliceAsVariants(t *testing.T) {
	cfg := setupConfig(t)

	cfg.Set("direct_str", []string{"a", "b"})
	assert.ElementsMatch(t, []string{"a", "b"}, GetSliceAs[string](cfg, "direct_str"))

	cfg.Set("iface_mix", []interface{}{"1", 2, true})
	assert.ElementsMatch(t, []int{1, 2, 1}, GetSliceAs[int](cfg, "iface_mix")) // bool->int 为1
	assert.ElementsMatch(t, []bool{true, true, true}, GetSliceAs[bool](cfg, "iface_mix"))

	cfg.Set("int_slice", []int{3, 4})
	assert.ElementsMatch(t, []float64{3, 4}, GetSliceAs[float64](cfg, "int_slice"))

	assert.Empty(t, GetSliceAs[int](cfg, "missing"))
	assert.Empty(t, GetSliceAs[int](cfg, ""))
}

func TestGetWithFallback(t *testing.T) {
	cfg := setupConfig(t)
	cfg.Set("primary", "p")
	cfg.Set("backup", "b")

	assert.Equal(t, "p", GetWithFallback[string](cfg, "primary", "backup"))
	assert.Equal(t, "b", GetWithFallback[string](cfg, "missing", "backup"))
	assert.Equal(t, 0, GetWithFallback[int](cfg, "missing", "")) // 全部缺失返回零值
}
