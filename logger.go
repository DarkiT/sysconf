package sysconf

// Logger 日志接口定义，用于配置系统中的日志记录
type Logger interface {
	// Debug 记录调试级别的日志
	Debug(args ...interface{})
	// Debugf 格式化记录调试级别的日志
	Debugf(format string, args ...interface{})
	// Info 记录信息级别的日志
	Info(args ...interface{})
	// Infof 格式化记录信息级别的日志
	Infof(format string, args ...interface{})
	// Warn 记录警告级别的日志
	Warn(args ...interface{})
	// Warnf 格式化记录警告级别的日志
	Warnf(format string, args ...interface{})
	// Error 记录错误级别的日志
	Error(args ...interface{})
	// Errorf 格式化记录错误级别的日志
	Errorf(format string, args ...interface{})
	// Fatal 记录致命错误级别的日志
	Fatal(args ...interface{})
	// Fatalf 格式化记录致命错误级别的日志
	Fatalf(format string, args ...interface{})
}

// NopLogger 空日志实现，不执行任何操作
type NopLogger struct{}

// Debug 实现Logger接口
func (l *NopLogger) Debug(args ...interface{}) {}

// Debugf 实现Logger接口
func (l *NopLogger) Debugf(format string, args ...interface{}) {}

// Info 实现Logger接口
func (l *NopLogger) Info(args ...interface{}) {}

// Infof 实现Logger接口
func (l *NopLogger) Infof(format string, args ...interface{}) {}

// Warn 实现Logger接口
func (l *NopLogger) Warn(args ...interface{}) {}

// Warnf 实现Logger接口
func (l *NopLogger) Warnf(format string, args ...interface{}) {}

// Error 实现Logger接口
func (l *NopLogger) Error(args ...interface{}) {}

// Errorf 实现Logger接口
func (l *NopLogger) Errorf(format string, args ...interface{}) {}

// Fatal 实现Logger接口
func (l *NopLogger) Fatal(args ...interface{}) {}

// Fatalf 实现Logger接口
func (l *NopLogger) Fatalf(format string, args ...interface{}) {}
