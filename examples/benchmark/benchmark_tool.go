package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/darkit/sysconf"
)

// BenchmarkSuite 基准测试套件
type BenchmarkSuite struct {
	results []BenchmarkResult
}

// BenchmarkResult 基准测试结果
type BenchmarkResult struct {
	Name        string
	Operations  int64
	NsPerOp     int64
	AllocsPerOp int64
	BytesPerOp  int64
	Duration    time.Duration
	Description string
}

// 配置测试场景
type TestScenario struct {
	Name        string
	EnvVarCount int
	ConfigSize  string
	Description string
}

func main() {
	fmt.Println("🚀 Sysconf 配置管理基准测试工具")
	fmt.Println(strings.Repeat("=", 50))

	suite := &BenchmarkSuite{}

	// 运行所有基准测试
	suite.runAllBenchmarks()

	// 生成报告
	suite.generateReport()
}

func (s *BenchmarkSuite) runAllBenchmarks() {
	fmt.Println("📊 开始执行基准测试...")

	// 1. 配置初始化性能测试
	s.benchmarkConfigInit()

	// 2. 环境变量绑定性能测试
	s.benchmarkEnvBinding()

	// 3. 配置获取性能测试
	s.benchmarkConfigGet()

	// 4. 并发访问性能测试
	s.benchmarkConcurrentAccess()

	// 5. 配置文件I/O性能测试
	s.benchmarkFileIO()

	// 6. 内存使用测试
	s.benchmarkMemoryUsage()

	// 7. 大型配置性能测试
	s.benchmarkLargeConfig()

	fmt.Println("✅ 所有基准测试完成")
}

// 1. 配置初始化性能测试
func (s *BenchmarkSuite) benchmarkConfigInit() {
	fmt.Println("\n🔧 测试配置初始化性能...")

	scenarios := []TestScenario{
		{"基础配置", 0, "small", "最小配置初始化"},
		{"环境变量配置", 100, "small", "包含环境变量的配置"},
		{"完整配置", 100, "large", "包含所有功能的配置"},
	}

	for _, scenario := range scenarios {
		result := testing.Benchmark(func(b *testing.B) {
			s.setupTestEnv(scenario.EnvVarCount)
			defer s.cleanupTestEnv(scenario.EnvVarCount)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cfg, err := sysconf.New(
					sysconf.WithEnv("BENCH"),
					sysconf.WithPath("./testdata"),
					sysconf.WithName("config"),
				)
				if err != nil {
					b.Fatalf("配置初始化失败: %v", err)
				}
				_ = cfg
			}
		})

		s.results = append(s.results, BenchmarkResult{
			Name:        fmt.Sprintf("ConfigInit_%s", scenario.Name),
			Operations:  int64(result.N),
			NsPerOp:     result.NsPerOp(),
			AllocsPerOp: int64(result.AllocsPerOp()),
			BytesPerOp:  int64(result.AllocedBytesPerOp()),
			Duration:    result.T,
			Description: scenario.Description,
		})
	}
}

// 2. 环境变量绑定性能测试
func (s *BenchmarkSuite) benchmarkEnvBinding() {
	fmt.Println("\n🌍 测试环境变量绑定性能...")

	envCounts := []int{10, 100, 500, 1000, 5000}

	for _, count := range envCounts {
		result := testing.Benchmark(func(b *testing.B) {
			s.setupTestEnv(count)
			defer s.cleanupTestEnv(count)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cfg, err := sysconf.New(
					sysconf.WithEnv("BENCH"),
				)
				if err != nil {
					b.Fatalf("环境变量绑定失败: %v", err)
				}
				_ = cfg
			}
		})

		s.results = append(s.results, BenchmarkResult{
			Name:        fmt.Sprintf("EnvBinding_%d_vars", count),
			Operations:  int64(result.N),
			NsPerOp:     result.NsPerOp(),
			AllocsPerOp: int64(result.AllocsPerOp()),
			BytesPerOp:  int64(result.AllocedBytesPerOp()),
			Duration:    result.T,
			Description: fmt.Sprintf("绑定%d个环境变量", count),
		})
	}
}

