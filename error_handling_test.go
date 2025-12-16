package sysconf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// helper: new config with default content and logger
func newErrTestConfig(t *testing.T) *Config {
	t.Helper()
	cfg, err := New(
		WithPath(t.TempDir()),
		WithName("err"),
		WithMode("yaml"),
		WithContent("a: 1\n"),
	)
	if err != nil {
		t.Fatalf("create config failed: %v", err)
	}
	return cfg
}

func TestRecoverFromErrorBranches(t *testing.T) {
	cfg := newErrTestConfig(t)
	er := NewErrorRecovery(cfg)

	// FileNotFound with default content -> createDefaultConfig
	notFoundErr := NewConfigError(ErrTypeFileNotFound, "missing")
	if err := er.RecoverFromError(notFoundErr); err != nil {
		t.Fatalf("recover file not found: %v", err)
	}

	// Permission -> read-only fallback
	permErr := NewConfigError(ErrTypePermission, "perm")
	if err := er.RecoverFromError(permErr); err == nil {
		t.Fatalf("permission recover should return config error")
	}

	// Format -> recreate from defaults
	formatErr := NewConfigError(ErrTypeInvalidFormat, "fmt")
	if err := er.RecoverFromError(formatErr); err != nil {
		t.Fatalf("format recover expected nil, got %v", err)
	}
}

func TestPrintErrorHelpVariants(t *testing.T) {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()
	os.Stdout = w

	errs := []error{
		NewConfigErrorWithDetails(ErrTypeValidation, "bad value", "k", "v", "f", nil),
		NewConfigError(ErrTypeDecryption, "fail"),
		NewConfigError(ErrTypeConversion, "conv"),
	}
	for _, e := range errs {
		PrintErrorHelp(e)
	}
	_ = w.Close()
	bufBytes, _ := io.ReadAll(r)
	if len(bufBytes) == 0 {
		t.Fatalf("expected help output")
	}
}

func TestWrapErrorEnvAndConversion(t *testing.T) {
	cfg := newErrTestConfig(t)

	// Env error: contains "environment"
	envErr := cfg.wrapError(errors.New("environment variable missing"), "env")
	if GetConfigErrorType(envErr) != ErrTypeInitialization {
		t.Fatalf("env error should be classified to initialization (default branch)")
	}

	// Conversion: craft message
	convErr := cfg.wrapError(errors.New("convert int"), "convert")
	if GetConfigErrorType(convErr) == "" {
		t.Fatalf("conversion should return config error")
	}

	// generic: ensure Unwrap works
	innerErr := fmt.Errorf("inner")
	wrapped := NewConfigErrorWithCause(ErrTypeConversion, "fail", innerErr)
	if !errors.Is(wrapped, innerErr) {
		t.Fatalf("unwrap should expose cause")
	}

	// getConfigFilePath with name/path set
	p := filepath.Base(cfg.getConfigFilePath())
	if p == "" {
		t.Fatalf("expected config file path")
	}
}
