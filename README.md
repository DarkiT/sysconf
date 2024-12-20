# 配置管理器

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/sysconf.svg)](https://pkg.go.dev/github.com/darkit/sysconf)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/sysconf)](https://goreportcard.com/report/github.com/darkit/sysconf)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/sysconf/blob/master/LICENSE)

**Sysconf** 是一个灵活的 Go 程序配置管理包，提供了易用的 API 来处理配置文件、环境变量和热重载功能。

## 特性

- 支持多种配置格式：YAML, JSON, TOML, INI, ENV, Properties
- 环境变量覆盖配置
- 配置热重载
- 默认值设置
- 必填字段验证
- 结构体映射
- 类型自动转换
- 线程安全

## 安装

```bash
go get github.com/darkit/sysconf
```

## 快速开始

### 基础使用

```go
package main

import (
    "log"
	
    "github.com/darkit/sysconf"
)

func main() {
    cfg, err := sysconf.New(
        sysconf.WithPath("configs"),          
        sysconf.WithMode("yaml"),             
        sysconf.WithName("app"),              
        sysconf.WithContent(defaultConfig),    
        sysconf.WithEnvOptions(sysconf.EnvOptions{
            Prefix:  "APP",
            Enabled: true,
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    host := cfg.GetString("database.host", "localhost")
    port := cfg.GetInt("database.port")
}
```

**注意事项：**
1. 配置路径不存在时会自动创建。
2. 配置文件不存在且提供了默认内容时会自动创建。
3. 默认配置格式为 YAML。
4. 环境变量支持可选开启。

### 使用全局配置

```go
package main

import "github.com/darkit/sysconf"

func main() {
    cfg := sysconf.Default()
    sysconf.Register("database", "host", "localhost")
    host := cfg.GetString("database.host")
}
```

### 配置结构体映射

```go
type DatabaseConfig struct {
    Host     string        `config:"host" default:"localhost"`
    Port     int           `config:"port" default:"5432"`
    Username string         `config:"username" required:"true"`
    Password string         `config:"password" required:"true"`
    Timeout  time.Duration  `config:"timeout" default:"30s"`
}

func main() {
    cfg := sysconf.Default()
    
    var dbConfig DatabaseConfig
    if err := cfg.Unmarshal(&dbConfig,"database"); err != nil {
        log.Fatal(err)
    }
}
```

### 环境变量支持

配置遵循 Viper 的环境变量命名规则：

- 环境变量名称全大写。
- 配置键中的点号 (.) 转换为下划线 (_).
- 支持设置前缀以避免命名冲突。

**示例：**
```yaml
database:
  host: localhost
  port: 5432
```

对应的环境变量：
```bash
DATABASE_HOST=prod-db
DATABASE_PORT=5432

APP_DATABASE_HOST=prod-db
APP_DATABASE_PORT=5432
```

### 配置热重载

```go
cfg.Watch(func() {
    log.Println("配置已更新")
})
```

**注意事项：**
- 内置 1 秒防抖机制。
- 仅文件写入操作触发回调。
- 确保配置内容有效。

### 配置更新

```go
err := cfg.Set("database.host", "new-host")
if err != nil {
    log.Printf("更新配置失败: %v", err)
}
```

**注意事项：**
- Set 方法自动写入配置文件，内置 3 秒写入延迟，合并短时间内的更新。
- 支持任意有效的配置路径。

## API 文档

### 配置选项

- `WithPath(path string)`: 设置配置文件路径。
- `WithMode(mode string)`: 设置配置文件格式。
- `WithName(name string)`: 设置配置文件名称。
- `WithContent(content string)`: 设置默认配置内容。
- `WithEnvOptions(opts EnvOptions)`: 设置环境变量选项。

### 配置读取

支持两种读取方式：

1. 点号分隔的键名
   ```go
   host := cfg.GetString("database.host", "localhost")
   ```

2. 多参数方式
   ```go
   host := cfg.GetString("database", "host", "localhost")
   ```

所有配置读取方法都支持这两种方式。

### 配置解析

支持将配置解析到结构体，可以指定解析整个配置或特定配置段：

```go
type Config struct {
    Database struct {
        Host     string        `config:"host" default:"localhost"`
        Port     int           `config:"port" default:"5432"`
        Username string        `config:"username" required:"true"`
        Password string        `config:"password" required:"true"`
        Timeout  time.Duration `config:"timeout" default:"30s"`
    } `config:"database"`
}

// 解析整个配置
var config Config
if err := cfg.Unmarshal(&config); err != nil {
    log.Fatal(err)
}

// 或者只解析特定段
var dbConfig struct {
    Host     string        `config:"host" default:"localhost"`
    Port     int           `config:"port" default:"5432"`
    Username string        `config:"username" required:"true"`
    Password string        `config:"password" required:"true"`
    Timeout  time.Duration `config:"timeout" default:"30s"`
}

if err := cfg.Unmarshal(&dbConfig, "database"); err != nil {
    log.Fatal(err)
}
```

### 结构体标签

- `config:"key"`: 指定配置键名。
- `default:"value"`: 设置默认值。
- `required:"true"`: 标记必填字段。

### 类型转换支持

支持以下类型的自动转换：

- 字符串到数值类型 (int, float 等)
- 字符串到布尔值
- 字符串到时间类型 (time.Duration 和 time.Time)
- 字符串到切片 (支持逗号分隔的字符串和 JSON 数组)
- 字符串到映射 (支持 JSON 对象)

## 配置文件示例

```yaml
app:
   name: "MyApp"
   env: "development"

database:
   host: "localhost"
   port: 5432
   username: "user"
   password: "pass"
   timeout: "30s"

redis:
   host: "localhost"
   port: 6379
```

## 许可证

MIT License - 查看 [LICENSE](LICENSE) 文件了解详情。