// 3. 配置获取性能测试
func (s *BenchmarkSuite) benchmarkConfigGet() {
	fmt.Println("\n📖 测试配置获取性能...")

	cfg := s.setupLargeConfig()

	testCases := []struct {
		name string
		key  string
		desc string
	}{
		{"简单键", "simple.key", "获取简单配置值"},
		{"嵌套键", "database.connection.host", "获取深层嵌套值"},
		{"数组索引", "servers.0.host", "获取数组元素"},
		{"不存在键", "nonexistent.key", "获取不存在的键"},
	}

	for _, tc := range testCases {
		result := testing.Benchmark(func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = cfg.GetString(tc.key)
			}
		})

		s.results = append(s.results, BenchmarkResult{
			Name:        fmt.Sprintf("ConfigGet_%s", tc.name),
			Operations:  int64(result.N),
			NsPerOp:     result.NsPerOp(),
			AllocsPerOp: int64(result.AllocsPerOp()),
			BytesPerOp:  int64(result.AllocedBytesPerOp()),
			Duration:    result.T,
			Description: tc.desc,
		})
	}
}

// 4. 并发访问性能测试
func (s *BenchmarkSuite) benchmarkConcurrentAccess() {
	fmt.Println("\n🔄 测试并发访问性能...")

	cfg := s.setupLargeConfig()

	concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

	for _, level := range concurrencyLevels {
		result := testing.Benchmark(func(b *testing.B) {
			b.SetParallelism(level)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = cfg.GetString("database.host")
				}
			})
		})

		s.results = append(s.results, BenchmarkResult{
			Name:        fmt.Sprintf("ConcurrentGet_%d_goroutines", level),
			Operations:  int64(result.N),
			NsPerOp:     result.NsPerOp(),
			AllocsPerOp: int64(result.AllocsPerOp()),
			BytesPerOp:  int64(result.AllocedBytesPerOp()),
			Duration:    result.T,
			Description: fmt.Sprintf("%d个协程并发读取", level),
		})
	}
}

// 5. 配置文件I/O性能测试
func (s *BenchmarkSuite) benchmarkFileIO() {
	fmt.Println("\n💾 测试配置文件I/O性能...")

	fileSizes := []string{"small", "medium", "large"}

	for _, size := range fileSizes {
		configContent := s.generateConfigContent(size)

		result := testing.Benchmark(func(b *testing.B) {
			tmpDir := s.createTempDir()
			defer os.RemoveAll(tmpDir)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cfg, err := sysconf.New(
					sysconf.WithPath(tmpDir),
					sysconf.WithName("config"),
					sysconf.WithMode("yaml"),
					sysconf.WithContent(configContent),
				)
				if err != nil {
					b.Fatalf("配置文件I/O失败: %v", err)
				}
				_ = cfg
			}
		})

		s.results = append(s.results, BenchmarkResult{
			Name:        fmt.Sprintf("FileIO_%s", size),
			Operations:  int64(result.N),
			NsPerOp:     result.NsPerOp(),
			AllocsPerOp: int64(result.AllocsPerOp()),
			BytesPerOp:  int64(result.AllocedBytesPerOp()),
			Duration:    result.T,
			Description: fmt.Sprintf("%s配置文件I/O", size),
		})
	}
}

// 6. 内存使用测试
func (s *BenchmarkSuite) benchmarkMemoryUsage() {
	fmt.Println("\n🧠 测试内存使用情况...")

	// 单个配置实例内存测试
	result := testing.Benchmark(func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := sysconf.New(
				sysconf.WithContent(s.generateConfigContent("medium")),
			)
			if err != nil {
				b.Fatalf("配置创建失败: %v", err)
			}
			_ = cfg
		}
	})

	s.results = append(s.results, BenchmarkResult{
		Name:        "MemoryUsage_single_instance",
		Operations:  int64(result.N),
		NsPerOp:     result.NsPerOp(),
		AllocsPerOp: int64(result.AllocsPerOp()),
		BytesPerOp:  int64(result.AllocedBytesPerOp()),
		Duration:    result.T,
		Description: "单个配置实例的内存使用",
	})

	// 大配置内存测试
	largeResult := testing.Benchmark(func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := sysconf.New(
				sysconf.WithContent(s.generateConfigContent("large")),
			)
			if err != nil {
				b.Fatalf("大配置创建失败: %v", err)
			}
			_ = cfg
		}
	})

	s.results = append(s.results, BenchmarkResult{
		Name:        "MemoryUsage_large_config",
		Operations:  int64(largeResult.N),
		NsPerOp:     largeResult.NsPerOp(),
		AllocsPerOp: int64(largeResult.AllocsPerOp()),
		BytesPerOp:  int64(largeResult.AllocedBytesPerOp()),
		Duration:    largeResult.T,
		Description: "大型配置的内存使用",
	})
}

