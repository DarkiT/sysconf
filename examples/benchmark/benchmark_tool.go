package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkit/sysconf"
)

// BenchmarkSuite 基准测试套件
type BenchmarkSuite struct {
	results []BenchmarkResult
	cfg     *sysconf.Config // 共享配置实例
}

// BenchmarkResult 基准测试结果
type BenchmarkResult struct {
	Name        string        `json:"name"`
	Category    string        `json:"category"`
	Operations  int64         `json:"operations"`
	NsPerOp     int64         `json:"ns_per_op"`
	AllocsPerOp int64         `json:"allocs_per_op"`
	BytesPerOp  int64         `json:"bytes_per_op"`
	Duration    time.Duration `json:"duration"`
	Throughput  float64       `json:"throughput"` // ops/sec
	Description string        `json:"description"`
	Grade       string        `json:"grade"` // 性能等级 A/B/C/D
}

// 性能阈值配置
var perfThresholds = map[string]struct{ A, B, C int64 }{
	"get":    {100, 500, 2000},      // ns/op
	"set":    {5000, 20000, 100000}, // ns/op
	"init":   {100000, 500000, 2000000},
	"cache":  {50, 200, 1000},
	"concat": {1000, 5000, 20000},
}

func main() {
	fmt.Println("🚀 Sysconf 配置管理基准测试工具 v2.0")
	fmt.Println(strings.Repeat("=", 60))

	suite := &BenchmarkSuite{}

	// 初始化共享配置
	suite.initSharedConfig()

	// 运行所有基准测试
	suite.runAllBenchmarks()

	// 生成报告
	suite.generateReport()
}

func (s *BenchmarkSuite) initSharedConfig() {
	content := `
database:
  host: "localhost"
  port: 5432
  username: "admin"
  password: "secret"
  connection:
    host: "db.example.com"
    timeout: 30
    max_conns: 100
server:
  port: 8080
  debug: true
  hosts: ["host1", "host2", "host3"]
  timeout: "30s"
servers:
  - host: "server1.com"
    port: 8080
  - host: "server2.com"
    port: 8081
simple:
  key: "value"
  number: 42
  float: 3.14159
  enabled: true
features:
  auth: true
  cors: true
  gzip: false
metrics:
  rates: [0.5, 0.9, 0.95, 0.99]
  counts: [100, 200, 300]
`
	cfg, err := sysconf.New(sysconf.WithContent(content))
	if err != nil {
		panic(fmt.Sprintf("初始化共享配置失败: %v", err))
	}
	s.cfg = cfg
}

func (s *BenchmarkSuite) runAllBenchmarks() {
	fmt.Println("\n📊 开始执行基准测试...")

	// 核心读取性能
	s.benchmarkGetString()
	s.benchmarkGetInt()
	s.benchmarkGetBool()
	s.benchmarkGetFloat()
	s.benchmarkGetNested()

	// 泛型 API 性能
	s.benchmarkGetAs()
	s.benchmarkGetSliceAs()

	// 缓存性能
	s.benchmarkCacheHit()
	s.benchmarkCacheMiss()

	// 写入性能
	s.benchmarkSet()
	s.benchmarkSetNested()

	// 并发性能
	s.benchmarkConcurrentRead()
	s.benchmarkConcurrentReadWrite()

	// 初始化性能
	s.benchmarkConfigInit()
	s.benchmarkEnvBinding()

	// 大型配置
	s.benchmarkLargeConfig()

	// 类型转换性能
	s.benchmarkTypeConversion()

	fmt.Println("\n✅ 所有基准测试完成")
}

// ============================================================================
// 核心读取性能测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkGetString() {
	fmt.Print("  测试 GetString...")

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = s.cfg.GetString("database.host")
		}
	})

	s.addResult("GetString_simple", "get", result, "简单字符串获取")
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkGetInt() {
	fmt.Print("  测试 GetInt...")

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = s.cfg.GetInt("database.port")
		}
	})

	s.addResult("GetInt_simple", "get", result, "简单整数获取")
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkGetBool() {
	fmt.Print("  测试 GetBool...")

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = s.cfg.GetBool("server.debug")
		}
	})

	s.addResult("GetBool_simple", "get", result, "简单布尔值获取")
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkGetFloat() {
	fmt.Print("  测试 GetFloat...")

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = s.cfg.GetFloat("simple.float")
		}
	})

	s.addResult("GetFloat_simple", "get", result, "简单浮点数获取")
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkGetNested() {
	fmt.Print("  测试 GetNested...")

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = s.cfg.GetString("database.connection.host")
		}
	})

	s.addResult("GetString_nested", "get", result, "嵌套路径获取")
	fmt.Println(" ✓")
}

