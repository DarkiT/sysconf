package sysconf

import (
	"errors"
	"testing"
)

// stub crypto to force encrypt error
type failingCrypto struct{ err error }

func (f failingCrypto) Encrypt([]byte) ([]byte, error) { return nil, f.err }
func (f failingCrypto) Decrypt([]byte) ([]byte, error) { return nil, nil }
func (f failingCrypto) IsEncrypted([]byte) bool        { return true }

func TestIsEncryptedNegative(t *testing.T) {
	crypto, _ := NewDefaultCrypto("k")
	// 破坏前缀
	data, _ := crypto.Encrypt([]byte("hello"))
	data[0] = 'X'
	if crypto.IsEncrypted(data) {
		t.Fatalf("tampered prefix should be detected as not encrypted")
	}
	if crypto.IsEncrypted([]byte("plain")) {
		t.Fatalf("plain text should not be encrypted")
	}
}

func TestEncryptFailurePropagation(t *testing.T) {
	cfg := newTestConfig(t)
	defer cfg.Close()
	cfg.cryptoOptions.Enabled = true
	cfg.crypto = failingCrypto{err: errors.New("boom")}

	if err := cfg.writeConfigFileWithData(map[string]any{"a": 1}); err == nil {
		t.Fatalf("expected encrypt failure to bubble up")
	}
}

func TestGetKeyBytesReturnsCopy(t *testing.T) {
	crypto, err := NewDefaultCrypto("secret")
	if err != nil {
		t.Fatalf("create crypto failed: %v", err)
	}

	k1 := crypto.GetKeyBytes()
	if len(k1) != 32 {
		t.Fatalf("key length mismatch, got %d", len(k1))
	}
	// 修改副本不应影响内部密钥
	k1[0] ^= 0xFF
	k2 := crypto.GetKeyBytes()
	if k1[0] == k2[0] {
		t.Fatalf("expected key bytes to be copied (immutable to caller)")
	}
}