// 7. 大型配置性能测试
func (s *BenchmarkSuite) benchmarkLargeConfig() {
	fmt.Println("\n📚 测试大型配置性能...")

	largeConfig := s.generateLargeConfig(10000) // 10k 配置项

	result := testing.Benchmark(func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := sysconf.New(
				sysconf.WithContent(largeConfig),
			)
			if err != nil {
				b.Fatalf("大型配置加载失败: %v", err)
			}

			// 测试访问性能
			_ = cfg.GetString("section_5000.key_5000")
		}
	})

	s.results = append(s.results, BenchmarkResult{
		Name:        "LargeConfig_10k_keys",
		Operations:  int64(result.N),
		NsPerOp:     result.NsPerOp(),
		AllocsPerOp: int64(result.AllocsPerOp()),
		BytesPerOp:  int64(result.AllocedBytesPerOp()),
		Duration:    result.T,
		Description: "10k配置项的大型配置",
	})
}

// 辅助函数

func (s *BenchmarkSuite) setupTestEnv(count int) {
	for i := 0; i < count; i++ {
		os.Setenv(fmt.Sprintf("BENCH_VAR_%d", i), fmt.Sprintf("value_%d", i))
	}
}

func (s *BenchmarkSuite) cleanupTestEnv(count int) {
	for i := 0; i < count; i++ {
		os.Unsetenv(fmt.Sprintf("BENCH_VAR_%d", i))
	}
}

func (s *BenchmarkSuite) setupLargeConfig() *sysconf.Config {
	content := `
database:
  host: "localhost"
  port: 5432
  username: "user"
  password: "pass"
  connection:
    host: "db.example.com"
    timeout: 30
server:
  port: 8080
  debug: true
  hosts: ["host1", "host2", "host3"]
servers:
  - host: "server1.com"
    port: 8080
  - host: "server2.com"
    port: 8081
simple:
  key: "value"
`

	cfg, err := sysconf.New(
		sysconf.WithContent(content),
	)
	if err != nil {
		panic(err)
	}

	return cfg
}

func (s *BenchmarkSuite) generateConfigContent(size string) string {
	switch size {
	case "small":
		return `
app:
  name: "test"
  version: "1.0.0"
`
	case "medium":
		return `
app:
  name: "test"
  version: "1.0.0"
database:
  host: "localhost"
  port: 5432
  pools:
    read: 10
    write: 5
server:
  port: 8080
  timeout: 30
  features:
    - auth
    - cors
    - gzip
logging:
  level: "info"
  outputs: ["stdout", "file"]
`
	case "large":
		var builder strings.Builder
		for i := 0; i < 100; i++ {
			builder.WriteString(fmt.Sprintf(`
section_%d:
  key_%d: "value_%d"
  number_%d: %d
  boolean_%d: %t
`, i, i, i, i, i, i, i%2 == 0))
		}
		return builder.String()
	}
	return ""
}

func (s *BenchmarkSuite) generateLargeConfig(keyCount int) string {
	var builder strings.Builder

	sectionsCount := keyCount / 100
	keysPerSection := 100

	for i := 0; i < sectionsCount; i++ {
		builder.WriteString(fmt.Sprintf("section_%d:\n", i))
		for j := 0; j < keysPerSection; j++ {
			builder.WriteString(fmt.Sprintf("  key_%d: \"value_%d_%d\"\n", j, i, j))
		}
	}

	return builder.String()
}

func (s *BenchmarkSuite) createTempDir() string {
	tmpDir, err := os.MkdirTemp("", "sysconf_bench_")
	if err != nil {
		panic(err)
	}
	return tmpDir
}

// 报告生成

