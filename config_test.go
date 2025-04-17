package sysconf

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// 测试配置结构体
type TestConf struct {
	Database struct {
		Host     string `config:"host" default:"localhost"`
		Port     int    `config:"port" default:"5432"`
		User     string `config:"user" required:"true"`
		Password string `config:"password" required:"true"`
		Timeout  int    `config:"timeout" default:"30"`
	} `config:"database"`
	Server struct {
		Port    int      `config:"port" default:"8080"`
		Host    string   `config:"host" default:"0.0.0.0"`
		Debug   bool     `config:"debug" default:"false"`
		Origins []string `config:"origins" default:"localhost,127.0.0.1"`
	} `config:"server"`
}

func TestConfig(t *testing.T) {
	// 创建临时测试目录
	tmpDir := filepath.Join(os.TempDir(), "config_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 测试配置内容
	const testConfig = `
database:
  host: "testdb.example.com"
  port: 5432
  user: "testuser"
  password: "testpass"
  timeout: 60
server:
  port: 9090
  host: "127.0.0.1"
  debug: true
  origins: ["localhost", "example.com"]
`

	// 创建配置实例
	cfg, err := New(
		WithPath(tmpDir),
		WithMode("yaml"),
		WithName("config"),
		WithContent(testConfig),
		WithEnvOptions(EnvOptions{
			Prefix:  "APP",
			Enabled: true,
		}),
	)
	if err != nil {
		t.Fatalf("创建配置实例失败: %v", err)
	}

	// 测试基本获取方法
	t.Run("基本获取方法", func(t *testing.T) {
		if host := cfg.GetString("database.host"); host != "testdb.example.com" {
			t.Errorf("GetString 失败, 期望 testdb.example.com, 获得 %s", host)
		}

		if port := cfg.GetInt("database.port"); port != 5432 {
			t.Errorf("GetInt 失败, 期望 5432, 获得 %d", port)
		}

		if debug := cfg.GetBool("server.debug"); !debug {
			t.Error("GetBool 失败, 期望 true")
		}

		origins := cfg.GetStringSlice("server.origins")
		if len(origins) != 2 || origins[0] != "localhost" || origins[1] != "example.com" {
			t.Errorf("GetStringSlice 失败, 获得 %v", origins)
		}
	})

	// 测试设置值
	t.Run("设置值", func(t *testing.T) {
		if err := cfg.Set("database.host", "newhost.example.com"); err != nil {
			t.Errorf("Set 失败: %v", err)
		}

		// 等待一会以确保值被设置
		time.Sleep(100 * time.Millisecond)

		if host := cfg.GetString("database.host"); host != "newhost.example.com" {
			t.Errorf("设置后的值不正确, 期望 newhost.example.com, 获得 %s", host)
		}
	})

	// 测试结构体解析
	t.Run("结构体解析", func(t *testing.T) {
		var conf TestConf
		if err := cfg.Unmarshal(&conf); err != nil {
			t.Errorf("Unmarshal 失败: %v", err)
		}

		if conf.Database.Host != "newhost.example.com" {
			t.Errorf("解析后的 Host 不正确, 期望 newhost.example.com, 获得 %s", conf.Database.Host)
		}

		if conf.Server.Port != 9090 {
			t.Errorf("解析后的 Port 不正确, 期望 9090, 获得 %d", conf.Server.Port)
		}
	})

	// 测试配置监听
	t.Run("配置监听", func(t *testing.T) {
		changed := make(chan bool)
		cfg.Watch(func() {
			changed <- true
		})

		// 修改配置
		if err := cfg.Set("database.port", 5433); err != nil {
			t.Errorf("修改配置失败: %v", err)
		}

		// 等待配置变更通知
		select {
		case <-changed:
			// 成功接收到变更通知
		case <-time.After(5 * time.Second):
			t.Error("未收到配置变更通知")
		}
	})

	// 测试环境变量
	t.Run("环境变量", func(t *testing.T) {
		// t.Skip("环境变量测试可能依赖文件系统或特定环境，暂时跳过")

		os.Setenv("APP_DATABASE_HOST", "envhost.example.com")

		// 重新初始化以加载环境变量
		cfg.SetEnvPrefix("APP")

		if host := cfg.GetString("database.host"); host != "envhost.example.com" {
			t.Errorf("环境变量未生效, 期望 envhost.example.com, 获得 %s", host)
		}

		os.Unsetenv("APP_DATABASE_HOST")
	})
}

// 测试全局配置实例
func TestGlobalConfig(t *testing.T) {
	globalCfg := Default()
	if globalCfg == nil {
		t.Error("获取全局配置实例失败")
	}

	// 测试注册全局配置
	if err := Register("test", "key", "value"); err != nil {
		t.Errorf("注册全局配置失败: %v", err)
	}

	if val := globalCfg.GetString("test.key"); val != "value" {
		t.Errorf("获取注册的全局配置失败, 期望 value, 获得 %s", val)
	}
}

// 测试错误处理和边界条件
func TestConfigEdgeCases(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "config_edge_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("无效的配置模式", func(t *testing.T) {
		_, err := New(
			WithPath(tmpDir),
			WithMode("invalid"),
		)
		if err == nil {
			t.Error("期望获得错误,但是没有")
		}
	})

	t.Run("无效的配置路径", func(t *testing.T) {
		invalidPath := string([]byte{0x00}) // 包含非法字符的路径
		_, err := New(
			WithPath(invalidPath),
			WithMode("yaml"),
		)
		if err == nil {
			t.Error("期望获得路径错误,但是没有")
		}
	})

	// 测试必填字段验证
	t.Run("必填字段验证", func(t *testing.T) {
		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("required_test"),
			WithContent(`
required:
    required: "true"
    optional: "ssss"
`),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		type ConfigWithRequired struct {
			Required string `config:"required" required:"true"`
			Optional string `config:"optional"`
		}

		var conf ConfigWithRequired
		err = cfg.Unmarshal(&conf)
		if err == nil || !strings.Contains(err.Error(), "required") {
			t.Error("期望获得必填字段错误,但是没有")
		}
	})

	// 测试默认值处理
	t.Run("默认值处理", func(t *testing.T) {
		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("default_test"),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		type ConfigWithDefaults struct {
			String string   `config:"string" default:"default"`
			Int    int      `config:"int" default:"42"`
			Bool   bool     `config:"bool" default:"true"`
			Float  float64  `config:"float" default:"3.14"`
			Slice  []string `config:"slice" default:"a,b,c"`
		}

		var conf ConfigWithDefaults
		err = cfg.Unmarshal(&conf)
		if err != nil {
			t.Errorf("解析默认值失败: %v", err)
		}

		if conf.String != "default" {
			t.Errorf("字符串默认值错误,期望 'default',获得 %s", conf.String)
		}
		if conf.Int != 42 {
			t.Errorf("整数默认值错误,期望 42,获得 %d", conf.Int)
		}
		if !conf.Bool {
			t.Error("布尔默认值错误,期望 true")
		}
		if conf.Float != 3.14 {
			t.Errorf("浮点数默认值错误,期望 3.14,获得 %f", conf.Float)
		}
		if len(conf.Slice) != 3 || conf.Slice[0] != "a" {
			t.Errorf("切片默认值错误,获得 %v", conf.Slice)
		}
	})

	// 测试类型转换
	t.Run("类型转换", func(t *testing.T) {
		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("type_test"),
			WithContent(`
values:
    slice_string: [1,2,3]
    slice_int: [1,2,3]
    slice_float: [1.1,2.2,3.3]
`),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		// 测试字符串切片
		strSlice := cfg.GetStringSlice("values.slice_string")
		if len(strSlice) != 3 || strSlice[0] != "1" || strSlice[1] != "2" || strSlice[2] != "3" {
			t.Errorf("字符串切片转换失败,获得 %v", strSlice)
		}

		// 使用结构体测试
		type TestStruct struct {
			Values struct {
				SliceString []string  `config:"slice_string"`
				SliceInt    []int     `config:"slice_int"`
				SliceFloat  []float64 `config:"slice_float"`
			} `config:"values"`
		}

		var conf TestStruct
		if err := cfg.Unmarshal(&conf); err != nil {
			t.Errorf("解析结构体失败: %v", err)
		}

		// 验证各种类型的切片
		if len(conf.Values.SliceInt) != 3 || conf.Values.SliceInt[0] != 1 {
			t.Errorf("整数切片转换失败,获得 %v", conf.Values.SliceInt)
		}

		if len(conf.Values.SliceFloat) != 3 || conf.Values.SliceFloat[0] != 1.1 {
			t.Errorf("浮点数切片转换失败,获得 %v", conf.Values.SliceFloat)
		}
	})

	// 测试并发安全性
	t.Run("并发安全性", func(t *testing.T) {
		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("concurrent_test"),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		const goroutines = 10
		const iterations = 100
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				for j := 0; j < iterations; j++ {
					// 并发读写测试
					key := fmt.Sprintf("test.key.%d", id)
					if err := cfg.Set(key, j); err != nil {
						t.Errorf("并发写入失败: %v", err)
					}
					_ = cfg.GetInt(key)
				}
				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < goroutines; i++ {
			<-done
		}
	})

	// 测试配置重载
	t.Run("配置重载", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, "reload_test.yaml")
		initialContent := []byte("key: initial_value")
		if err := os.WriteFile(configFile, initialContent, 0o644); err != nil {
			t.Fatalf("写入初始配置失败: %v", err)
		}

		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("reload_test"),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		// 验证初始值
		if val := cfg.GetString("key"); val != "initial_value" {
			t.Errorf("初始值错误,期望 initial_value,获得 %s", val)
		}

		// 设置监听
		reloaded := make(chan bool)
		cfg.Watch(func() {
			reloaded <- true
		})

		// 修改配置文件
		newContent := []byte("key: updated_value")
		if err := os.WriteFile(configFile, newContent, 0o644); err != nil {
			t.Fatalf("更新配置文件失败: %v", err)
		}

		// 等待配置重载
		select {
		case <-reloaded:
			// 验证更新后的值
			if val := cfg.GetString("key"); val != "updated_value" {
				t.Errorf("更新后的值错误,期望 updated_value,获得 %s", val)
			}
		case <-time.After(5 * time.Second):
			t.Error("配置重载超时")
		}
	})
}

