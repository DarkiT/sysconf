# Sysconf 基准测试报告

生成时间: 2025-07-02T16:29:37+08:00

## 系统信息

- Go版本: go1.24.2
- 操作系统: linux
- 架构: amd64
- CPU核心数: 383

## 基准测试结果

| 测试名称 | 操作次数 | ns/op | allocs/op | bytes/op | 描述 |
|----------|----------|-------|-----------|----------|------|
| ConfigInit_基础配置 | 15811 | 72838 | 308 | 18102 | 最小配置初始化 |
| ConfigInit_环境变量配置 | 8175 | 169663 | 2146 | 100551 | 包含环境变量的配置 |
| ConfigInit_完整配置 | 8088 | 177191 | 2146 | 104659 | 包含所有功能的配置 |
| EnvBinding_10_vars | 17602 | 70327 | 388 | 29576 | 绑定10个环境变量 |
| EnvBinding_100_vars | 9664 | 154300 | 2028 | 102368 | 绑定100个环境变量 |
| EnvBinding_500_vars | 1821 | 619531 | 9226 | 555858 | 绑定500个环境变量 |
| EnvBinding_1000_vars | 999 | 1167696 | 18231 | 1109166 | 绑定1000个环境变量 |
| EnvBinding_5000_vars | 187 | 6084532 | 90149 | 4889396 | 绑定5000个环境变量 |
| ConfigGet_简单键 | 2201272 | 569 | 6 | 160 | 获取简单配置值 |
| ConfigGet_嵌套键 | 1102196 | 1122 | 12 | 368 | 获取深层嵌套值 |
| ConfigGet_数组索引 | 1000000 | 1131 | 16 | 416 | 获取数组元素 |
| ConfigGet_不存在键 | 3572384 | 299 | 3 | 80 | 获取不存在的键 |
| ConcurrentGet_1_goroutines | 6609396 | 199 | 6 | 160 | 1个协程并发读取 |
| ConcurrentGet_2_goroutines | 4860673 | 392 | 6 | 160 | 2个协程并发读取 |
| ConcurrentGet_4_goroutines | 4295160 | 473 | 6 | 160 | 4个协程并发读取 |
| ConcurrentGet_8_goroutines | 3304923 | 363 | 6 | 160 | 8个协程并发读取 |
| ConcurrentGet_16_goroutines | 2608808 | 606 | 6 | 160 | 16个协程并发读取 |
| ConcurrentGet_32_goroutines | 1968396 | 511 | 6 | 160 | 32个协程并发读取 |
| FileIO_small | 20475 | 63507 | 172 | 17295 | small配置文件I/O |
| FileIO_medium | 12742 | 98161 | 440 | 30312 | medium配置文件I/O |
| FileIO_large | 1350 | 965133 | 6750 | 408752 | large配置文件I/O |
| MemoryUsage_single_instance | 12390 | 95753 | 406 | 27915 | 单个配置实例的内存使用 |
| MemoryUsage_large_config | 1324 | 958182 | 6819 | 432291 | 大型配置的内存使用 |
| LargeConfig_10k_keys | 69 | 22845298 | 163788 | 10212310 | 10k配置项的大型配置 |

## 性能分析

### 配置初始化

- **ConfigInit_基础配置**: 72838 ns/op, 308 allocs/op
- **ConfigInit_环境变量配置**: 169663 ns/op, 2146 allocs/op
- **ConfigInit_完整配置**: 177191 ns/op, 2146 allocs/op

### 环境变量

- **EnvBinding_10_vars**: 70327 ns/op, 388 allocs/op
- **EnvBinding_100_vars**: 154300 ns/op, 2028 allocs/op
- **EnvBinding_500_vars**: 619531 ns/op, 9226 allocs/op
- **EnvBinding_1000_vars**: 1167696 ns/op, 18231 allocs/op
- **EnvBinding_5000_vars**: 6084532 ns/op, 90149 allocs/op

### 配置获取

- **ConfigGet_简单键**: 569 ns/op, 6 allocs/op
- **ConfigGet_嵌套键**: 1122 ns/op, 12 allocs/op
- **ConfigGet_数组索引**: 1131 ns/op, 16 allocs/op
- **ConfigGet_不存在键**: 299 ns/op, 3 allocs/op

### 并发访问

- **ConcurrentGet_1_goroutines**: 199 ns/op, 6 allocs/op
- **ConcurrentGet_2_goroutines**: 392 ns/op, 6 allocs/op
- **ConcurrentGet_4_goroutines**: 473 ns/op, 6 allocs/op
- **ConcurrentGet_8_goroutines**: 363 ns/op, 6 allocs/op
- **ConcurrentGet_16_goroutines**: 606 ns/op, 6 allocs/op
- **ConcurrentGet_32_goroutines**: 511 ns/op, 6 allocs/op

### 文件I/O

- **FileIO_small**: 63507 ns/op, 172 allocs/op
- **FileIO_medium**: 98161 ns/op, 440 allocs/op
- **FileIO_large**: 965133 ns/op, 6750 allocs/op

### 内存使用

- **MemoryUsage_single_instance**: 95753 ns/op, 406 allocs/op
- **MemoryUsage_large_config**: 958182 ns/op, 6819 allocs/op

