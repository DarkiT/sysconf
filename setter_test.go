package sysconf

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "sysconf_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 测试: 基本设置功能
	t.Run("基本设置功能", func(t *testing.T) {
		c, err := New(
			WithPath(tempDir),
			WithName("test_set"),
			WithMode("yaml"),
		)
		if !assert.NoError(t, err, "配置初始化应该成功") {
			t.FailNow()
		}

		// 设置字符串
		err = c.Set("string_key", "字符串值")
		assert.NoError(t, err)
		assert.Equal(t, "字符串值", c.GetString("string_key"))

		// 设置整数
		err = c.Set("int_key", 123)
		assert.NoError(t, err)
		assert.Equal(t, 123, c.GetInt("int_key"))

		// 设置浮点数
		err = c.Set("float_key", 3.14)
		assert.NoError(t, err)
		assert.Equal(t, 3.14, c.GetFloat("float_key"))

		// 设置布尔值
		err = c.Set("bool_key", true)
		assert.NoError(t, err)
		assert.Equal(t, true, c.GetBool("bool_key"))

		// 设置切片
		err = c.Set("slice_key", []string{"值1", "值2", "值3"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"值1", "值2", "值3"}, c.GetStringSlice("slice_key"))

		// 设置映射
		err = c.Set("map_key", map[string]interface{}{"子键1": "子值1", "子键2": 123})
		assert.NoError(t, err)
		assert.Equal(t, "子值1", c.GetString("map_key.子键1"))
		assert.Equal(t, 123, c.GetInt("map_key.子键2"))
	})

	// 测试: 嵌套键
	t.Run("嵌套键", func(t *testing.T) {
		c, err := New(
			WithPath(tempDir),
			WithName("test_set_nested"),
		)
		if !assert.NoError(t, err, "嵌套键配置初始化应该成功") {
			t.FailNow()
		}

		// 设置嵌套值
		err = c.Set("parent.child.grandchild", "嵌套值")
		assert.NoError(t, err)
		assert.Equal(t, "嵌套值", c.GetString("parent.child.grandchild"))

		// 整个对象覆盖
		err = c.Set("parent.child", map[string]interface{}{
			"grandchild": "新嵌套值",
			"sibling":    "兄弟值",
		})
		assert.NoError(t, err)
		assert.Equal(t, "新嵌套值", c.GetString("parent.child.grandchild"))
		assert.Equal(t, "兄弟值", c.GetString("parent.child.sibling"))
	})

	// 测试: 空键
	t.Run("空键", func(t *testing.T) {
		c, err := New()
		if !assert.NoError(t, err, "空键测试配置初始化应该成功") {
			t.FailNow()
		}

		err = c.Set("", "任何值")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKey, err)
	})

	// 测试: 延迟写入
	t.Run("延迟写入", func(t *testing.T) {
		c, err := New(
			WithPath(tempDir),
			WithName("test_delayed_write"),
		)
		if !assert.NoError(t, err, "延迟写入配置初始化应该成功") {
			t.FailNow()
		}

		// 设置值
		err = c.Set("delayed_key", "延迟写入值")
		assert.NoError(t, err)

		// 立即读取
		assert.Equal(t, "延迟写入值", c.GetString("delayed_key"))

		// 等待写入
		time.Sleep(4 * time.Second)

		// 重新加载确认持久化
		newConfig, err := New(
			WithPath(tempDir),
			WithName("test_delayed_write"),
		)
		if !assert.NoError(t, err, "延迟写入后重新加载配置应该成功") {
			t.FailNow()
		}
		assert.Equal(t, "延迟写入值", newConfig.GetString("delayed_key"))
	})
}

func TestSetEnvPrefix(t *testing.T) {
	t.Skip("环境变量设置测试依赖于文件系统，暂时跳过。")
}
