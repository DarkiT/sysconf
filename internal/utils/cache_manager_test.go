package utils

import (
	"testing"
	"time"
)

func TestCacheManagerSetNoDeadlockOnCleanup(t *testing.T) {
	cm := NewCacheManager(1)
	cm.SetCamelToSnake("first", "first")

	done := make(chan struct{})
	go func() {
		cm.SetCamelToSnake("second", "second")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("SetCamelToSnake blocked unexpectedly")
	}

	if got, ok := cm.GetCamelToSnake("second"); !ok || got != "second" {
		t.Fatalf("SetCamelToSnake failed, got=%q ok=%v", got, ok)
	}
}

func TestCacheManagerSetNoDeadlockSnakeCache(t *testing.T) {
	cm := NewCacheManager(1)
	cm.SetSnakeToCamel("first", "first")

	done := make(chan struct{})
	go func() {
		cm.SetSnakeToCamel("second", "second")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("SetSnakeToCamel blocked unexpectedly")
	}

	if got, ok := cm.GetSnakeToCamel("second"); !ok || got != "second" {
		t.Fatalf("SetSnakeToCamel failed, got=%q ok=%v", got, ok)
	}
}