func (s *BenchmarkSuite) generateReport() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📈 基准测试报告")
	fmt.Println(strings.Repeat("=", 80))

	// 控制台报告
	s.printConsoleReport()

	// 生成详细报告文件
	s.generateDetailedReport()

	// 生成性能趋势图数据
	s.generateTrendData()

	fmt.Println("\n✅ 报告生成完成")
	fmt.Println("📁 详细报告: ./benchmark_report.md")
	fmt.Println("📊 趋势数据: ./benchmark_trends.json")
}

func (s *BenchmarkSuite) printConsoleReport() {
	goVersion := runtime.Version()
	numCPU := runtime.NumCPU()
	gomax := runtime.GOMAXPROCS(0)

	fmt.Printf("\nGo %s | NumCPU=%d | GOMAXPROCS=%d | %s/%s\n",
		goVersion, numCPU, gomax, runtime.GOOS, runtime.GOARCH)

	fmt.Printf("\n%-40s %12s %12s %12s %12s\n",
		"测试名称", "操作次数", "ns/op", "allocs/op", "bytes/op")
	fmt.Println(strings.Repeat("-", 100))

	for _, result := range s.results {
		fmt.Printf("%-40s %12d %12d %12d %12d\n",
			result.Name,
			result.Operations,
			result.NsPerOp,
			result.AllocsPerOp,
			result.BytesPerOp,
		)
	}

	// 性能摘要
	s.printPerformanceSummary()
}

func (s *BenchmarkSuite) printPerformanceSummary() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎯 性能摘要")
	fmt.Println(strings.Repeat("=", 50))

	// 最快/最慢的操作
	fastest := s.results[0]
	slowest := s.results[0]

	for _, result := range s.results {
		if result.NsPerOp < fastest.NsPerOp && result.NsPerOp > 0 {
			fastest = result
		}
		if result.NsPerOp > slowest.NsPerOp {
			slowest = result
		}
	}

	fmt.Printf("🚀 最快操作: %s (%d ns/op)\n", fastest.Name, fastest.NsPerOp)
	fmt.Printf("🐌 最慢操作: %s (%d ns/op)\n", slowest.Name, slowest.NsPerOp)

	// 内存使用分析
	var totalMemory int64
	memoryTests := 0
	for _, result := range s.results {
		// 只统计内存相关的测试，排除异常值
		if strings.Contains(result.Name, "MemoryUsage") && result.BytesPerOp > 0 && result.BytesPerOp < 1000000000 {
			totalMemory += result.BytesPerOp
			memoryTests++
		}
	}

	if memoryTests > 0 {
		avgMemory := totalMemory / int64(memoryTests)
		fmt.Printf("🧠 平均内存使用: %d bytes/op\n", avgMemory)
	}

	// 推荐优化建议
	s.printOptimizationSuggestions()
}

func (s *BenchmarkSuite) printOptimizationSuggestions() {
	fmt.Println("\n💡 优化建议:")

	for _, result := range s.results {
		if strings.Contains(result.Name, "EnvBinding") && result.NsPerOp > 1000000 {
			fmt.Printf("   - 环境变量绑定较慢 (%s): 考虑实现前缀过滤优化\n", result.Name)
		}
		if strings.Contains(result.Name, "ConfigGet") && result.AllocsPerOp > 10 {
			fmt.Printf("   - 配置获取内存分配较多 (%s): 考虑实现缓存机制\n", result.Name)
		}
		if strings.Contains(result.Name, "FileIO") && result.NsPerOp > 5000000 {
			fmt.Printf("   - 文件I/O较慢 (%s): 考虑异步读取或内存映射\n", result.Name)
		}
	}
}

