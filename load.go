package sysconf

// Load 一次完成 New + Unmarshal，是生产应用最常见的调用路径。
// T 应为结构体值类型（例如 AppConfig），不要传 *AppConfig。
//
//	app, cfg, err := sysconf.Load[AppConfig](
//	    sysconf.WithPath("config.yaml"),
//	    sysconf.WithEnv("APP"),
//	)
//	if err != nil { ... }
//	defer cfg.Close()
func Load[T any](opts ...Option) (T, *Config, error) {
	var value T
	cfg, err := New(opts...)
	if err != nil {
		return value, nil, err
	}
	if err := cfg.Unmarshal(&value); err != nil {
		_ = cfg.Close()
		return value, nil, err
	}
	return value, cfg, nil
}

// MustLoad 同 Load，失败时 panic。适用于 main 中的必达启动路径。
func MustLoad[T any](opts ...Option) (T, *Config) {
	value, cfg, err := Load[T](opts...)
	if err != nil {
		panic(err)
	}
	return value, cfg
}

// UnmarshalAs 将配置解码为 T 并返回值，避免先声明零值再取地址。
//
//	app, err := sysconf.UnmarshalAs[AppConfig](cfg)
//	server, err := sysconf.UnmarshalAs[ServerConfig](cfg, "server")
func UnmarshalAs[T any](c *Config, key ...string) (T, error) {
	var value T
	if c == nil {
		return value, ErrNilConfig
	}
	if err := c.Unmarshal(&value, key...); err != nil {
		return value, err
	}
	return value, nil
}
