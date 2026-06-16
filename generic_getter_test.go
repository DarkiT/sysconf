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
	require.NoError(t, cfg.Set("port", 8080))

	val, err := GetAsWithError[int](cfg, "port")
	assert.NoError(t, err)
	assert.Equal(t, 8080, val)

	_, err = GetAsWithError[int](cfg, "missing")
	assert.Error(t, err)

	require.NoError(t, cfg.Set("bad", "oops"))
	_, err = GetAsWithError[int](cfg, "bad")
	assert.Error(t, err)
}

func TestMustGetAs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := setupConfig(t)
		require.NoError(t, cfg.Set("port", 8080))

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
		require.NoError(t, cfg.Set("bad_number", "not_a_number"))

		assert.Panics(t, func() {
			_ = MustGetAs[int](cfg, "bad_number")
		})
	})
}

func TestMustGetAsVariousTypes(t *testing.T) {
	cfg := setupConfig(t)
	require.NoError(t, cfg.Set("str", "text"))
	require.NoError(t, cfg.Set("num", 42))
	require.NoError(t, cfg.Set("flt", 3.14))
	require.NoError(t, cfg.Set("flag", true))
	require.NoError(t, cfg.Set("dur", "5s"))

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
	require.NoError(t, cfg.Set("bad", "oops"))
	assert.Equal(t, 7, GetAs(cfg, "bad", 7))
	// 无默认值返回零值并记录 warn（不可观测，仅验证返回值）
	assert.Equal(t, 0, GetAs[int](cfg, "bad"))
}

func TestGenericGettersNilConfigAndAnyType(t *testing.T) {
	assert.Equal(t, "fallback", GetAs[string](nil, "missing", "fallback"))
	assert.Equal(t, 0, GetAs[int](nil, "missing"))
	assert.Empty(t, GetSliceAs[string](nil, "missing"))
	assert.Equal(t, 0, GetWithFallback[int](nil, "missing"))

	var nilCfg *Config
	_, err := GetAsWithError[string](nilCfg, "missing")
	assert.Error(t, err)

	cfg := setupConfig(t)
	require.NoError(t, cfg.Set("raw", map[string]any{"k": "v"}))
	raw := GetAs[any](cfg, "raw")
	require.IsType(t, map[string]any{}, raw)
	raw.(map[string]any)["k"] = "mutated"
	assert.Equal(t, "v", cfg.GetString("raw.k"))
}

func TestGetSliceAsVariants(t *testing.T) {
	cfg := setupConfig(t)

	require.NoError(t, cfg.Set("direct_str", []string{"a", "b"}))
	direct := GetSliceAs[string](cfg, "direct_str")
	assert.ElementsMatch(t, []string{"a", "b"}, direct)
	direct[0] = "mutated"
	assert.ElementsMatch(t, []string{"a", "b"}, GetSliceAs[string](cfg, "direct_str"))

	require.NoError(t, cfg.Set("iface_mix", []any{"1", 2, true}))
	assert.ElementsMatch(t, []int{1, 2, 1}, GetSliceAs[int](cfg, "iface_mix")) // bool->int 为1
	assert.ElementsMatch(t, []bool{true, true, true}, GetSliceAs[bool](cfg, "iface_mix"))

	require.NoError(t, cfg.Set("int_slice", []int{3, 4}))
	assert.ElementsMatch(t, []float64{3, 4}, GetSliceAs[float64](cfg, "int_slice"))

	assert.Empty(t, GetSliceAs[int](cfg, "missing"))
	assert.Empty(t, GetSliceAs[int](cfg, ""))
}

func TestGetWithFallback(t *testing.T) {
	cfg := setupConfig(t)
	require.NoError(t, cfg.Set("primary", "p"))
	require.NoError(t, cfg.Set("backup", "b"))

	assert.Equal(t, "p", GetWithFallback[string](cfg, "primary", "backup"))
	assert.Equal(t, "b", GetWithFallback[string](cfg, "missing", "backup"))
	assert.Equal(t, 0, GetWithFallback[int](cfg, "missing", "")) // 全部缺失返回零值
}
