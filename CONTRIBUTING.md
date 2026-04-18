# 贡献指南

感谢您对 sysconf 项目的关注！我们欢迎各种形式的贡献，包括但不限于：

- 报告 Bug
- 提交功能建议
- 改进文档
- 提交代码修复或新功能
- 分享使用经验

## 📋 目录

- [开发环境搭建](#开发环境搭建)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [Pull Request 流程](#pull-request-流程)
- [测试要求](#测试要求)
- [文档贡献](#文档贡献)

## 开发环境搭建

### 前置要求

- **Go 1.24+** - 项目最低要求
- **Git** - 版本控制
- **Make** - 构建工具（可选）

### 克隆项目

```bash
# 克隆仓库
git clone https://github.com/darkit/sysconf.git
cd sysconf

# 安装依赖
go mod download

# 验证环境
go version  # 应输出 1.24 或更高版本
```

### 常用命令

```bash
# 运行测试
make test
# 或
go test ./...

# 运行基准测试
make bench
# 或
go test -bench=. ./...

# 代码检查
make lint
# 或
golangci-lint run

# 格式化代码
make fmt
# 或
go fmt ./...

# 运行示例
cd examples/cmd/demo_basic && go run .
```

## 代码规范

### Go 代码风格

我们遵循标准的 Go 代码规范：

1. **使用 `gofmt` 格式化代码**
   ```bash
   go fmt ./...
   ```

2. **遵循 Effective Go 指南**
   - 使用驼峰命名（CamelCase）
   - 接口名以 `-er` 结尾（如 `Reader`, `Writer`）
   - 避免不必要的缩写

3. **错误处理**
   - 始终检查 error，不要忽略
   - 使用 `fmt.Errorf("...: %w", err)` 包装错误
   - 提供有意义的错误信息

4. **并发安全**
   - 共享状态必须使用同步机制
   - 优先使用 `sync.RWMutex` 或 `atomic`
   - 避免 data race

### 代码注释

- 所有导出的函数、类型、常量必须添加注释
- 注释以被注释对象的名称开头
- 使用中文注释（项目约定）

```go
// Config 配置管理器的核心结构体，提供线程安全的配置读写功能
type Config struct {
    // data 使用 atomic.Value 实现无锁读取
    data atomic.Value
    
    // mu 保护元数据和写操作
    mu sync.RWMutex
}

// GetString 获取字符串类型的配置值，支持默认值
func (c *Config) GetString(key string, def ...string) string {
    // ...
}
```

### 测试规范

1. **测试文件命名**: `*_test.go`
2. **测试函数命名**: `TestXxx`（导出函数）或 `testXxx`（内部函数）
3. **使用 testify**: 
   ```go
   import "github.com/stretchr/testify/assert"
   import "github.com/stretchr/testify/require"
   ```
4. **并发测试**: 必须包含并发安全测试

```go
func TestConfigConcurrent(t *testing.T) {
    cfg, err := sysconf.New(/* 选项 */)
    require.NoError(t, err)
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            cfg.Set(fmt.Sprintf("key%d", id), id)
            _ = cfg.GetInt(fmt.Sprintf("key%d", id))
        }(i)
    }
    wg.Wait()
}
```

## 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

### 提交格式

```
<类型>(<范围>): <描述>

[可选的正文]

[可选的脚注]
```

### 类型说明

| 类型 | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat: 添加泛型API支持` |
| `fix` | Bug修复 | `fix: 修复并发写入竞态条件` |
| `docs` | 文档更新 | `docs: 更新README示例代码` |
| `style` | 代码格式 | `style: 格式化代码` |
| `refactor` | 重构 | `refactor: 优化缓存实现` |
| `perf` | 性能优化 | `perf: 减少内存分配` |
| `test` | 测试相关 | `test: 添加并发安全测试` |
| `chore` | 构建/工具 | `chore: 更新依赖` |

### 范围说明

可选的范围标识修改的模块：

- `config` - 核心配置模块
- `getter` - 读取功能
- `setter` - 写入功能
- `validation` - 验证系统
- `crypto` - 加密模块
- `cache` - 缓存模块
- `docs` - 文档

### 提交示例

```bash
# 新功能
feat(validation): 添加 hostname_rfc1123 验证规则

# Bug修复
fix(crypto): 修复时序攻击漏洞

使用 crypto/subtle.ConstantTimeCompare 替代直接字节比较

# 文档更新
docs(readme): 修正 Go 版本要求

将 Go 版本从 1.23+ 更新为 1.24+

Refs: #123
```

## Pull Request 流程

### 1. 创建分支

```bash
# 从 main 分支创建新分支
git checkout main
git pull origin main
git checkout -b feat/your-feature-name
# 或
git checkout -b fix/issue-description
```

### 2. 开发和提交

```bash
# 进行代码修改
# ...

# 提交更改
git add .
git commit -m "feat(validation): 添加新验证规则"

# 推送到远程
git push origin feat/your-feature-name
```

### 3. 创建 PR

在 GitHub 上创建 Pull Request，请确保：

- [ ] PR 标题清晰描述变更
- [ ] 填写 PR 描述模板
- [ ] 关联相关 Issue（如果有）
- [ ] 所有 CI 检查通过
- [ ] 代码审查通过

### PR 描述模板

```markdown
## 描述
简要描述此 PR 的变更内容

## 变更类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 性能优化
- [ ] 文档更新
- [ ] 重构
- [ ] 其他

## 测试
- [ ] 已添加单元测试
- [ ] 已添加并发测试
- [ ] 所有测试通过

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 已更新相关文档
- [ ] 无 breaking changes（或已记录）
```

## 测试要求

### 必须包含的测试

1. **单元测试** - 覆盖正常路径和边界情况
2. **并发测试** - 验证线程安全性
3. **基准测试** - 性能关键路径

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行带竞态检测的测试
go test -race ./...

# 运行基准测试
go test -bench=. ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 测试覆盖率要求

- 核心模块：> 80%
- 验证系统：> 75%
- 加密模块：> 90%

## 文档贡献

### 文档类型

1. **README.md** - 项目主文档
2. **CHANGELOG.md** - 变更日志
3. **validation/README.md** - 验证器文档
4. **docs/** - 扩展文档
5. **examples/** - 示例代码

### 文档规范

- 使用中文编写文档
- 代码示例必须可运行
- 更新相关文档以保持一致性

### 更新 CHANGELOG

添加新功能或修复时，请在 `CHANGELOG.md` 的 `[Unreleased]` 部分添加条目：

```markdown
## [Unreleased]

### 新增 (Added)
- 新功能描述

### 修复 (Fixed)
- Bug 修复描述
```

## 获取帮助

如果您在贡献过程中遇到问题：

1. 查看 [GitHub Issues](https://github.com/darkit/sysconf/issues)
2. 发起 [Discussion](https://github.com/darkit/sysconf/discussions)
3. 阅读 [API 文档](https://pkg.go.dev/github.com/darkit/sysconf)

## 行为准则

参与本项目即表示您同意遵守我们的行为准则：

- 尊重所有参与者
- 接受建设性批评
- 关注对社区最有利的事情
- 对其他社区成员表示同理心

## 许可证

通过贡献代码，您同意您的贡献将在 [MIT 许可证](LICENSE) 下发布。

---

**感谢您对 sysconf 项目的贡献！**