func (s *BenchmarkSuite) generateDetailedReport() {
	reportPath := "benchmark_report.md"
	file, err := os.Create(reportPath)
	if err != nil {
		fmt.Printf("无法创建报告文件: %v\n", err)
		return
	}
	defer file.Close()

	// 写入Markdown报告
	fmt.Fprintf(file, "# Sysconf 基准测试报告\n\n")
	fmt.Fprintf(file, "生成时间: %s\n\n", time.Now().Format(time.RFC3339))

	// 系统信息
	fmt.Fprintf(file, "## 系统信息\n\n")
	goVersion := runtime.Version()
	numCPU := runtime.NumCPU()
	gomax := runtime.GOMAXPROCS(0)

	fmt.Fprintf(file, "- Go版本: %s\n", goVersion)
	fmt.Fprintf(file, "- 操作系统: %s\n", runtime.GOOS)
	fmt.Fprintf(file, "- 架构: %s\n", runtime.GOARCH)
	fmt.Fprintf(file, "- CPU核心数: %d\n", numCPU)
	fmt.Fprintf(file, "- GOMAXPROCS: %d\n\n", gomax)

	// 详细结果
	fmt.Fprintf(file, "## 基准测试结果\n\n")
	fmt.Fprintf(file, "| 测试名称 | 操作次数 | ns/op | allocs/op | bytes/op | 描述 |\n")
	fmt.Fprintf(file, "|----------|----------|-------|-----------|----------|------|\n")

	for _, result := range s.results {
		fmt.Fprintf(file, "| %s | %d | %d | %d | %d | %s |\n",
			result.Name,
			result.Operations,
			result.NsPerOp,
			result.AllocsPerOp,
			result.BytesPerOp,
			result.Description,
		)
	}

	// 性能分析
	fmt.Fprintf(file, "\n## 性能分析\n\n")
	s.writePerformanceAnalysis(file)
}

func (s *BenchmarkSuite) writePerformanceAnalysis(file *os.File) {
	// 分类分析
	categories := map[string][]BenchmarkResult{
		"配置初始化": {},
		"环境变量":  {},
		"配置获取":  {},
		"并发访问":  {},
		"文件I/O": {},
		"内存使用":  {},
	}

	for _, result := range s.results {
		switch {
		case strings.Contains(result.Name, "ConfigInit"):
			categories["配置初始化"] = append(categories["配置初始化"], result)
		case strings.Contains(result.Name, "EnvBinding"):
			categories["环境变量"] = append(categories["环境变量"], result)
		case strings.Contains(result.Name, "ConfigGet"):
			categories["配置获取"] = append(categories["配置获取"], result)
		case strings.Contains(result.Name, "Concurrent"):
			categories["并发访问"] = append(categories["并发访问"], result)
		case strings.Contains(result.Name, "FileIO"):
			categories["文件I/O"] = append(categories["文件I/O"], result)
		case strings.Contains(result.Name, "Memory"):
			categories["内存使用"] = append(categories["内存使用"], result)
		}
	}

	for category, results := range categories {
		if len(results) > 0 {
			fmt.Fprintf(file, "### %s\n\n", category)
			for _, result := range results {
				fmt.Fprintf(file, "- **%s**: %d ns/op, %d allocs/op\n",
					result.Name, result.NsPerOp, result.AllocsPerOp)
			}
			fmt.Fprintf(file, "\n")
		}
	}
}

func (s *BenchmarkSuite) generateTrendData() {
	// 生成JSON格式的趋势数据，用于后续分析
	trendPath := "benchmark_trends.json"
	file, err := os.Create(trendPath)
	if err != nil {
		fmt.Printf("无法创建趋势数据文件: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "{\n")
	fmt.Fprintf(file, "  \"timestamp\": \"%s\",\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "  \"results\": [\n")

	for i, result := range s.results {
		fmt.Fprintf(file, "    {\n")
		fmt.Fprintf(file, "      \"name\": \"%s\",\n", result.Name)
		fmt.Fprintf(file, "      \"operations\": %d,\n", result.Operations)
		fmt.Fprintf(file, "      \"ns_per_op\": %d,\n", result.NsPerOp)
		fmt.Fprintf(file, "      \"allocs_per_op\": %d,\n", result.AllocsPerOp)
		fmt.Fprintf(file, "      \"bytes_per_op\": %d,\n", result.BytesPerOp)
		fmt.Fprintf(file, "      \"description\": \"%s\"\n", result.Description)

		if i < len(s.results)-1 {
			fmt.Fprintf(file, "    },\n")
		} else {
			fmt.Fprintf(file, "    }\n")
		}
	}

	fmt.Fprintf(file, "  ]\n")
	fmt.Fprintf(file, "}\n")
}
