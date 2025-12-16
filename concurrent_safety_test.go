package sysconf

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/darkit/sysconf/internal/testutil"

	"github.com/stretchr/testify/assert"
)

// TestConcurrentSet_NoDataLoss 验证并发 Set 操作不会丢失数据
func TestConcurrentSet_NoDataLoss(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "concurrent_set_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	cfg, err := New(
		WithPath(tmpDir),
		WithName("concurrent"),
		WithMode("yaml"),
		WithEncryption("test-key-32-bytes-long-enough!!"),
	)
	if err != nil {
		t.Fatalf("创建配置实例失败: %v", err)
	}
	testutil.Cleanup(t, cfg.Close)

	const goroutines = 10
	const iterations = 100
	var wg sync.WaitGroup

	// 并发写入数据
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				if err := cfg.Set(key, j); err != nil {
					t.Errorf("Set 失败 %s: %v", key, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 等待异步写入完成
	time.Sleep(4 * time.Second)

	// 验证所有数据都写入成功
	for i := 0; i < goroutines; i++ {
		for j := 0; j < iterations; j++ {
			key := fmt.Sprintf("key_%d_%d", i, j)
			got := cfg.GetInt(key)
			assert.Equal(t, j, got, "数据丢失或错误: %s, 期望 %d, 获得 %d", key, j, got)
		}
	}
}

// TestWriteConfigFileWithData_NilSnapshot 验证快照为 nil 时的处理
func TestWriteConfigFileWithData_NilSnapshot(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "nil_snapshot_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	cfg, err := New(
		WithPath(tmpDir),
		WithName("nil_test"),
		WithMode("yaml"),
	)
	if err != nil {
		t.Fatalf("创建配置实例失败: %v", err)
	}
	testutil.Cleanup(t, cfg.Close)

	// 直接调用 writeConfigFileWithData，传入 nil 快照
	err = cfg.writeConfigFileWithData(nil)
	assert.Error(t, err, "应该返回错误")
}

// TestSetWithoutEncryption_NoDeadlock 验证非加密模式不会死锁
func TestSetWithoutEncryption_NoDeadlock(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "no_encrypt_deadlock_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	cfg, err := New(
		WithPath(tmpDir),
		WithName("no_encrypt"),
		WithMode("yaml"),
		// 不启用加密
	)
	if err != nil {
		t.Fatalf("创建配置实例失败: %v", err)
	}
	testutil.Cleanup(t, cfg.Close)

	done := make(chan bool)
	go func() {
		if err := cfg.Set("key", "value"); err != nil {
			t.Errorf("Set 失败: %v", err)
		}
		done <- true
	}()

	select {
	case <-done:
		// OK，没有死锁
		assert.Equal(t, "value", cfg.GetString("key"))
	case <-time.After(5 * time.Second):
		t.Fatal("非加密模式死锁")
	}
}

// TestConcurrentReadWrite 验证并发读写不会产生数据竞争
func TestConcurrentReadWrite(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "concurrent_rw_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	cfg, err := New(
		WithPath(tmpDir),
		WithName("concurrent_rw"),
		WithMode("yaml"),
	)
	if err != nil {
		t.Fatalf("创建配置实例失败: %v", err)
	}
	testutil.Cleanup(t, cfg.Close)

	// 初始化一些数据
	for i := 0; i < 10; i++ {
		if err := cfg.Set(fmt.Sprintf("key%d", i), i); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// 启动读 goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					key := fmt.Sprintf("key%d", id%10)
					_ = cfg.GetInt(key)
					_ = cfg.AllSettings()
				}
			}
		}(i)
	}

	// 启动写 goroutines
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				select {
				case <-stop:
					return
				default:
					key := fmt.Sprintf("key%d", j%10)
					if err := cfg.Set(key, j); err != nil {
						t.Errorf("Set 失败 %s: %v", key, err)
						return
					}
				}
			}
		}(i)
	}

	// 运行 2 秒后停止
	time.Sleep(2 * time.Second)
	close(stop)
	wg.Wait()

	// 验证数据可读取
	assert.True(t, cfg.IsSet("key0"))
}

// TestEncryptedWriteWithConcurrentReads 验证加密写入时并发读取的安全性
func TestEncryptedWriteWithConcurrentReads(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "encrypted_concurrent_test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	cfg, err := New(
		WithPath(tmpDir),
		WithName("encrypted_concurrent"),
		WithMode("yaml"),
		WithEncryption("test-key-32-bytes-long-enough!!"),
	)
	if err != nil {
		t.Fatalf("创建配置实例失败: %v", err)
	}
	testutil.Cleanup(t, cfg.Close)

	// 初始化数据
	if err := cfg.Set("counter", 0); err != nil {
		t.Fatalf("Set(counter) failed: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// 并发写入
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if err := cfg.Set("counter", id*10+j); err != nil {
					errors <- fmt.Errorf("写入失败 goroutine %d: %w", id, err)
				}
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				val := cfg.GetInt("counter")
				if val < 0 {
					errors <- fmt.Errorf("读取到无效值 goroutine %d: %d", id, val)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Error(err)
	}

	// 最终值应该是有效的
	finalValue := cfg.GetInt("counter")
	assert.GreaterOrEqual(t, finalValue, 0, "最终值应该是非负数")
	assert.LessOrEqual(t, finalValue, 99, "最终值应该在合理范围内")
}
