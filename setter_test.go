package sysconf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// limitValidator 简单验证器：限制 number 不得超过 10
type limitValidator struct{}

func (limitValidator) Validate(m map[string]any) error {
	if v, ok := m["number"].(int); ok && v > 10 {
		return fmt.Errorf("number too large")
	}
	return nil
}

func (limitValidator) GetName() string { return "default rollback validator" }

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

	// 测试: 验证失败时保持原值
	t.Run("验证失败回滚原值", func(t *testing.T) {
		cfg, err := New(
			WithContent("number: 5"),
			WithValidator(limitValidator{}),
		)
		if !assert.NoError(t, err, "回滚测试初始化失败") {
			t.FailNow()
		}

		assert.Equal(t, 5, cfg.GetInt("number"))

		// 非法值应触发错误且保持原值不变
		err = cfg.Set("number", 20)
		assert.Error(t, err)
		assert.Equal(t, 5, cfg.GetInt("number"), "验证失败后原值应被保留")
	})

	// 测试: 写入失败回滚
	t.Run("写入失败自动回滚", func(t *testing.T) {
		roDir, err := os.MkdirTemp("", "deny_write")
		assert.NoError(t, err)
		defer os.RemoveAll(roDir)

		configFile := filepath.Join(roDir, "rollback_write.yaml")
		assert.NoError(t, os.WriteFile(configFile, []byte("number: 1\n"), 0o400)) // 只读文件

		cfg, err := New(
			WithPath(roDir),
			WithName("rollback_write"),
			WithMode("yaml"),
		)
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		// 强制走加密写入路径并制造序列化错误
		cfg.cryptoOptions.Enabled = true
		cfg.mode = "invalid" // marshalConfig 不支持，写入应失败

		assert.Equal(t, 1, cfg.GetInt("number"))

		err = cfg.Set("number", 2)
		assert.Error(t, err)
		assert.Equal(t, 1, cfg.GetInt("number"), "写入失败后应回滚为旧值")
	})

	// 测试: 并发 Set 保持一致性
	t.Run("并发设置保持一致性", func(t *testing.T) {
		cfg, err := New(
			WithContent("a: 1\nb: 1"),
		)
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func(i int) {
				_ = cfg.Set(fmt.Sprintf("a.%d", i), i)
				done <- struct{}{}
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		// 确认不会 panic 且数据可读
		assert.True(t, cfg.IsSet("a.0"))
	})
}

func TestSetEnvPrefix(t *testing.T) {
	t.Skip("环境变量设置测试依赖于文件系统，暂时跳过。")
}
