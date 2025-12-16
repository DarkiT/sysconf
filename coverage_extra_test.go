package sysconf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

// newTestConfig 创建带初始内容的配置，便于重复使用。
func newTestConfig(t *testing.T) *Config {
	t.Helper()
	cfg, err := New(
		WithPath(t.TempDir()),
		WithName("cov"),
		WithMode("yaml"),
		WithContent(`
root: "old"
database:
  host: "localhost"
  port: 5432
`),
	)
	if err != nil {
		t.Fatalf("create config failed: %v", err)
	}
	return cfg
}

func TestCacheLifecycleAndNestedLookup(t *testing.T) {
	cfg := newTestConfig(t)
	defer cfg.Close()

	// 预热缓存并验证命中
	cfg.cacheEnabled.Store(true)
	cfg.updateReadCache()

	if v, ok := cfg.getCachedValue("database.host"); !ok || v != "localhost" {
		t.Fatalf("expect cached host, got %v ok=%v", v, ok)
	}
	if v, ok := cfg.getCachedValue("database.port"); !ok || v != 5432 {
		t.Fatalf("expect cached port, got %v ok=%v", v, ok)
	}
	// 嵌套路径查找
	cacheMap := cfg.readCache.Load().(map[string]any)
	if nested := cfg.getNestedValue(cacheMap, "database.host"); nested != "localhost" {
		t.Fatalf("expect nested host lookup, got %v", nested)
	}

	// 禁用缓存后应返回未命中
	cfg.disableReadCache()
	if _, ok := cfg.getCachedValue("database.host"); ok {
		t.Fatalf("expect cache miss after disable")
	}

	// 缓存失效后调度重建（立即和延迟分支）
	cfg.cacheEnabled.Store(true)
	cfg.cacheRebuildDelay = 0
	cfg.invalidateCache() // 立即分支
	cfg.cacheRebuildDelay = 5 * time.Millisecond
	cfg.invalidateCache() // 延迟分支
}

func TestMarshalDeepMergeAndKeys(t *testing.T) {
	cfg := newTestConfig(t)
	defer cfg.Close()

	type App struct {
		Name  string `config:"name"`
		Title string `config:"title"`
	}
	payload := struct {
		App App `config:"app"`
	}{
		App: App{Name: "demo", Title: "hello"},
	}
	if err := cfg.Marshal(payload); err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	// 触发同步：Marshal 只写入 viper，需要同步回内部 map
	cfg.syncFromViperUnsafe()
	// Marshal 会将结构体与已有配置合并，确保新字段生效
	if got := cfg.GetString("app.name"); got != "demo" {
		t.Fatalf("expect merged app.name, got %s", got)
	}
	// 原有键仍存在
	if got := cfg.GetString("root"); got != "old" {
		t.Fatalf("expect original root preserved, got %s", got)
	}
	if len(cfg.Keys()) == 0 {
		t.Fatalf("Keys should return non-empty slice")
	}
}

func TestConfigValidatorApis(t *testing.T) {
	cfg := newTestConfig(t)
	defer cfg.Close()

	cfg.AddValidator(ConfigValidateFunc(func(config map[string]any) error { return nil }))
	cfg.AddValidateFunc(func(config map[string]any) error { return fmt.Errorf("boom") })

	validators := cfg.GetValidators()
	if len(validators) != 2 {
		t.Fatalf("expect 2 validators, got %d", len(validators))
	}
	// 验证返回副本
	validators[0] = nil
	if len(cfg.GetValidators()) != 2 {
		t.Fatalf("validators should be immutable copy")
	}

	cfg.ClearValidators()
	if len(cfg.GetValidators()) != 0 {
		t.Fatalf("validators should be cleared")
	}

	// 触发验证失败场景：直接调用获取到的函数式验证器
	cfg.AddValidateFunc(func(config map[string]any) error { return fmt.Errorf("fail") })
	if err := cfg.GetValidators()[0].Validate(cfg.AllSettings()); err == nil {
		t.Fatalf("Validate should surface validator errors")
	}
}

func TestLoggerNop(t *testing.T) {
	var l Logger = &NopLogger{}
	l.Debug("a")
	l.Debugf("%s", "b")
	l.Info("c")
	l.Infof("%s", "d")
	l.Warn("e")
	l.Warnf("%s", "f")
	l.Error("g")
	l.Errorf("%s", "h")
	l.Fatal("i")
	l.Fatalf("%s", "j")
}

