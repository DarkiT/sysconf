package sysconf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMarshalConfigUnsupportedMode(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.mode = "unsupported"
	if _, err := cfg.marshalConfig(); err == nil {
		t.Fatalf("unsupported mode should error")
	}
}

func TestWriteConfigFile_NoEncrypt(t *testing.T) {
	cfg := newTestConfig(t)
	defer cfg.Close()

	// 覆盖内存模式保护与写文件成功路径
	cfg.name = "" // 空 name -> 内存模式直接返回
	if err := cfg.writeConfigFile(); err != nil {
		t.Fatalf("memory mode should not error: %v", err)
	}

	cfg.name = "written"
	cfg.cryptoOptions.Enabled = false
	if err := cfg.writeConfigFile(); err != nil {
		t.Fatalf("write config without encryption should succeed: %v", err)
	}

	// 验证写入内容（非加密路径）
	target := filepath.Join(cfg.path, "written."+cfg.mode)
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read written file failed: %v", err)
	}
	if !bytes.Contains(data, []byte("database")) {
		t.Fatalf("expected marshaled config content, got: %s", string(data))
	}
}