// ============================================================================
// 泛型 API 性能测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkGetAs() {
	fmt.Print("  测试 GetAs[T]...")

	// 测试不同类型
	types := []struct {
		name string
		fn   func()
	}{
		{"string", func() { _ = sysconf.GetAs[string](s.cfg, "database.host") }},
		{"int", func() { _ = sysconf.GetAs[int](s.cfg, "database.port") }},
		{"bool", func() { _ = sysconf.GetAs[bool](s.cfg, "server.debug") }},
		{"float64", func() { _ = sysconf.GetAs[float64](s.cfg, "simple.float") }},
		{"duration", func() { _ = sysconf.GetAs[time.Duration](s.cfg, "server.timeout") }},
	}

	for _, t := range types {
		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				t.fn()
			}
		})
		s.addResult(fmt.Sprintf("GetAs_%s", t.name), "get", result, fmt.Sprintf("泛型获取 %s 类型", t.name))
	}
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkGetSliceAs() {
	fmt.Print("  测试 GetSliceAs[T]...")

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = sysconf.GetSliceAs[float64](s.cfg, "metrics.rates")
		}
	})

	s.addResult("GetSliceAs_float64", "get", result, "泛型切片获取")
	fmt.Println(" ✓")
}

// ============================================================================
// 缓存性能测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkCacheHit() {
	fmt.Print("  测试缓存命中...")

	// 预热缓存
	_ = s.cfg.GetString("database.host")

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = s.cfg.GetString("database.host")
		}
	})

	s.addResult("CacheHit_repeated", "cache", result, "重复访问同一键（缓存命中）")
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkCacheMiss() {
	fmt.Print("  测试缓存未命中...")

	keys := []string{
		"database.host", "database.port", "server.port",
		"simple.key", "features.auth", "database.username",
	}

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = s.cfg.GetString(keys[i%len(keys)])
		}
	})

	s.addResult("CacheHit_rotating", "cache", result, "轮换访问多个键")
	fmt.Println(" ✓")
}

// ============================================================================
// 写入性能测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkSet() {
	fmt.Print("  测试 Set...")

	// 创建独立配置实例用于写入测试
	cfg, _ := sysconf.New(sysconf.WithContent("test: value"))

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cfg.Set("benchmark.key", fmt.Sprintf("value_%d", i))
		}
	})

	s.addResult("Set_simple", "set", result, "简单键值设置")
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkSetNested() {
	fmt.Print("  测试 Set 嵌套...")

	cfg, _ := sysconf.New(sysconf.WithContent("root: {}"))

	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cfg.Set(fmt.Sprintf("section.subsection.key_%d", i%100), i)
		}
	})

	s.addResult("Set_nested", "set", result, "嵌套路径设置")
	fmt.Println(" ✓")
}

// ============================================================================
// 并发性能测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkConcurrentRead() {
	fmt.Print("  测试并发读取...")

	concurrencyLevels := []int{1, 4, 8, 16}

	for _, level := range concurrencyLevels {
		result := testing.Benchmark(func(b *testing.B) {
			b.SetParallelism(level)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = s.cfg.GetString("database.host")
				}
			})
		})

		s.addResult(fmt.Sprintf("ConcurrentRead_%dG", level), "concat", result,
			fmt.Sprintf("%d 协程并发读取", level*runtime.GOMAXPROCS(0)))
	}
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkConcurrentReadWrite() {
	fmt.Print("  测试并发读写...")

	cfg, _ := sysconf.New(sysconf.WithContent("counter: 0"))

	result := testing.Benchmark(func(b *testing.B) {
		var wg sync.WaitGroup
		readers := 8
		writers := 2

		b.ResetTimer()

		// 启动读取协程
		for r := 0; r < readers; r++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < b.N/readers; i++ {
					_ = cfg.GetInt("counter")
				}
			}()
		}

		// 启动写入协程
		for w := 0; w < writers; w++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < b.N/(readers*10); i++ {
					_ = cfg.Set("counter", i)
				}
			}(w)
		}

		wg.Wait()
	})

	s.addResult("ConcurrentReadWrite_8R2W", "concat", result, "8读2写并发混合")
	fmt.Println(" ✓")
}

