package sysconf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loadApp struct {
	Name string `config:"name" default:"demo"`
	Port int    `config:"port" default:"8080"`
}

func TestLoad(t *testing.T) {
	app, cfg, err := Load[loadApp](
		WithContent("name: prod\nport: 9090\n"),
		WithMode("yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "prod", app.Name)
	assert.Equal(t, 9090, app.Port)
}

func TestLoadUsesStructDefaults(t *testing.T) {
	app, cfg, err := Load[loadApp](WithContent("{}\n"), WithMode("yaml"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "demo", app.Name)
	assert.Equal(t, 8080, app.Port)
}

func TestLoadInvalidOption(t *testing.T) {
	_, cfg, err := Load[loadApp](WithMode("invalid"))
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestMustLoadAndMustNew(t *testing.T) {
	app, cfg := MustLoad[loadApp](
		WithContent("name: must\nport: 1\n"),
		WithMode("yaml"),
	)
	t.Cleanup(func() { _ = cfg.Close() })
	assert.Equal(t, "must", app.Name)

	cfg2 := MustNew(WithContent("k: v\n"), WithMode("yaml"))
	t.Cleanup(func() { _ = cfg2.Close() })
	assert.Equal(t, "v", cfg2.GetString("k"))

	assert.Panics(t, func() {
		_ = MustNew(WithMode("invalid"))
	})
	assert.Panics(t, func() {
		_, _ = MustLoad[loadApp](WithMode("invalid"))
	})
}

func TestUnmarshalAsAndMustUnmarshal(t *testing.T) {
	cfg := MustNew(
		WithContent("name: as\nport: 7\nserver:\n  timeout: 2s\n"),
		WithMode("yaml"),
	)
	t.Cleanup(func() { _ = cfg.Close() })

	app, err := UnmarshalAs[loadApp](cfg)
	require.NoError(t, err)
	assert.Equal(t, "as", app.Name)
	assert.Equal(t, 7, app.Port)

	type serverCfg struct {
		Timeout time.Duration `config:"timeout"`
	}
	server, err := UnmarshalAs[serverCfg](cfg, "server")
	require.NoError(t, err)
	assert.Equal(t, 2*time.Second, server.Timeout)

	_, err = UnmarshalAs[loadApp](nil)
	assert.ErrorIs(t, err, ErrNilConfig)

	var out loadApp
	cfg.MustUnmarshal(&out)
	assert.Equal(t, "as", out.Name)

	assert.Panics(t, func() {
		cfg.MustUnmarshal(out)
	})
}

func TestWithEnvEmptyPrefixDoesNotEnable(t *testing.T) {
	t.Setenv("PATH", "/should/not/override")

	cfg, err := New(WithEnv(""))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.False(t, cfg.envOptions.Enabled)
	assert.NotEqual(t, "/should/not/override", cfg.GetString("path"))
}

func TestUnmarshalSeesEnvOverride(t *testing.T) {
	t.Setenv("UNMARSHAL_DATABASE_PASSWORD", "from-env")

	cfg := MustNew(
		WithContent("database:\n  host: localhost\n  password: file-pass\n"),
		WithMode("yaml"),
		WithEnv("UNMARSHAL"),
	)
	t.Cleanup(func() { _ = cfg.Close() })

	type db struct {
		Host     string `config:"host"`
		Password string `config:"password"`
	}
	type app struct {
		Database db `config:"database"`
	}

	got, err := UnmarshalAs[app](cfg)
	require.NoError(t, err)
	assert.Equal(t, "from-env", got.Database.Password)
	assert.Equal(t, "from-env", cfg.GetString("database.password"))
	assert.Equal(t, "from-env", cfg.AllSettings()["database"].(map[string]any)["password"])
}

func TestWatchReturnsClosedError(t *testing.T) {
	cfg := MustNew()
	require.NoError(t, cfg.Close())
	assert.ErrorIs(t, cfg.Watch(func() {}), ErrAlreadyClosed)
}
