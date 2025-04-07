package sysconf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkPath(t *testing.T) {
	// 保存当前工作目录
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	defer os.Chdir(origDir) // 测试完成后恢复

	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), "sysconf_workpath_test")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 切换到临时目录
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("无法切换到临时目录: %v", err)
	}

	// 测试 WorkPath 函数
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, result string)
	}{
		{
			name:  "使用当前目录",
			input: ".",
			check: func(t *testing.T, result string) {
				if result != tempDir {
					t.Errorf("WorkPath('.') = %s, 期望 %s", result, tempDir)
				}
			},
		},
		{
			name:  "使用绝对路径",
			input: tempDir,
			check: func(t *testing.T, result string) {
				if result != tempDir {
					t.Errorf("WorkPath(absolute) = %s, 期望 %s", result, tempDir)
				}
			},
		},
		{
			name:  "使用相对路径",
			input: "./subdir",
			check: func(t *testing.T, result string) {
				expected := filepath.Join(tempDir, "subdir")
				if result != expected {
					t.Errorf("WorkPath(relative) = %s, 期望 %s", result, expected)
				}
			},
		},
		{
			name:  "空路径",
			input: "",
			check: func(t *testing.T, result string) {
				if result != tempDir {
					t.Errorf("WorkPath('') = %s, 期望 %s", result, tempDir)
				}
			},
		},
		{
			name:  "使用~进行展开",
			input: "~/myconfig",
			check: func(t *testing.T, result string) {
				// 获取用户主目录
				home, err := os.UserHomeDir()
				if err != nil {
					t.Skipf("无法获取用户主目录: %v", err)
					return
				}

				expected := filepath.Join(home, "myconfig")
				if result != expected {
					t.Errorf("WorkPath('~/myconfig') = %s, 期望 %s", result, expected)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WorkPath(tt.input)
			tt.check(t, result)
		})
	}

	// 测试不存在的目录
	nonExistPath := filepath.Join(tempDir, "nonexistent")
	result := WorkPath(nonExistPath)
	if result != nonExistPath {
		t.Errorf("WorkPath(nonexistent) = %s, 期望 %s", result, nonExistPath)
	}

	// 测试文件而非目录
	testFile := filepath.Join(tempDir, "testfile")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("无法创建测试文件: %v", err)
	}

	result = WorkPath(testFile)
	// 文件路径也应该被返回，因为函数只负责解析路径，不验证是否为目录
	if result != testFile {
		t.Errorf("WorkPath(file) = %s, 期望 %s", result, testFile)
	}
}