// ============================================================================
// 初始化性能测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkConfigInit() {
	fmt.Print("  测试配置初始化...")

	sizes := []struct {
		name    string
		content string
	}{
		{"minimal", "app: test"},
		{"small", s.generateConfigContent(10)},
		{"medium", s.generateConfigContent(100)},
	}

	for _, size := range sizes {
		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cfg, _ := sysconf.New(sysconf.WithContent(size.content))
				_ = cfg
			}
		})

		s.addResult(fmt.Sprintf("Init_%s", size.name), "init", result,
			fmt.Sprintf("%s 配置初始化", size.name))
	}
	fmt.Println(" ✓")
}

func (s *BenchmarkSuite) benchmarkEnvBinding() {
	fmt.Print("  测试环境变量绑定...")

	envCounts := []int{10, 100}

	for _, count := range envCounts {
		// 设置环境变量
		for i := 0; i < count; i++ {
			os.Setenv(fmt.Sprintf("BENCH_VAR_%d", i), fmt.Sprintf("value_%d", i))
		}

		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cfg, _ := sysconf.New(sysconf.WithEnv("BENCH"))
				_ = cfg
			}
		})

		// 清理环境变量
		for i := 0; i < count; i++ {
			os.Unsetenv(fmt.Sprintf("BENCH_VAR_%d", i))
		}

		s.addResult(fmt.Sprintf("EnvBinding_%d", count), "init", result,
			fmt.Sprintf("绑定 %d 个环境变量", count))
	}
	fmt.Println(" ✓")
}

// ============================================================================
// 大型配置测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkLargeConfig() {
	fmt.Print("  测试大型配置...")

	largeContent := s.generateConfigContent(1000)
	cfg, _ := sysconf.New(sysconf.WithContent(largeContent))

	// 测试大型配置中的访问性能
	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cfg.GetString(fmt.Sprintf("section_%d.key_%d", i%100, i%10))
		}
	})

	s.addResult("LargeConfig_1k_access", "get", result, "1000节配置随机访问")
	fmt.Println(" ✓")
}

// ============================================================================
// 类型转换测试
// ============================================================================

func (s *BenchmarkSuite) benchmarkTypeConversion() {
	fmt.Print("  测试类型转换...")

	// 测试从 interface{} 到目标类型的转换
	cfg, _ := sysconf.New(sysconf.WithContent(`
number_str: "12345"
bool_str: "true"
float_str: "3.14159"
`))

	conversions := []struct {
		name string
		fn   func()
	}{
		{"str_to_int", func() { _ = cfg.GetInt("number_str") }},
		{"str_to_bool", func() { _ = cfg.GetBool("bool_str") }},
		{"str_to_float", func() { _ = cfg.GetFloat("float_str") }},
	}

	for _, c := range conversions {
		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				c.fn()
			}
		})
		s.addResult(fmt.Sprintf("TypeConv_%s", c.name), "get", result,
			fmt.Sprintf("类型转换: %s", c.name))
	}
	fmt.Println(" ✓")
}

// ============================================================================
// 辅助函数
// ============================================================================

func (s *BenchmarkSuite) addResult(name, category string, result testing.BenchmarkResult, desc string) {
	nsPerOp := result.NsPerOp()
	throughput := 0.0
	if nsPerOp > 0 {
		throughput = 1e9 / float64(nsPerOp)
	}

	grade := s.calculateGrade(category, nsPerOp)

	s.results = append(s.results, BenchmarkResult{
		Name:        name,
		Category:    category,
		Operations:  int64(result.N),
		NsPerOp:     nsPerOp,
		AllocsPerOp: int64(result.AllocsPerOp()),
		BytesPerOp:  int64(result.AllocedBytesPerOp()),
		Duration:    result.T,
		Throughput:  throughput,
		Description: desc,
		Grade:       grade,
	})
}

func (s *BenchmarkSuite) calculateGrade(category string, nsPerOp int64) string {
	thresholds, ok := perfThresholds[category]
	if !ok {
		thresholds = perfThresholds["get"]
	}

	switch {
	case nsPerOp <= thresholds.A:
		return "A"
	case nsPerOp <= thresholds.B:
		return "B"
	case nsPerOp <= thresholds.C:
		return "C"
	default:
		return "D"
	}
}

