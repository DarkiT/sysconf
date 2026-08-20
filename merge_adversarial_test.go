package sysconf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlushRetriesAfterFailedWrite(t *testing.T) {
	dir := t.TempDir()
	cfg, err := New(
		WithPath(dir),
		WithName("retry_write"),
		WithMode("yaml"),
		WithContent("k: v\n"),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	require.NoError(t, cfg.Set("k", "new"))

	target := filepath.Join(dir, "retry_write.yaml")
	require.NoError(t, os.Remove(target))
	require.NoError(t, os.Mkdir(target, 0o755))

	require.Error(t, cfg.Flush(), "first flush should fail while path is a directory")

	require.NoError(t, os.Remove(target))
	require.NoError(t, cfg.Flush(), "second flush must retry the pending write")

	raw, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "new")
	assert.Equal(t, "new", cfg.GetString("k"))
}

func TestAllSettingsIncludesEnvOnlyKey(t *testing.T) {
	t.Setenv("ADVONLY_TOKEN", "secret-token")

	cfg, err := New(
		WithContent("host: localhost\n"),
		WithMode("yaml"),
		WithEnv("ADVONLY"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "secret-token", cfg.GetString("token"))
	settings := cfg.AllSettings()
	assert.Equal(t, "secret-token", settings["token"], "AllSettings must see env-only keys that Get sees")
	assert.Contains(t, cfg.Keys(), "token")
}

func TestAllSettingsIncludesEnvOnlyKeyWithoutSmartCase(t *testing.T) {
	t.Setenv("STRICTONLY_TOKEN", "strict-token")

	cfg, err := New(
		WithContent("host: localhost\n"),
		WithMode("yaml"),
		WithEnvSmartCase("STRICTONLY", false),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "strict-token", cfg.GetString("token"))
	assert.Equal(t, "strict-token", cfg.AllSettings()["token"])
}

func TestGetParentIncludesEnvChild(t *testing.T) {
	t.Setenv("PAR_DATABASE_PASSWORD", "from-env")

	cfg, err := New(
		WithContent("database:\n  host: h\n  password: file\n"),
		WithMode("yaml"),
		WithEnv("PAR"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	parent, ok := cfg.Get("database").(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "from-env", parent["password"])
	assert.Equal(t, "h", parent["host"])
}

func TestLookupDoesNotAliasSnakeToDottedNested(t *testing.T) {
	cfg, err := New(
		WithContent("max:\n  conns: 9\n"),
		WithMode("yaml"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, 9, cfg.GetInt("max.conns"))
	assert.Zero(t, cfg.GetInt("max_conns"), "lowercase snake_case must not silently alias dotted nested keys")
}

func TestOnChangeHandlerCanSet(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "onchange.yaml")
	require.NoError(t, os.WriteFile(file, []byte("port: 1\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("onchange"),
		WithMode("yaml"),
		WithWatchDebounce(20*time.Millisecond),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	done := make(chan struct{}, 1)
	require.NoError(t, cfg.OnChange(func(_, _ map[string]any, err error) {
		if err != nil {
			return
		}
		_ = cfg.Set("touched", true)
		done <- struct{}{}
	}))

	require.NoError(t, os.WriteFile(file, []byte("port: 2\n"), 0o644))
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("OnChange deadlocked or did not fire when handler called Set")
	}
	assert.Equal(t, 2, cfg.GetInt("port"))
	assert.Equal(t, true, cfg.GetBool("touched"))
}

func TestDotenvUppercaseStillResolves(t *testing.T) {
	cfg, err := New(
		WithContent("DATABASE_HOST=localhost\nDATABASE_PORT=5432\n"),
		WithMode("dotenv"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "localhost", cfg.GetString("DATABASE_HOST"))
	assert.Equal(t, 5432, cfg.GetInt("DATABASE_PORT"))
}

func TestMarshalDoesNotPinUnrelatedKeys(t *testing.T) {
	t.Setenv("PIN_PASSWORD", "from-env")
	dir := t.TempDir()
	file := filepath.Join(dir, "pin.yaml")
	require.NoError(t, os.WriteFile(file, []byte("port: 1\npassword: file\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("pin"),
		WithMode("yaml"),
		WithEnv("PIN"),
		WithWatchDebounce(20*time.Millisecond),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	require.NoError(t, cfg.Marshal(struct {
		Foo string `config:"foo"`
	}{Foo: "bar"}))
	assert.Equal(t, "bar", cfg.GetString("foo"))
	assert.Equal(t, "from-env", cfg.GetString("password"))

	changed := make(chan struct{}, 1)
	require.NoError(t, cfg.Watch(func() { changed <- struct{}{} }))
	require.NoError(t, os.WriteFile(file, []byte("port: 9\npassword: file\n"), 0o644))
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("Watch did not fire after Marshal of unrelated key")
	}
	assert.Equal(t, 9, cfg.GetInt("port"))
	assert.Equal(t, "from-env", cfg.GetString("password"))
}

func TestFlagSnapshotSeesParseAfterNew(t *testing.T) {
	flags := pflag.NewFlagSet("late", pflag.ContinueOnError)
	flags.String("host", "flag-default", "host")

	cfg, err := New(
		WithContent("host: file\n"),
		WithMode("yaml"),
		WithPFlagOptions(PFlagOptions{FlagSets: []*pflag.FlagSet{flags}, OnlyChanged: true}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "file", cfg.GetString("host"))
	require.NoError(t, flags.Parse([]string{"--host=from-flag"}))
	assert.Equal(t, "from-flag", cfg.GetString("host"))
}

func TestWatchAppliesValidRepairAfterInvalid(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "repair.yaml")
	require.NoError(t, os.WriteFile(file, []byte("port: 8080\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("repair"),
		WithMode("yaml"),
		WithWatchDebounce(200*time.Millisecond),
		WithValidateFunc(func(m map[string]any) error {
			return requirePortAtLeast(m, 1024)
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	got := make(chan error, 2)
	require.NoError(t, cfg.OnChange(func(_, _ map[string]any, err error) {
		got <- err
	}))

	require.NoError(t, os.WriteFile(file, []byte("port: 80\n"), 0o644))
	select {
	case err := <-got:
		require.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("invalid reload did not notify")
	}

	require.NoError(t, os.WriteFile(file, []byte("port: 9090\n"), 0o644))
	select {
	case err := <-got:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("valid repair was swallowed by watch debounce")
	}
	assert.Equal(t, 9090, cfg.GetInt("port"))
}

func TestViperMatchesGetAfterInvalidWatch(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "viperdiv.yaml")
	require.NoError(t, os.WriteFile(file, []byte("port: 8080\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("viperdiv"),
		WithMode("yaml"),
		WithWatchDebounce(20*time.Millisecond),
		WithValidateFunc(func(m map[string]any) error {
			return requirePortAtLeast(m, 1024)
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	notified := make(chan struct{}, 1)
	require.NoError(t, cfg.OnChange(func(_, _ map[string]any, err error) {
		if err != nil {
			notified <- struct{}{}
		}
	}))
	require.NoError(t, os.WriteFile(file, []byte("port: 80\n"), 0o644))
	select {
	case <-notified:
	case <-time.After(3 * time.Second):
		t.Fatal("invalid reload did not notify")
	}

	assert.Equal(t, 8080, cfg.GetInt("port"))
	assert.Equal(t, 8080, cfg.Viper().GetInt("port"))
}

func TestGetParentIncludesEnvOnlyChild(t *testing.T) {
	t.Setenv("PARONLY_DATABASE_TOKEN", "env-only")

	cfg, err := New(
		WithContent("database:\n  host: h\n"),
		WithMode("yaml"),
		WithEnv("PARONLY"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	parent, ok := cfg.Get("database").(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "h", parent["host"])
	assert.Equal(t, "env-only", parent["token"])
}

func TestCloseRetriesAfterFailedFlush(t *testing.T) {
	dir := t.TempDir()
	cfg, err := New(
		WithPath(dir),
		WithName("close_retry"),
		WithMode("yaml"),
		WithContent("k: v\n"),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)

	require.NoError(t, cfg.Set("k", "new"))

	target := filepath.Join(dir, "close_retry.yaml")
	require.NoError(t, os.Remove(target))
	require.NoError(t, os.Mkdir(target, 0o755))

	require.Error(t, cfg.Flush(), "first flush should fail while path is a directory")
	require.Error(t, cfg.Close(), "close must fail while pending write cannot persist")

	require.NoError(t, os.Remove(target))
	require.NoError(t, cfg.Close(), "close must retry the pending write after the path is restored")

	raw, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "new")
}

func TestInitValidationSeesEnvOnlyRequired(t *testing.T) {
	t.Setenv("VALONLY_TOKEN", "ok")

	cfg, err := New(
		WithContent("host: localhost\n"),
		WithMode("yaml"),
		WithEnv("VALONLY"),
		WithValidateFunc(func(m map[string]any) error {
			if m["token"] == nil || m["token"] == "" {
				return errors.New("token required")
			}
			return nil
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })
	assert.Equal(t, "ok", cfg.GetString("token"))
}

func TestInitValidationRejectsMissingEnvOnlyRequired(t *testing.T) {
	_, err := New(
		WithContent("host: localhost\n"),
		WithMode("yaml"),
		WithEnv("VALMISS"),
		WithValidateFunc(func(m map[string]any) error {
			if m["token"] == nil || m["token"] == "" {
				return errors.New("token required")
			}
			return nil
		}),
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestUnmarshalMapIncludesEnvOnlyKey(t *testing.T) {
	t.Setenv("UNMAP_TOKEN", "secret")

	cfg, err := New(
		WithContent("host: localhost\n"),
		WithMode("yaml"),
		WithEnv("UNMAP"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	var out map[string]any
	require.NoError(t, cfg.Unmarshal(&out))
	assert.Equal(t, "secret", out["token"])
	assert.Equal(t, "localhost", out["host"])
}

func TestFlagSetAfterNewIsLive(t *testing.T) {
	flags := pflag.NewFlagSet("setlate", pflag.ContinueOnError)
	flags.String("host", "flag-default", "host")

	cfg, err := New(
		WithContent("host: file\n"),
		WithMode("yaml"),
		WithPFlagOptions(PFlagOptions{FlagSets: []*pflag.FlagSet{flags}, OnlyChanged: true}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "file", cfg.GetString("host"))
	require.NoError(t, flags.Set("host", "from-set"))
	assert.Equal(t, "from-set", cfg.GetString("host"))
}

func TestWatchAppliesOverlayFileChange(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	over := filepath.Join(dir, "over.yaml")
	require.NoError(t, os.WriteFile(base, []byte("port: 1\nname: base\n"), 0o644))
	require.NoError(t, os.WriteFile(over, []byte("port: 2\n"), 0o644))

	cfg, err := New(
		WithFiles(base, over),
		WithWatchDebounce(20*time.Millisecond),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })
	assert.Equal(t, 2, cfg.GetInt("port"))

	done := make(chan struct{}, 1)
	require.NoError(t, cfg.OnChange(func(_, _ map[string]any, err error) {
		if err != nil {
			return
		}
		done <- struct{}{}
	}))

	require.NoError(t, os.WriteFile(over, []byte("port: 3\n"), 0o644))
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("overlay file change was not watched")
	}
	assert.Equal(t, 3, cfg.GetInt("port"))
	assert.Equal(t, "base", cfg.GetString("name"))
}

func TestSetVisibleThenClosePersists(t *testing.T) {
	for range 80 {
		dir := t.TempDir()
		cfg, err := New(
			WithPath(dir),
			WithName("visclose"),
			WithMode("yaml"),
			WithContent("k: before\n"),
			WithWriteDebounceDelay(time.Hour),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Flush())

		started := make(chan struct{})
		setDone := make(chan error, 1)
		go func() {
			close(started)
			setDone <- cfg.Set("k", "during-close")
		}()
		<-started
		seen := false
		for range 1_000_000 {
			if cfg.GetString("k") == "during-close" {
				seen = true
				break
			}
		}
		require.NoError(t, cfg.Close())
		<-setDone
		if !seen {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, "visclose.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(raw), "during-close", "Get already saw the Set; Close must persist it")
	}
}

func TestConcurrentSetDuringClosePersistsOrRejects(t *testing.T) {
	for range 30 {
		dir := t.TempDir()
		cfg, err := New(
			WithPath(dir),
			WithName("close_race"),
			WithMode("yaml"),
			WithContent("k: init\n"),
			WithWriteDebounceDelay(time.Hour),
		)
		require.NoError(t, err)

		require.NoError(t, cfg.Set("k", "before"))
		require.NoError(t, cfg.Flush())

		var setErr error
		var wg sync.WaitGroup
		wg.Go(func() {
			setErr = cfg.Set("k", "during-close")
		})
		closeErr := cfg.Close()
		wg.Wait()

		target := filepath.Join(dir, "close_race.yaml")
		raw, err := os.ReadFile(target)
		require.NoError(t, err)
		content := string(raw)

		if setErr == nil {
			require.NoError(t, closeErr)
			assert.Contains(t, content, "during-close", "successful Set must be on disk after Close")
			continue
		}
		assert.ErrorIs(t, setErr, ErrAlreadyClosed)
		assert.True(t, strings.Contains(content, "before") || strings.Contains(content, "during-close"))
	}
}

func TestViperMatchesGetAfterInvalidThenValidRepair(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "repairviper.yaml")
	require.NoError(t, os.WriteFile(file, []byte("port: 8080\n"), 0o644))

	cfg, err := New(
		WithPath(dir),
		WithName("repairviper"),
		WithMode("yaml"),
		WithWatchDebounce(20*time.Millisecond),
		WithValidateFunc(func(m map[string]any) error {
			return requirePortAtLeast(m, 1024)
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	got := make(chan error, 2)
	require.NoError(t, cfg.OnChange(func(_, _ map[string]any, err error) {
		got <- err
	}))

	require.NoError(t, os.WriteFile(file, []byte("port: 80\n"), 0o644))
	select {
	case err := <-got:
		require.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("invalid reload did not notify")
	}

	require.NoError(t, os.WriteFile(file, []byte("port: 9090\n"), 0o644))
	select {
	case err := <-got:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("valid repair was swallowed")
	}

	assert.Equal(t, 9090, cfg.GetInt("port"))
	assert.Equal(t, 9090, cfg.Viper().GetInt("port"))
}

func TestOverlayRemovalViperMatchesGet(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	over := filepath.Join(dir, "over.yaml")
	require.NoError(t, os.WriteFile(base, []byte("port: 1\nname: base\n"), 0o644))
	require.NoError(t, os.WriteFile(over, []byte("port: 2\nextra: yes\n"), 0o644))

	cfg, err := New(
		WithFiles(base, over),
		WithWatchDebounce(20*time.Millisecond),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })
	assert.Equal(t, 2, cfg.GetInt("port"))

	done := make(chan struct{}, 1)
	require.NoError(t, cfg.OnChange(func(_, _ map[string]any, err error) {
		if err != nil {
			return
		}
		done <- struct{}{}
	}))

	require.NoError(t, os.WriteFile(over, []byte("name: over\n"), 0o644))
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("overlay rewrite was not watched")
	}

	assert.Equal(t, 1, cfg.GetInt("port"))
	assert.Nil(t, cfg.Get("extra"))
	assert.Equal(t, 1, cfg.Viper().GetInt("port"))
	assert.False(t, cfg.Viper().IsSet("extra"))
}

func TestLiveFlagVisibleToGetAndViper(t *testing.T) {
	flags := pflag.NewFlagSet("liveviper", pflag.ContinueOnError)
	flags.String("host", "flag-default", "host")

	cfg, err := New(
		WithContent("host: file\n"),
		WithMode("yaml"),
		WithPFlagOptions(PFlagOptions{FlagSets: []*pflag.FlagSet{flags}, OnlyChanged: true}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.Equal(t, "file", cfg.GetString("host"))
	assert.Equal(t, "file", cfg.Viper().GetString("host"))

	require.NoError(t, flags.Parse([]string{"--host=from-flag"}))
	assert.Equal(t, "from-flag", cfg.GetString("host"))
	assert.Equal(t, "from-flag", cfg.Viper().GetString("host"))
}

func TestCloseDoesNotDeadlockWithFlush(t *testing.T) {
	for i := range 40 {
		dir := t.TempDir()
		cfg, err := New(
			WithPath(dir),
			WithName("close_flush"),
			WithMode("yaml"),
			WithContent("k: v\n"),
			WithWriteDebounceDelay(time.Hour),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Set("k", "new"))

		done := make(chan struct{})
		go func() {
			_ = cfg.Flush()
			close(done)
		}()
		closeErr := cfg.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("Close and Flush deadlocked on iteration %d", i)
		}
		if closeErr != nil {
			assert.ErrorIs(t, closeErr, ErrAlreadyClosed)
		}
	}
}

func TestCloseAfterWriteTimerDoesNotSurfaceAlreadyClosed(t *testing.T) {
	dir := t.TempDir()
	cfg, err := New(
		WithPath(dir),
		WithName("timer_close"),
		WithMode("yaml"),
		WithContent("k: v\n"),
		WithWriteDebounceDelay(8*time.Millisecond),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Set("k", "new"))
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, cfg.Close())
}

func TestTomlOverlayAndWatch(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.toml")
	over := filepath.Join(dir, "over.toml")
	require.NoError(t, os.WriteFile(base, []byte("port = 1\nname = \"base\"\n"), 0o644))
	require.NoError(t, os.WriteFile(over, []byte("port = 2\n"), 0o644))

	cfg, err := New(
		WithFiles(base, over),
		WithWatchDebounce(20*time.Millisecond),
		WithWriteDebounceDelay(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })
	assert.Equal(t, 2, cfg.GetInt("port"))
	assert.Equal(t, 2, cfg.Viper().GetInt("port"))

	done := make(chan error, 1)
	require.NoError(t, cfg.OnChange(func(_, _ map[string]any, err error) {
		done <- err
	}))
	require.NoError(t, os.WriteFile(over, []byte("port = 3\n"), 0o644))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("toml overlay watch did not fire")
	}
	assert.Equal(t, 3, cfg.GetInt("port"))
	assert.Equal(t, 3, cfg.Viper().GetInt("port"))
}

func TestEmptyEnvOnlyWithoutSmartCase(t *testing.T) {
	t.Setenv("EMPTYSTRICT_TOKEN", "")

	cfg, err := New(
		WithContent("host: localhost\n"),
		WithMode("yaml"),
		WithEnvSmartCase("EMPTYSTRICT", false),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	assert.True(t, cfg.IsSet("token"))
	assert.Equal(t, "", cfg.GetString("token"))
	assert.Equal(t, "", cfg.AllSettings()["token"])
	assert.Contains(t, cfg.Keys(), "token")

	var out map[string]any
	require.NoError(t, cfg.Unmarshal(&out))
	assert.Equal(t, "", out["token"])
}

func requirePortAtLeast(m map[string]any, min int) error {
	var n int
	switch v := m["port"].(type) {
	case int:
		n = v
	case int64:
		n = int(v)
	case uint64:
		n = int(v)
	case float64:
		n = int(v)
	default:
		return errors.New("port missing")
	}
	if n < min {
		return errors.New("port too small")
	}
	return nil
}
