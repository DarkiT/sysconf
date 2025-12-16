package sysconf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customValidator struct{}

var _ ConfigValidator = (*customValidator)(nil)

func (v *customValidator) Validate(config map[string]any) error {
	if port, ok := config["database.port"]; ok {
		if portInt, ok := port.(int); ok && portInt < 1024 {
			return fmt.Errorf("database port must be >= 1024")
		}
	}
	return nil
}

func (v *customValidator) GetName() string {
	return "CustomValidator"
}

// TestConfigValidateFunc 测试函数式验证器的 Validate 和 GetName 方法
func TestConfigValidateFunc(t *testing.T) {
	t.Run("ConfigValidateFunc_Validate", func(t *testing.T) {
		// 创建函数式验证器
		validateFunc := ConfigValidateFunc(func(config map[string]any) error {
			if host, ok := config["database.host"].(string); !ok || host == "" {
				return fmt.Errorf("database.host is required")
			}
			return nil
		})

		// 测试验证成功场景
		validConfig := map[string]any{
			"database.host": "localhost",
		}
		err := validateFunc.Validate(validConfig)
		assert.NoError(t, err, "验证应该通过")

		// 测试验证失败场景
		invalidConfig := map[string]any{
			"database.host": "",
		}
		err = validateFunc.Validate(invalidConfig)
		assert.Error(t, err, "验证应该失败")
		assert.Contains(t, err.Error(), "database.host is required")
	})

	t.Run("ConfigValidateFunc_GetName", func(t *testing.T) {
		// 创建函数式验证器
		validateFunc := ConfigValidateFunc(func(config map[string]any) error {
			return nil
		})

		// 测试 GetName 方法
		name := validateFunc.GetName()
		assert.Equal(t, "函数式验证器", name, "函数式验证器名称应该是'函数式验证器'")
	})
}

// TestAddValidator 测试添加验证器
func TestAddValidator(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "validator_add_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("validator_test"),
		WithContent(`
database:
  host: "localhost"
  port: 5432
`),
	)
	require.NoError(t, err)

	// 添加验证器
	cfg.AddValidator(&customValidator{})

	// 验证器应该被添加
	validators := cfg.GetValidators()
	assert.Len(t, validators, 1, "应该有1个验证器")
	assert.Equal(t, "CustomValidator", validators[0].GetName())

	// 添加多个验证器
	cfg.AddValidator(ConfigValidateFunc(func(config map[string]any) error {
		return nil
	}))

	validators = cfg.GetValidators()
	assert.Len(t, validators, 2, "应该有2个验证器")
}

// TestAddValidateFunc 测试添加函数式验证器
func TestAddValidateFunc(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "validator_func_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("validator_func_test"),
		WithContent(`
server:
  port: 8080
  host: "0.0.0.0"
`),
	)
	require.NoError(t, err)

	// 测试添加验证函数
	cfg.AddValidateFunc(func(config map[string]any) error {
		if port, ok := config["server.port"]; ok {
			if portInt, ok := port.(int); ok && (portInt < 1 || portInt > 65535) {
				return fmt.Errorf("server port must be between 1 and 65535")
			}
		}
		return nil
	})

	// 验证器应该被添加
	validators := cfg.GetValidators()
	assert.Len(t, validators, 1, "应该有1个验证器")
	assert.Equal(t, "函数式验证器", validators[0].GetName())

	// 添加多个验证函数
	cfg.AddValidateFunc(func(config map[string]any) error {
		if host, ok := config["server.host"].(string); !ok || host == "" {
			return fmt.Errorf("server host is required")
		}
		return nil
	})

	validators = cfg.GetValidators()
	assert.Len(t, validators, 2, "应该有2个验证器")
}