func (s *BenchmarkSuite) generateConfigContent(sections int) string {
	var builder strings.Builder
	for i := 0; i < sections; i++ {
		builder.WriteString(fmt.Sprintf("section_%d:\n", i))
		for j := 0; j < 10; j++ {
			builder.WriteString(fmt.Sprintf("  key_%d: \"value_%d_%d\"\n", j, i, j))
		}
	}
	return builder.String()
}

// ============================================================================
// 报告生成
// ============================================================================

func (s *BenchmarkSuite) generateReport() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📈 基准测试报告")
	fmt.Println(strings.Repeat("=", 60))

	s.printConsoleReport()
	s.generateMarkdownReport()
	s.generateJSONReport()

	fmt.Println("\n✅ 报告生成完成")
	fmt.Println("📁 Markdown报告: ./benchmark_report.md")
	fmt.Println("📊 JSON数据: ./benchmark_trends.json")
}

func (s *BenchmarkSuite) printConsoleReport() {
	fmt.Printf("\nGo %s | CPU=%d | GOMAXPROCS=%d | %s/%s\n",
		runtime.Version(), runtime.NumCPU(), runtime.GOMAXPROCS(0),
		runtime.GOOS, runtime.GOARCH)

	fmt.Printf("\n%-35s %8s %10s %8s %10s %6s\n",
		"测试名称", "ops", "ns/op", "allocs", "bytes", "等级")
	fmt.Println(strings.Repeat("-", 85))

	// 按类别分组显示
	categories := []string{"get", "cache", "set", "concat", "init"}
	categoryNames := map[string]string{
		"get":    "📖 读取操作",
		"cache":  "💾 缓存操作",
		"set":    "✏️  写入操作",
		"concat": "🔄 并发操作",
		"init":   "🚀 初始化",
	}

	for _, cat := range categories {
		results := s.filterByCategory(cat)
		if len(results) == 0 {
			continue
		}

		fmt.Printf("\n%s\n", categoryNames[cat])
		for _, r := range results {
			gradeIcon := map[string]string{"A": "🟢", "B": "🟡", "C": "🟠", "D": "🔴"}[r.Grade]
			fmt.Printf("  %-33s %8d %10d %8d %10d %s %s\n",
				r.Name, r.Operations, r.NsPerOp, r.AllocsPerOp, r.BytesPerOp, gradeIcon, r.Grade)
		}
	}

	s.printSummary()
}

func (s *BenchmarkSuite) filterByCategory(category string) []BenchmarkResult {
	var filtered []BenchmarkResult
	for _, r := range s.results {
		if r.Category == category {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (s *BenchmarkSuite) printSummary() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎯 性能摘要")
	fmt.Println(strings.Repeat("=", 60))

	// 统计各等级数量
	grades := map[string]int{"A": 0, "B": 0, "C": 0, "D": 0}
	for _, r := range s.results {
		grades[r.Grade]++
	}

	fmt.Printf("性能等级分布: 🟢A=%d 🟡B=%d 🟠C=%d 🔴D=%d\n",
		grades["A"], grades["B"], grades["C"], grades["D"])

	// 找出最快和最慢的操作
	if len(s.results) > 0 {
		sorted := make([]BenchmarkResult, len(s.results))
		copy(sorted, s.results)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].NsPerOp < sorted[j].NsPerOp
		})

		fmt.Printf("🚀 最快: %s (%d ns/op, %.0f ops/s)\n",
			sorted[0].Name, sorted[0].NsPerOp, sorted[0].Throughput)
		fmt.Printf("🐌 最慢: %s (%d ns/op, %.0f ops/s)\n",
			sorted[len(sorted)-1].Name, sorted[len(sorted)-1].NsPerOp, sorted[len(sorted)-1].Throughput)
	}

	// 读取操作平均性能
	var totalGetNs int64
	var getCount int
	for _, r := range s.results {
		if r.Category == "get" {
			totalGetNs += r.NsPerOp
			getCount++
		}
	}
	if getCount > 0 {
		avgGetNs := totalGetNs / int64(getCount)
		fmt.Printf("📖 读取操作平均: %d ns/op (%.0f ops/s)\n", avgGetNs, 1e9/float64(avgGetNs))
	}
}

