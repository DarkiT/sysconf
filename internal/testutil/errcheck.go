package testutil

import "testing"

// Cleanup runs fn in t.Cleanup and fails the test if fn returns an error.
func Cleanup(t *testing.T, fn func() error) {
	t.Helper()
	t.Cleanup(func() {
		if err := fn(); err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
	})
}