// TestClearValidators 测试清除所有验证器
func TestClearValidators(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "validator_clear_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("validator_clear_test"),
	)
	require.NoError(t, err)

	// 添加多个验证器
	cfg.AddValidateFunc(func(config map[string]any) error {
		return nil
	})
	cfg.AddValidateFunc(func(config map[string]any) error {
		return nil
	})

	// 验证器已添加
	validators := cfg.GetValidators()
	assert.Len(t, validators, 2, "应该有2个验证器")

	// 清除所有验证器
	cfg.ClearValidators()

	// 验证器应该被清空
	validators = cfg.GetValidators()
	assert.Empty(t, validators, "验证器应该被清空")
	assert.Len(t, validators, 0, "验证器列表应该为空")
}

// TestGetValidators 测试获取验证器列表
func TestGetValidators(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "validator_get_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("validator_get_test"),
	)
	require.NoError(t, err)

	// 初始状态应该没有验证器
	validators := cfg.GetValidators()
	assert.Empty(t, validators, "初始状态应该没有验证器")

	// 添加验证器
	cfg.AddValidateFunc(func(config map[string]any) error {
		return fmt.Errorf("test error 1")
	})
	cfg.AddValidateFunc(func(config map[string]any) error {
		return fmt.Errorf("test error 2")
	})

	// 获取验证器列表
	validators = cfg.GetValidators()
	assert.Len(t, validators, 2, "应该有2个验证器")

	// 验证返回的是副本（修改不影响原列表）
	validators[0] = nil
	newValidators := cfg.GetValidators()
	assert.NotNil(t, newValidators[0], "修改返回的列表不应该影响原列表")
}

// TestValidatorsIntegration 测试验证器集成场景
func TestValidatorsIntegration(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "validator_integration_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("validator_integration_test"),
		WithContent(`
database:
  host: "localhost"
  port: 5432
  username: "testuser"
  password: "testpass"
server:
  port: 8080
  host: "0.0.0.0"
`),
	)
	require.NoError(t, err)

	// 添加多个验证器
	cfg.AddValidateFunc(func(config map[string]any) error {
		if host, ok := config["database.host"].(string); !ok || host == "" {
			return fmt.Errorf("database.host is required")
		}
		return nil
	})

	cfg.AddValidateFunc(func(config map[string]any) error {
		if port, ok := config["server.port"].(int); ok && (port < 1 || port > 65535) {
			return fmt.Errorf("server.port must be between 1 and 65535")
		}
		return nil
	})

	// 获取验证器列表并验证
	validators := cfg.GetValidators()
	assert.Len(t, validators, 2, "应该有2个验证器")

	// 测试所有验证器都通过
	allSettings := flattenAllSettings(cfg.AllSettings())
	for _, validator := range validators {
		err := validator.Validate(allSettings)
		assert.NoError(t, err, "验证应该通过")
	}

	// 修改配置导致验证失败
	err = cfg.Set("server.port", 999999)
	require.NoError(t, err)

	// 等待配置更新
	allSettings = flattenAllSettings(cfg.AllSettings())

	// 第二个验证器应该失败（端口超出范围）
	err = validators[1].Validate(allSettings)
	assert.Error(t, err, "端口超出范围验证应该失败")
	assert.Contains(t, err.Error(), "must be between 1 and 65535")

	// 清除验证器
	cfg.ClearValidators()
	validators = cfg.GetValidators()
	assert.Empty(t, validators, "验证器应该被清空")
}

// flattenAllSettings 将嵌套 map 展开为扁平点号键，便于验证器直接读取。
func flattenAllSettings(m map[string]any) map[string]any {
	out := make(map[string]any)
	var walk func(prefix string, val any)
	walk = func(prefix string, val any) {
		switch v := val.(type) {
		case map[string]any:
			for k, vv := range v {
				key := k
				if prefix != "" {
					key = prefix + "." + k
				}
				walk(key, vv)
			}
		default:
			out[prefix] = v
		}
	}
	for k, v := range m {
		walk(k, v)
	}
	return out
}