// 测试环境变量处理
func TestEnvironmentVariables(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "config_env_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("环境变量优先级", func(t *testing.T) {
		const testConfig = `
database:
  host: "config_host"
  port: 5432
`
		// 设置环境变量
		os.Setenv("TEST_DATABASE_HOST", "env_host")
		defer os.Unsetenv("TEST_DATABASE_HOST")

		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("env_test"),
			WithContent(testConfig),
			WithEnvOptions(EnvOptions{
				Prefix:  "TEST",
				Enabled: true,
			}),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		if host := cfg.GetString("database.host"); host != "env_host" {
			t.Errorf("环境变量优先级错误,期望 env_host,获得 %s", host)
		}
	})

	t.Run("环境变量前缀处理", func(t *testing.T) {
		os.Setenv("MY_APP_VALUE", "test_value")
		defer os.Unsetenv("MY_APP_VALUE")

		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("prefix_test"),
			WithEnvOptions(EnvOptions{
				Prefix:  "MY_APP",
				Enabled: true,
			}),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		if val := cfg.GetString("value"); val != "test_value" {
			t.Errorf("环境变量前缀处理错误,期望 test_value,获得 %s", val)
		}
	})
}

// 测试工具函数
func TestUtilityFunctions(t *testing.T) {
	t.Run("WorkPath函数", func(t *testing.T) {
		// 测试基本路径
		path := WorkPath()
		if path == "" {
			t.Error("WorkPath返回空路径")
		}

		// 测试带子路径
		subPath := WorkPath("config", "test")
		if !strings.HasSuffix(subPath, filepath.Join("config", "test")) {
			t.Errorf("WorkPath子路径拼接错误: %s", subPath)
		}
	})
}

