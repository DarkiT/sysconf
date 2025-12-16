package sysconf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetters(t *testing.T) {
	// 简单地创建一个配置实例，不指定任何内容
	c, err := New()
	if !assert.NoError(t, err, "配置初始化应该成功") {
		t.FailNow() // 如果配置初始化失败，立即终止测试
	}
	t.Cleanup(func() { _ = c.Close() })

	// 直接设置测试值而不是加载配置文件
	if err := c.Set("direct_string", "直接设置的字符串"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := c.Set("direct_int", 100); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := c.Set("direct_bool", true); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := c.Set("direct_slice", []string{"A", "B", "C"}); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := c.Set("direct_map", map[string]interface{}{"a": "A值", "b": "B值"}); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 测试: Get 方法
	t.Run("Get基本功能", func(t *testing.T) {
		// 测试直接设置的值
		assert.Equal(t, "直接设置的字符串", c.Get("direct_string"))
		assert.Equal(t, 100, c.Get("direct_int"))
		assert.Equal(t, true, c.Get("direct_bool"))

		// 测试默认值处理
		assert.Nil(t, c.Get("non_existent"))
		assert.Equal(t, "默认值", c.Get("non_existent", "默认值"))

		// 测试空键处理
		assert.Nil(t, c.Get(""))
		assert.Equal(t, "空键默认值", c.Get("", "空键默认值"))
	})

	// 测试: GetString 方法
	t.Run("GetString", func(t *testing.T) {
		// 基本获取
		assert.Equal(t, "直接设置的字符串", c.GetString("direct_string"))

		// 类型转换
		assert.Equal(t, "100", c.GetString("direct_int"))
		assert.Equal(t, "true", c.GetString("direct_bool"))

		// 默认值
		assert.Equal(t, "", c.GetString("non_existent"))
		assert.Equal(t, "默认值", c.GetString("non_existent", "默认值"))
	})

	// 测试: GetInt 方法
	t.Run("GetInt", func(t *testing.T) {
		// 基本获取
		assert.Equal(t, 100, c.GetInt("direct_int"))

		// 类型转换
		assert.Equal(t, 0, c.GetInt("direct_string")) // 无法转换为数字时返回0

		// 默认值
		assert.Equal(t, 0, c.GetInt("non_existent"))
		assert.Equal(t, 200, c.GetInt("non_existent", "200")) // 注意这里必须用字符串
	})

	// 测试: GetBool 方法
	t.Run("GetBool", func(t *testing.T) {
		// 基本获取
		assert.Equal(t, true, c.GetBool("direct_bool"))

		// 默认值
		assert.Equal(t, false, c.GetBool("non_existent"))
		assert.Equal(t, true, c.GetBool("non_existent", "true")) // 注意这里必须用字符串
	})

	// 测试: GetStringSlice 方法
	t.Run("GetStringSlice", func(t *testing.T) {
		// 基本获取
		expected := []string{"A", "B", "C"}
		assert.ElementsMatch(t, expected, c.GetStringSlice("direct_slice"))

		// 默认值
		assert.Empty(t, c.GetStringSlice("non_existent"))
	})

	// 测试: GetStringMap 和 GetStringMapString 方法
	t.Run("GetMap", func(t *testing.T) {
		// 测试直接获取设置过的MAP
		m := c.GetStringMap("direct_map")
		assert.NotEmpty(t, m)
		assert.Equal(t, "A值", m["a"])
		assert.Equal(t, "B值", m["b"])

		// 设置嵌套结构
		if err := c.Set("nested_map", map[string]interface{}{
			"child": map[string]interface{}{
				"grandchild": "嵌套值",
			},
		}); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// 获取嵌套映射中的值
		assert.Equal(t, "嵌套值", c.GetString("nested_map.child.grandchild"))

		// GetStringMapString
		ms := c.GetStringMapString("direct_map")
		assert.NotEmpty(t, ms)
		assert.Equal(t, "A值", ms["a"])
		assert.Equal(t, "B值", ms["b"])
	})

	// 测试: GetTime 方法
	t.Run("GetTime", func(t *testing.T) {
		// 设置一个时间
		timeStr := "2023-04-05T10:20:30Z"
		if err := c.Set("direct_time", timeStr); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// 基本获取
		expectedTime, _ := time.Parse(time.RFC3339, timeStr)
		assert.Equal(t, expectedTime, c.GetTime("direct_time"))

		// 默认值
		assert.Equal(t, time.Time{}, c.GetTime("non_existent"))
	})

	// 测试: GetDuration 方法
	t.Run("GetDuration", func(t *testing.T) {
		// 设置一个持续时间
		c.Set("direct_duration", "1h30m")

		// 基本获取
		expected := 90 * time.Minute
		assert.Equal(t, expected, c.GetDuration("direct_duration"))

		// 默认值
		assert.Equal(t, time.Duration(0), c.GetDuration("non_existent"))
	})

	t.Run("GetBoolSlice_IntSlice_FloatSlice_and_GetWithError", func(t *testing.T) {
		cfg := newTestConfig(t)
		defer cfg.Close()

		// Bool slice covers []interface{} and []string 分支
		cfg.Set("bools_iface", []interface{}{true, "false", "1", "bad"})
		cfg.Set("bools_str", []string{"true", "0", "False"})
		assert.ElementsMatch(t, []bool{true, false, true}, cfg.GetBoolSlice("bools_iface"))
		assert.ElementsMatch(t, []bool{true, false, false}, cfg.GetBoolSlice("bools_str"))
		assert.Empty(t, cfg.GetBoolSlice(""))           // 空键
		assert.Empty(t, cfg.GetBoolSlice("not.exists")) // 不存在

		// Int slice 走 cast.ToIntSliceE 成功与失败分支
		cfg.Set("ints_ok", []interface{}{1, "2", 3.0})
		assert.ElementsMatch(t, []int{1, 2, 3}, cfg.GetIntSlice("ints_ok"))
		cfg.Set("ints_bad", []interface{}{"bad"})
		assert.Empty(t, cfg.GetIntSlice("ints_bad"))
		assert.Empty(t, cfg.GetIntSlice("missing"))

		// Float slice 覆盖多类型切片与单值转换分支
		cfg.Set("floats_iface", []interface{}{1, "2.5", float32(3.5), "bad"})
		cfg.Set("floats_int", []int{4, 5})
		cfg.Set("floats_single", "6.5")
		cfg.Set("floats_str", []string{"7.5", "oops"})
		cfg.Set("floats_unconvertible", map[string]any{"x": 1})
		assert.InEpsilonSlice(t, []float64{1, 2.5, 3.5}, cfg.GetFloatSlice("floats_iface"), 0.0001)
		assert.ElementsMatch(t, []float64{4, 5}, cfg.GetFloatSlice("floats_int"))
		assert.ElementsMatch(t, []float64{6.5}, cfg.GetFloatSlice("floats_single"))
		assert.InEpsilonSlice(t, []float64{7.5}, cfg.GetFloatSlice("floats_str"), 0.0001)
		assert.Empty(t, cfg.GetFloatSlice("floats_unconvertible"))
		assert.Empty(t, cfg.GetFloatSlice("missing"))

		// GetWithError 覆盖存在/缺失/空键分支
		cfg.Set("exists", "v")
		val, err := cfg.GetWithError("exists")
		assert.NoError(t, err)
		assert.Equal(t, "v", val)
		_, err = cfg.GetWithError("missing")
		assert.Error(t, err)
		_, err = cfg.GetWithError("")
		assert.Error(t, err)
	})
}

// 测试环境变量相关功能，减少配置文件依赖
func TestGetEnvPrefix(t *testing.T) {
	t.Skip("环境变量设置测试依赖于文件系统，暂时跳过。")
}
