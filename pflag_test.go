package sysconf

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// TestPFlagIntegration 测试pflag集成是否正常工作
func TestPFlagIntegration(t *testing.T) {
	// 创建pflag.FlagSet（模拟cobra命令）
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// 定义一些标志（类似cobra.Command.Flags()）
	var (
		host  = flags.String("host", "localhost", "Database host")
		port  = flags.Int("port", 5432, "Database port")
		debug = flags.Bool("debug", false, "Enable debug mode")
	)

	// 模拟命令行参数解析
	args := []string{"--host=testhost.com", "--port=3306", "--debug"}
	err := flags.Parse(args)
	assert.NoError(t, err)

	// 创建配置实例，绑定pflag
	cfg, err := New(
		WithBindPFlags(flags),
		WithContent(`
app:
  name: "TestApp"
database:
  host: "localhost"
  port: 5432
  debug: false
`),
	)
	assert.NoError(t, err)

	// 验证命令行参数覆盖了配置文件值
	assert.Equal(t, "testhost.com", cfg.GetString("host"))
	assert.Equal(t, 3306, cfg.GetInt("port"))
	assert.True(t, cfg.GetBool("debug"))

	// 验证配置文件中其他值仍然可用
	assert.Equal(t, "TestApp", cfg.GetString("app.name"))

	// 验证我们可以访问原始flag值
	assert.Equal(t, "testhost.com", *host)
	assert.Equal(t, 3306, *port)
	assert.True(t, *debug)

	t.Logf("✅ pflag集成测试通过 - 命令行参数正确覆盖配置文件值")
}

// TestMultiplePFlagSets 测试多个FlagSet
func TestMultiplePFlagSets(t *testing.T) {
	// 模拟多个子命令的flag集合
	serverFlags := pflag.NewFlagSet("server", pflag.ContinueOnError)
	serverFlags.String("bind", "0.0.0.0", "Server bind address")

	dbFlags := pflag.NewFlagSet("database", pflag.ContinueOnError)
	dbFlags.String("dsn", "postgres://", "Database DSN")

	// 解析不同的参数
	serverFlags.Parse([]string{"--bind=127.0.0.1"})
	dbFlags.Parse([]string{"--dsn=mysql://"})

	cfg, err := New(
		WithBindPFlags(serverFlags, dbFlags),
		WithContent(`
app:
  name: "MultiPFlagApp"
`),
	)
	assert.NoError(t, err)

	// 验证来自不同FlagSet的参数都生效
	assert.Equal(t, "127.0.0.1", cfg.GetString("bind"))
	assert.Equal(t, "mysql://", cfg.GetString("dsn"))
	assert.Equal(t, "MultiPFlagApp", cfg.GetString("app.name"))

	t.Logf("✅ 多FlagSet集成测试通过")
}
