package sysconf

import "testing"

// 覆盖 NopLogger 空实现
func TestNopLogger(t *testing.T) {
	var l Logger = &NopLogger{}
	l.Debug("a")
	l.Debugf("%s", "a")
	l.Info("a")
	l.Infof("%s", "a")
	l.Warn("a")
	l.Warnf("%s", "a")
	l.Error("a")
	l.Errorf("%s", "a")
	l.Fatal("a")
	l.Fatalf("%s", "a")
}