func (s *BenchmarkSuite) generateMarkdownReport() {
	file, err := os.Create("benchmark_report.md")
	if err != nil {
		fmt.Printf("无法创建报告文件: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "# Sysconf 基准测试报告\n\n")
	fmt.Fprintf(file, "> 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 系统信息
	fmt.Fprintf(file, "## 系统信息\n\n")
	fmt.Fprintf(file, "| 项目 | 值 |\n|------|----|\n")
	fmt.Fprintf(file, "| Go版本 | %s |\n", runtime.Version())
	fmt.Fprintf(file, "| 操作系统 | %s |\n", runtime.GOOS)
	fmt.Fprintf(file, "| 架构 | %s |\n", runtime.GOARCH)
	fmt.Fprintf(file, "| CPU核心数 | %d |\n", runtime.NumCPU())
	fmt.Fprintf(file, "| GOMAXPROCS | %d |\n\n", runtime.GOMAXPROCS(0))

	// 性能等级说明
	fmt.Fprintf(file, "## 性能等级说明\n\n")
	fmt.Fprintf(file, "| 等级 | 读取 (ns/op) | 写入 (ns/op) | 初始化 (ns/op) |\n")
	fmt.Fprintf(file, "|------|-------------|-------------|----------------|\n")
	fmt.Fprintf(file, "| 🟢 A | ≤100 | ≤5,000 | ≤100,000 |\n")
	fmt.Fprintf(file, "| 🟡 B | ≤500 | ≤20,000 | ≤500,000 |\n")
	fmt.Fprintf(file, "| 🟠 C | ≤2,000 | ≤100,000 | ≤2,000,000 |\n")
	fmt.Fprintf(file, "| 🔴 D | >2,000 | >100,000 | >2,000,000 |\n\n")

	// 详细结果表格
	fmt.Fprintf(file, "## 详细测试结果\n\n")
	fmt.Fprintf(file, "| 测试名称 | 类别 | ops | ns/op | allocs | bytes | 吞吐量 | 等级 |\n")
	fmt.Fprintf(file, "|----------|------|-----|-------|--------|-------|--------|------|\n")

	for _, r := range s.results {
		gradeIcon := map[string]string{"A": "🟢", "B": "🟡", "C": "🟠", "D": "🔴"}[r.Grade]
		fmt.Fprintf(file, "| %s | %s | %d | %d | %d | %d | %.0f/s | %s %s |\n",
			r.Name, r.Category, r.Operations, r.NsPerOp, r.AllocsPerOp, r.BytesPerOp, r.Throughput, gradeIcon, r.Grade)
	}

	// 性能分析
	fmt.Fprintf(file, "\n## 性能分析\n\n")

	categories := map[string]string{
		"get":    "### 读取操作分析",
		"cache":  "### 缓存性能分析",
		"set":    "### 写入操作分析",
		"concat": "### 并发性能分析",
		"init":   "### 初始化性能分析",
	}

	for cat, title := range categories {
		results := s.filterByCategory(cat)
		if len(results) == 0 {
			continue
		}

		fmt.Fprintf(file, "\n%s\n\n", title)
		for _, r := range results {
			fmt.Fprintf(file, "- **%s**: %d ns/op (%s) - %s\n",
				r.Name, r.NsPerOp, r.Grade, r.Description)
		}
	}
}

func (s *BenchmarkSuite) generateJSONReport() {
	file, err := os.Create("benchmark_trends.json")
	if err != nil {
		fmt.Printf("无法创建JSON文件: %v\n", err)
		return
	}
	defer file.Close()

	report := struct {
		Timestamp string            `json:"timestamp"`
		System    map[string]any    `json:"system"`
		Results   []BenchmarkResult `json:"results"`
		Summary   map[string]any    `json:"summary"`
	}{
		Timestamp: time.Now().Format(time.RFC3339),
		System: map[string]any{
			"go_version": runtime.Version(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"num_cpu":    runtime.NumCPU(),
			"gomaxprocs": runtime.GOMAXPROCS(0),
		},
		Results: s.results,
		Summary: s.calculateSummary(),
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
}

func (s *BenchmarkSuite) calculateSummary() map[string]any {
	grades := map[string]int{"A": 0, "B": 0, "C": 0, "D": 0}
	var totalNs int64
	var totalAllocs int64

	for _, r := range s.results {
		grades[r.Grade]++
		totalNs += r.NsPerOp
		totalAllocs += r.AllocsPerOp
	}

	count := int64(len(s.results))
	if count == 0 {
		count = 1
	}

	return map[string]any{
		"total_tests":   len(s.results),
		"grade_a":       grades["A"],
		"grade_b":       grades["B"],
		"grade_c":       grades["C"],
		"grade_d":       grades["D"],
		"avg_ns_per_op": totalNs / count,
		"avg_allocs":    totalAllocs / count,
	}
}
