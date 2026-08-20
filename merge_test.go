package sysconf

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergePrecedenceSetBeatsFlagBeatsEnvBeatsFile(t *testing.T) {
	t.Setenv("LAYER_HOST", "from-env")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("host", "flag-default", "host")
	require.NoError(t, flags.Parse([]string{"--host=from-flag"}))

	cfg, err := New(
		WithContent("host: from-file\n"),
		WithMode("yaml"),
		WithEnv("LAYER"),
		WithBindPFlags(flags),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "from-flag", cfg.GetString("host"))
	assert.Equal(t, "from-flag", GetAs(cfg, "host", ""))

	require.NoError(t, cfg.Set("host", "from-set"))
	assert.Equal(t, "from-set", cfg.GetString("host"))
	assert.Equal(t, "from-set", cfg.AllSettings()["host"])
}

func TestWithFilesLaterFileWins(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	over := filepath.Join(dir, "over.yaml")
	require.NoError(t, os.WriteFile(base, []byte("host: base\nport: 1\n"), 0o644))
	require.NoError(t, os.WriteFile(over, []byte("port: 2\nextra: yes\n"), 0o644))

	cfg, err := New(WithFiles(base, over))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "base", cfg.GetString("host"))
	assert.Equal(t, 2, cfg.GetInt("port"))
	assert.Equal(t, "yes", cfg.GetString("extra"))
}

func TestLoadSeesEnvOnlyTaggedFields(t *testing.T) {
	t.Setenv("ENVONLY_TOKEN", "secret-token")

	type app struct {
		Host  string `config:"host"`
		Token string `config:"token"`
	}

	got, cfg, err := Load[app](
		WithContent("host: localhost\n"),
		WithMode("yaml"),
		WithEnv("ENVONLY"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "localhost", got.Host)
	assert.Equal(t, "secret-token", got.Token)
	assert.Equal(t, "secret-token", cfg.GetString("token"))
}

func TestInitValidationFailsNew(t *testing.T) {
	_, err := New(
		WithContent("port: 1\n"),
		WithMode("yaml"),
		WithValidateFunc(func(m map[string]any) error {
			return errors.New("port too small")
		}),
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestSkipInitValidationAllowsStart(t *testing.T) {
	cfg, err := New(
		WithContent("port: 1\n"),
		WithMode("yaml"),
		WithSkipInitValidation(),
		WithValidateFunc(func(m map[string]any) error {
			return errors.New("port too small")
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })
	assert.Equal(t, 1, cfg.GetInt("port"))
}

func TestSkipInitValidationAllowsWatchReload(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "skip_watch.yaml")
	require.NoError(t, os.WriteFile(file, []byte("port: 8080\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("skip_watch"),
		WithMode("yaml"),
		WithWatchDebounce(20*time.Millisecond),
		WithSkipInitValidation(),
		WithValidateFunc(func(map[string]any) error {
			return errors.New("document validator should be skipped")
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	changed := make(chan struct{}, 1)
	require.NoError(t, cfg.Watch(func() { changed <- struct{}{} }))

	require.NoError(t, os.WriteFile(file, []byte("port: 9090\n"), 0o644))
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("Watch should fire when document validation is skipped")
	}
	assert.Equal(t, 9090, cfg.GetInt("port"))
}

func TestFlushWritesPendingAndOmitsEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLUSHTEST_PASSWORD", "env-secret")

	cfg, err := New(
		WithPath(dir),
		WithName("flush_test"),
		WithMode("yaml"),
		WithContent("password: file-pass\nhost: localhost\n"),
		WithEnv("FLUSHTEST"),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "env-secret", cfg.GetString("password"))
	require.NoError(t, cfg.Flush())

	require.NoError(t, cfg.Set("host", "written"))
	require.NoError(t, cfg.Flush())

	raw, err := os.ReadFile(filepath.Join(dir, "flush_test.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "written")
	assert.Contains(t, string(raw), "file-pass")
	assert.NotContains(t, string(raw), "env-secret")
}

func TestFlushNoPendingIsNil(t *testing.T) {
	cfg, err := New(WithContent("k: v\n"), WithMode("yaml"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })
	assert.NoError(t, cfg.Flush())
}

func TestWatchInvalidReloadKeepsOldValues(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "watch_valid.yaml")
	require.NoError(t, os.WriteFile(file, []byte("port: 8080\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("watch_valid"),
		WithMode("yaml"),
		WithWatchDebounce(20*time.Millisecond),
		WithValidateFunc(func(m map[string]any) error {
			port, _ := m["port"]
			switch n := port.(type) {
			case int:
				if n < 1024 {
					return errors.New("port too small")
				}
			case int64:
				if n < 1024 {
					return errors.New("port too small")
				}
			case uint64:
				if n < 1024 {
					return errors.New("port too small")
				}
			case float64:
				if n < 1024 {
					return errors.New("port too small")
				}
			}
			return nil
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	gotErr := make(chan error, 1)
	require.NoError(t, cfg.OnChange(func(prev, next map[string]any, err error) {
		gotErr <- err
	}))

	require.Equal(t, 8080, cfg.GetInt("port"))
	require.NoError(t, os.WriteFile(file, []byte("port: 80\n"), 0o644))

	select {
	case err := <-gotErr:
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidation)
	case <-time.After(3 * time.Second):
		t.Fatal("OnChange did not fire after invalid reload")
	}

	assert.Equal(t, 8080, cfg.GetInt("port"))
}

func TestSetSurvivesWatchReload(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "watch_set.yaml")
	require.NoError(t, os.WriteFile(file, []byte("host: file\nport: 1\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("watch_set"),
		WithMode("yaml"),
		WithWatchDebounce(20*time.Millisecond),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	require.NoError(t, cfg.Set("host", "from-set"))

	changed := make(chan struct{}, 1)
	require.NoError(t, cfg.Watch(func() { changed <- struct{}{} }))

	require.NoError(t, os.WriteFile(file, []byte("host: reloaded\nport: 2\n"), 0o644))
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("Watch did not fire")
	}

	assert.Equal(t, "from-set", cfg.GetString("host"))
	assert.Equal(t, 2, cfg.GetInt("port"))
}

func TestSmartCaseMixedPrefixEnv(t *testing.T) {
	t.Setenv("MyApp_Cache_Timeout", "5m")
	t.Setenv("myapp_database_host", "lowercase.example.com")

	cfg, err := New(
		WithContent("cache:\n  timeout: 1m\ndatabase:\n  host: localhost\n"),
		WithMode("yaml"),
		WithEnv("MYAPP"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "5m", cfg.GetString("cache.timeout"))
	assert.Equal(t, "lowercase.example.com", cfg.GetString("database.host"))
}