func TestCryptoDefaultAndHelpers(t *testing.T) {
	crypto, err := NewDefaultCrypto("key123")
	if err != nil {
		t.Fatalf("create default crypto failed: %v", err)
	}
	data := []byte("hello")
	enc, err := crypto.Encrypt(data)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if !crypto.IsEncrypted(enc) {
		t.Fatalf("expected encrypted marker")
	}
	if crypto.IsEncrypted([]byte("plain")) {
		t.Fatalf("plain data should not be considered encrypted")
	}
	dec, err := crypto.Decrypt(enc)
	if err != nil || string(dec) != "hello" {
		t.Fatalf("decrypt mismatch: %v %s", err, string(dec))
	}

	// 解密非加密数据应报错
	if _, err := crypto.Decrypt([]byte("plain")); err == nil {
		t.Fatalf("expected decrypt error on plain data")
	}

	// 从导出密钥恢复
	k := crypto.GetKey()
	crypto2, err := NewDefaultCryptoFromKey(k)
	if err != nil {
		t.Fatalf("restore crypto failed: %v", err)
	}
	if !crypto2.IsEncrypted(enc) {
		t.Fatalf("restored crypto should detect encryption")
	}

	// 非法密钥长度
	if _, err := NewDefaultCryptoFromKey("short"); err == nil {
		t.Fatalf("expected error for short key")
	}

	// 别名工厂
	if _, err := NewCrypto("alias"); err != nil {
		t.Fatalf("NewCrypto alias failed: %v", err)
	}
	if _, err := NewChaCha20Crypto("alias"); err != nil {
		t.Fatalf("NewChaCha20Crypto alias failed: %v", err)
	}
	if _, err := NewChaCha20CryptoFromKey(crypto.GetKey()); err != nil {
		t.Fatalf("NewChaCha20CryptoFromKey alias failed: %v", err)
	}
}

func TestErrorHandlingHelpers(t *testing.T) {
	tmp := t.TempDir()
	cfg := &Config{path: tmp, name: "c", mode: "yaml"}

	err := cfg.wrapError(os.ErrNotExist, "read")
	if !IsConfigError(err) || GetConfigErrorType(err) != ErrTypeFileNotFound {
		t.Fatalf("expected file not found config error, got %v", err)
	}
	if suggestion := GetErrorSuggestion(err); suggestion == "" {
		t.Fatalf("expected suggestion for file not found")
	}

	formatErr := cfg.wrapError(fmt.Errorf("yaml parse failed"), "parse")
	if GetConfigErrorType(formatErr) != ErrTypeInvalidFormat {
		t.Fatalf("expected invalid format, got %s", GetConfigErrorType(formatErr))
	}

	decryptErr := cfg.wrapError(fmt.Errorf("decrypt failure"), "decrypt")
	if GetConfigErrorType(decryptErr) != ErrTypeDecryption {
		t.Fatalf("expected decryption type")
	}

	r, w, _ := os.Pipe()
	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()
	os.Stdout = w
	PrintErrorHelp(decryptErr)
	_ = w.Close()
	buf, _ := io.ReadAll(r)
	if len(buf) == 0 {
		t.Fatalf("PrintErrorHelp should output help text")
	}

	// Recover 分支覆盖
	er := NewErrorRecovery(cfg)
	if err := er.RecoverFromError(err); err == nil {
		t.Fatalf("recover file not found should return error")
	}
	if err := er.RecoverFromError(formatErr); err == nil {
		t.Fatalf("recover format error should return error")
	}
	decryptionCfg := newTestConfig(t)
	decryptionCfg.cryptoOptions.Enabled = true
	decryptionCfg.crypto, _ = NewDefaultCrypto("abc")
	er2 := NewErrorRecovery(decryptionCfg)
	_ = er2.RecoverFromError(NewConfigError(ErrTypeDecryption, "x")) // 只覆盖分支，不断言结果
}

func TestMarshalToINIAndConfigFileHelpers(t *testing.T) {
	cfg := newTestConfig(t)
	defer cfg.Close()

	settings := map[string]any{
		"server": map[string]any{"port": 8080},
		"debug":  true,
	}
	ini, err := cfg.marshalToINI(settings)
	if err != nil {
		t.Fatalf("marshalToINI failed: %v", err)
	}
	content := string(ini)
	if !bytes.Contains(ini, []byte("[server]")) || !bytes.Contains(ini, []byte("port = 8080")) {
		t.Fatalf("ini output unexpected: %s", content)
	}

	// 验证加密开关和密钥获取
	cfg.cryptoOptions.Enabled = true
	cfg.crypto, _ = NewDefaultCrypto("abc")
	if !cfg.IsEncryptionEnabled() {
		t.Fatalf("encryption flag should be true")
	}
	if cfg.GetEncryptionKey() == "" {
		t.Fatalf("encryption key should not be empty")
	}
	if cfg.GetCryptoType() == "" {
		t.Fatalf("crypto type should be non-empty")
	}

	// 加密写入异常（nil data）
	if err := cfg.writeConfigFileWithData(nil); err == nil {
		t.Fatalf("nil data should fail")
	}
}

func TestConfigMetadataAccessors(t *testing.T) {
	cfg := newTestConfig(t)
	defer cfg.Close()

	// 仅验证关键元数据非空
	if cfg.Viper() == nil {
		t.Fatalf("Viper should not be nil")
	}
}
