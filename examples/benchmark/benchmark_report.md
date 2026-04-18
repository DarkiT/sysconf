# Sysconf 基准测试报告

> 生成时间: 2026-01-15 01:33:47

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
| GetString_simple | get | 103529637 | 10 | 0 | 0 | 100000000/s | 🟢 A |
| GetInt_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetBool_simple | get | 100000000 | 10 | 0 | 0 | 100000000/s | 🟢 A |
| GetFloat_simple | get | 100000000 | 10 | 0 | 0 | 100000000/s | 🟢 A |
| GetString_nested | get | 93080654 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetAs_string | get | 93785469 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_int | get | 96655634 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_bool | get | 97846536 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_float64 | get | 71731405 | 17 | 0 | 0 | 58823529/s | 🟢 A |
| GetAs_duration | get | 25847638 | 45 | 1 | 8 | 22222222/s | 🟢 A |
| GetSliceAs_float64 | get | 5808626 | 208 | 4 | 112 | 4807692/s | 🟡 B |
| CacheHit_repeated | cache | 100000000 | 10 | 0 | 0 | 100000000/s | 🟢 A |
| CacheHit_rotating | cache | 58764171 | 19 | 0 | 1 | 52631579/s | 🟢 A |
| Set_simple | set | 875989 | 4721 | 27 | 2504 | 211820/s | 🟢 A |
| Set_nested | set | 24570 | 68431 | 613 | 83670 | 14613/s | 🟠 C |
| ConcurrentRead_1G | concat | 414149607 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentRead_4G | concat | 471724846 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentRead_8G | concat | 511091340 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentRead_16G | concat | 526680048 | 2 | 0 | 0 | 500000000/s | 🟢 A |
| ConcurrentReadWrite_8R2W | concat | 35213611 | 100 | 0 | 43 | 10000000/s | 🟢 A |
| Init_minimal | init | 35185 | 37297 | 133 | 16320 | 26812/s | 🟢 A |
| Init_small | init | 4201 | 255648 | 2812 | 196966 | 3912/s | 🟡 B |
| Init_medium | init | 582 | 2317384 | 26689 | 2212892 | 432/s | 🔴 D |
| EnvBinding_10 | init | 6351 | 263286 | 4119 | 314429 | 3798/s | 🟡 B |
| EnvBinding_100 | init | 3978 | 373069 | 6733 | 407882 | 2680/s | 🟡 B |
| LargeConfig_1k_access | get | 14209602 | 91 | 1 | 16 | 10989011/s | 🟢 A |
| TypeConv_str_to_int | get | 32557011 | 35 | 0 | 0 | 28571429/s | 🟢 A |
| TypeConv_str_to_bool | get | 78079940 | 16 | 0 | 0 | 62500000/s | 🟢 A |
| TypeConv_str_to_float | get | 30587500 | 40 | 0 | 0 | 25000000/s | 🟢 A |

## 性能分析


### 并发性能分析

- **ConcurrentRead_1G**: 2 ns/op (A) - 8 协程并发读取
- **ConcurrentRead_4G**: 2 ns/op (A) - 32 协程并发读取
- **ConcurrentRead_8G**: 2 ns/op (A) - 64 协程并发读取
- **ConcurrentRead_16G**: 2 ns/op (A) - 128 协程并发读取
- **ConcurrentReadWrite_8R2W**: 100 ns/op (A) - 8读2写并发混合

### 初始化性能分析

- **Init_minimal**: 37297 ns/op (A) - minimal 配置初始化
- **Init_small**: 255648 ns/op (B) - small 配置初始化
- **Init_medium**: 2317384 ns/op (D) - medium 配置初始化
- **EnvBinding_10**: 263286 ns/op (B) - 绑定 10 个环境变量
- **EnvBinding_100**: 373069 ns/op (B) - 绑定 100 个环境变量

### 读取操作分析

- **GetString_simple**: 10 ns/op (A) - 简单字符串获取
- **GetInt_simple**: 11 ns/op (A) - 简单整数获取
- **GetBool_simple**: 10 ns/op (A) - 简单布尔值获取
- **GetFloat_simple**: 10 ns/op (A) - 简单浮点数获取
- **GetString_nested**: 11 ns/op (A) - 嵌套路径获取
- **GetAs_string**: 12 ns/op (A) - 泛型获取 string 类型
- **GetAs_int**: 12 ns/op (A) - 泛型获取 int 类型
- **GetAs_bool**: 12 ns/op (A) - 泛型获取 bool 类型
- **GetAs_float64**: 17 ns/op (A) - 泛型获取 float64 类型
- **GetAs_duration**: 45 ns/op (A) - 泛型获取 duration 类型
- **GetSliceAs_float64**: 208 ns/op (B) - 泛型切片获取
- **LargeConfig_1k_access**: 91 ns/op (A) - 1000节配置随机访问
- **TypeConv_str_to_int**: 35 ns/op (A) - 类型转换: str_to_int
- **TypeConv_str_to_bool**: 16 ns/op (A) - 类型转换: str_to_bool
- **TypeConv_str_to_float**: 40 ns/op (A) - 类型转换: str_to_float

### 缓存性能分析

- **CacheHit_repeated**: 10 ns/op (A) - 重复访问同一键（缓存命中）
- **CacheHit_rotating**: 19 ns/op (A) - 轮换访问多个键

### 写入操作分析

- **Set_simple**: 4721 ns/op (A) - 简单键值设置
- **Set_nested**: 68431 ns/op (C) - 嵌套路径设置