// 测试复杂数据结构
func TestComplexDataStructures(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "config_complex_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	const complexConfig = `
nested:
  level1:
    level2:
      string: "nested value"
      number: 42
      bool: true
arrays:
  strings: ["a", "b", "c"]
  numbers: [1, 2, 3]
  mixed: ["a", 1, true]
maps:
  simple:
    key1: "value1"
    key2: "value2"
  complex:
    key1:
      nested: "value"
durations:
  short: "1s"
  long: "24h"
`

	t.Run("复杂嵌套结构", func(t *testing.T) {
		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("complex"),
			WithContent(complexConfig),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		// 测试嵌套结构
		type NestedConfig struct {
			Nested struct {
				Level1 struct {
					Level2 struct {
						String string `config:"string"`
						Number int    `config:"number"`
						Bool   bool   `config:"bool"`
					} `config:"level2"`
				} `config:"level1"`
			} `config:"nested"`
			Arrays struct {
				Strings []string      `config:"strings"`
				Numbers []int         `config:"numbers"`
				Mixed   []interface{} `config:"mixed"`
			} `config:"arrays"`
			Maps struct {
				Simple  map[string]string            `config:"simple"`
				Complex map[string]map[string]string `config:"complex"`
			} `config:"maps"`
			Durations struct {
				Short time.Duration `config:"short"`
				Long  time.Duration `config:"long"`
			} `config:"durations"`
		}

		var conf NestedConfig
		if err := cfg.Unmarshal(&conf); err != nil {
			t.Fatalf("解析复杂配置失败: %v", err)
		}

		// 验证嵌套值
		if conf.Nested.Level1.Level2.String != "nested value" {
			t.Error("嵌套字符串值错误")
		}

		// 验证数组
		if len(conf.Arrays.Strings) != 3 || conf.Arrays.Strings[0] != "a" {
			t.Error("字符串数组解析错误")
		}

		// 验证映射
		if v, ok := conf.Maps.Simple["key1"]; !ok || v != "value1" {
			t.Error("简单映射解析错误")
		}

		// 验证时间间隔
		if conf.Durations.Short != time.Second {
			t.Error("短时间间隔解析错误")
		}
		if conf.Durations.Long != 24*time.Hour {
			t.Error("长时间间隔解析错误")
		}
	})
}

// 测试配置文件格式
func TestConfigFormats(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "config_formats_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		format  string
		content string
	}{
		{
			format: "json",
			content: `{
				"database": {
					"host": "localhost",
					"port": 5432
				}
			}`,
		},
		{
			format: "yaml",
			content: `
database:
  host: localhost
  port: 5432
`,
		},
		{
			format: "toml",
			content: `
[database]
host = "localhost"
port = 5432
`,
		},
		{
			format: "dotenv",
			content: `
DATABASE_HOST=localhost
DATABASE_PORT=5432
`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("格式_%s", test.format), func(t *testing.T) {
			cfg, err := New(
				WithPath(tmpDir),
				WithMode(test.format),
				WithName(fmt.Sprintf("config_%s", test.format)),
				WithContent(test.content),
			)
			if err != nil {
				t.Fatalf("创建 %s 配置实例失败: %v", test.format, err)
			}

			// 验证配置读取
			var host, portPath string
			if test.format == "dotenv" {
				host = "DATABASE_HOST"
				portPath = "DATABASE_PORT"
			} else {
				host = "database.host"
				portPath = "database.port"
			}

			if hostVal := cfg.GetString(host); hostVal != "localhost" {
				t.Errorf("%s 格式 host 值错误,期望 localhost,获得 %s",
					test.format, hostVal)
			}
			if port := cfg.GetInt(portPath); port != 5432 {
				t.Errorf("%s 格式 port 值错误,期望 5432,获得 %d",
					test.format, port)
			}
		})
	}
}

// 测试配置写入和持久化
func TestConfigPersistence(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "config_persistence_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("配置持久化", func(t *testing.T) {
		cfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("persistence"),
		)
		if err != nil {
			t.Fatalf("创建配置实例失败: %v", err)
		}

		// 设置一些配置值
		testData := map[string]interface{}{
			"string": "value",
			"number": 42,
			"bool":   true,
			"nested": map[string]interface{}{
				"key": "value",
			},
		}

		for k, v := range testData {
			if err := cfg.Set(k, v); err != nil {
				t.Errorf("设置配置值失败 %s: %v", k, err)
			}
		}

		// 等待异步写入完成
		time.Sleep(4 * time.Second)

		// 创建新的配置实例读取持久化的值
		newCfg, err := New(
			WithPath(tmpDir),
			WithMode("yaml"),
			WithName("persistence"),
		)
		if err != nil {
			t.Fatalf("创建新配置实例失败: %v", err)
		}

		// 验证持久化的值
		for k, v := range testData {
			got := newCfg.Get(k)
			if !reflect.DeepEqual(got, v) {
				t.Errorf("持久化值不匹配 %s: 期望 %v, 获得 %v", k, v, got)
			}
		}
	})
}