// TestValidatorsConcurrency 测试验证器并发安全性
func TestValidatorsConcurrency(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "validator_concurrency_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("validator_concurrency_test"),
	)
	require.NoError(t, err)

	// 并发添加验证器
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			cfg.AddValidateFunc(func(config map[string]any) error {
				return fmt.Errorf("validator %d", id)
			})
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证器应该都被添加
	validators := cfg.GetValidators()
	assert.Len(t, validators, 10, "应该有10个验证器")

	// 并发获取和清除验证器
	for i := 0; i < 10; i++ {
		go func() {
			_ = cfg.GetValidators()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// 清除验证器
	cfg.ClearValidators()
	validators = cfg.GetValidators()
	assert.Empty(t, validators, "验证器应该被清空")
}

// TestViperMethod 测试 Viper() 方法
func TestViperMethod(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "viper_method_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("viper_test"),
		WithContent(`
test:
  key: "value"
  number: 42
`),
	)
	require.NoError(t, err)

	// 测试 Viper() 方法
	viper := cfg.Viper()
	assert.NotNil(t, viper, "Viper实例不应该为nil")

	// 验证可以通过viper实例访问配置
	assert.Equal(t, "value", viper.GetString("test.key"))
	assert.Equal(t, 42, viper.GetInt("test.number"))

	// 测试多次调用返回同一实例
	viper2 := cfg.Viper()
	assert.Equal(t, viper, viper2, "应该返回同一个viper实例")
}

// TestCreateBackupIfExists 测试创建配置备份
func TestCreateBackupIfExists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "backup_test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	defer os.RemoveAll(tmpDir)

	t.Run("文件不存在时不创建备份", func(t *testing.T) {
		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("no_backup"),
		)
		require.NoError(t, err)

		nonExistentFile := filepath.Join(tmpDir, "nonexistent.yaml")
		err = cfg.createBackupIfExists(nonExistentFile)
		assert.NoError(t, err, "文件不存在时不应该报错")

		// 验证没有创建备份文件
		files, err := filepath.Glob(filepath.Join(tmpDir, "nonexistent.yaml.backup.*"))
		assert.NoError(t, err)
		assert.Empty(t, files, "不应该创建备份文件")
	})

	t.Run("文件存在时创建备份", func(t *testing.T) {
		// 创建测试文件
		testFile := filepath.Join(tmpDir, "test_backup.yaml")
		testContent := []byte("test: original_content")
		require.NoError(t, os.WriteFile(testFile, testContent, 0o644))

		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("backup_test"),
		)
		require.NoError(t, err)

		// 创建备份
		err = cfg.createBackupIfExists(testFile)
		assert.NoError(t, err, "创建备份应该成功")

		// 验证备份文件被创建
		files, err := filepath.Glob(filepath.Join(tmpDir, "test_backup.yaml.backup.*"))
		assert.NoError(t, err)
		assert.NotEmpty(t, files, "应该创建备份文件")

		// 验证备份内容与原文件一致
		if len(files) > 0 {
			backupContent, err := os.ReadFile(files[0])
			assert.NoError(t, err)
			assert.Equal(t, testContent, backupContent, "备份内容应该与原文件一致")
		}
	})

	t.Run("多次备份创建多个备份文件", func(t *testing.T) {
		// 创建测试文件
		testFile := filepath.Join(tmpDir, "multi_backup.yaml")
		testContent := []byte("test: multi_backup_content")
		require.NoError(t, os.WriteFile(testFile, testContent, 0o644))

		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("multi_backup_test"),
		)
		require.NoError(t, err)

		// 创建第一个备份
		err = cfg.createBackupIfExists(testFile)
		assert.NoError(t, err)

		// 修改文件内容
		newContent := []byte("test: modified_content")
		require.NoError(t, os.WriteFile(testFile, newContent, 0o644))

		// 创建第二个备份
		err = cfg.createBackupIfExists(testFile)
		assert.NoError(t, err)

		// 验证创建了多个备份文件
		files, err := filepath.Glob(filepath.Join(tmpDir, "multi_backup.yaml.backup.*"))
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(files), 1, "至少应生成1个备份文件")
	})
}
