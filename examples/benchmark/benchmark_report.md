# Sysconf 基准测试报告

> 生成时间: 2025-12-16 21:49:25

## 系统信息

| 项目 | 值 |
|------|----|
| Go版本 | go1.25.5 X:jsonv2,greenteagc |
| 操作系统 | linux |
| 架构 | amd64 |
| CPU核心数 | 8 |
| GOMAXPROCS | 8 |

## 性能等级说明

| 等级 | 读取 (ns/op) | 写入 (ns/op) | 初始化 (ns/op) |
|------|-------------|-------------|----------------|
| 🟢 A | ≤100 | ≤5,000 | ≤100,000 |
| 🟡 B | ≤500 | ≤20,000 | ≤500,000 |
| 🟠 C | ≤2,000 | ≤100,000 | ≤2,000,000 |
| 🔴 D | >2,000 | >100,000 | >2,000,000 |

## 详细测试结果

| 测试名称 | 类别 | ops | ns/op | allocs | bytes | 吞吐量 | 等级 |
|----------|------|-----|-------|--------|-------|--------|------|
| GetString_simple | get | 100000000 | 10 | 0 | 0 | 100000000/s | 🟢 A |
| GetInt_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetBool_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetFloat_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetString_nested | get | 88044018 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetAs_string | get | 88878420 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_int | get | 88669264 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_bool | get | 99628244 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_float64 | get | 99917168 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_duration | get | 23145954 | 46 | 1 | 8 | 21739130/s | 🟢 A |
| GetSliceAs_float64 | get | 5645626 | 208 | 4 | 112 | 4807692/s | 🟡 B |
| CacheHit_repeated | cache | 100000000 | 10 | 0 | 0 | 100000000/s | 🟢 A |
| CacheHit_rotating | cache | 62346115 | 19 | 0 | 1 | 52631579/s | 🟢 A |
| Set_simple | set | 918558 | 4992 | 28 | 2529 | 200321/s | 🟢 A |
| Set_nested | set | 20346 | 71167 | 612 | 83768 | 14051/s | 🟠 C |
| ConcurrentRead_1G | concat | 480670771 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentRead_4G | concat | 495015112 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentRead_8G | concat | 502833111 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentRead_16G | concat | 466089321 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentReadWrite_8R2W | concat | 40162653 | 106 | 0 | 44 | 9433962/s | 🟢 A |
| Init_minimal | init | 36250 | 33080 | 114 | 14588 | 30230/s | 🟢 A |
| Init_small | init | 5570 | 253997 | 2813 | 197054 | 3937/s | 🟡 B |
| Init_medium | init | 442 | 2358984 | 26660 | 2209334 | 424/s | 🔴 D |
| EnvBinding_10 | init | 20084 | 66737 | 589 | 33279 | 14984/s | 🟢 A |
| EnvBinding_100 | init | 9096 | 187289 | 3259 | 186174 | 5339/s | 🟡 B |
| LargeConfig_1k_access | get | 13427659 | 88 | 1 | 16 | 11363636/s | 🟢 A |
| TypeConv_str_to_int | get | 32387608 | 38 | 0 | 0 | 26315789/s | 🟢 A |
| TypeConv_str_to_bool | get | 77883691 | 16 | 0 | 0 | 62500000/s | 🟢 A |
| TypeConv_str_to_float | get | 28070372 | 40 | 0 | 0 | 25000000/s | 🟢 A |

## 性能分析


### 缓存性能分析

- **CacheHit_repeated**: 10 ns/op (A) - 重复访问同一键（缓存命中）
- **CacheHit_rotating**: 19 ns/op (A) - 轮换访问多个键

### 写入操作分析

- **Set_simple**: 4992 ns/op (A) - 简单键值设置
- **Set_nested**: 71167 ns/op (C) - 嵌套路径设置

### 并发性能分析

- **ConcurrentRead_1G**: 2 ns/op (A) - 8 协程并发读取
- **ConcurrentRead_4G**: 2 ns/op (A) - 32 协程并发读取
- **ConcurrentRead_8G**: 2 ns/op (A) - 64 协程并发读取
- **ConcurrentRead_16G**: 2 ns/op (A) - 128 协程并发读取
- **ConcurrentReadWrite_8R2W**: 106 ns/op (A) - 8读2写并发混合

### 初始化性能分析

- **Init_minimal**: 33080 ns/op (A) - minimal 配置初始化
- **Init_small**: 253997 ns/op (B) - small 配置初始化
- **Init_medium**: 2358984 ns/op (D) - medium 配置初始化
- **EnvBinding_10**: 66737 ns/op (A) - 绑定 10 个环境变量
- **EnvBinding_100**: 187289 ns/op (B) - 绑定 100 个环境变量

### 读取操作分析

- **GetString_simple**: 10 ns/op (A) - 简单字符串获取
- **GetInt_simple**: 11 ns/op (A) - 简单整数获取
- **GetBool_simple**: 11 ns/op (A) - 简单布尔值获取
- **GetFloat_simple**: 11 ns/op (A) - 简单浮点数获取
- **GetString_nested**: 11 ns/op (A) - 嵌套路径获取
- **GetAs_string**: 12 ns/op (A) - 泛型获取 string 类型
- **GetAs_int**: 12 ns/op (A) - 泛型获取 int 类型
- **GetAs_bool**: 12 ns/op (A) - 泛型获取 bool 类型
- **GetAs_float64**: 12 ns/op (A) - 泛型获取 float64 类型
- **GetAs_duration**: 46 ns/op (A) - 泛型获取 duration 类型
- **GetSliceAs_float64**: 208 ns/op (B) - 泛型切片获取
- **LargeConfig_1k_access**: 88 ns/op (A) - 1000节配置随机访问
- **TypeConv_str_to_int**: 38 ns/op (A) - 类型转换: str_to_int
- **TypeConv_str_to_bool**: 16 ns/op (A) - 类型转换: str_to_bool
- **TypeConv_str_to_float**: 40 ns/op (A) - 类型转换: str_to_float
