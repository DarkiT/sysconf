# Sysconf 基准测试报告

> 生成时间: 2026-04-27 22:14:43

## 系统信息

| 项目 | 值 |
|------|----|
| Go版本 | go1.26.2-X:jsonv2 |
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
| GetString_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetInt_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetBool_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetFloat_simple | get | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| GetString_nested | get | 99495763 | 12 | 0 | 0 | 83333333/s | 🟢 A |
| GetAs_string | get | 82262610 | 14 | 0 | 0 | 71428571/s | 🟢 A |
| GetAs_int | get | 82977669 | 14 | 0 | 0 | 71428571/s | 🟢 A |
| GetAs_bool | get | 82258826 | 15 | 0 | 0 | 66666667/s | 🟢 A |
| GetAs_float64 | get | 82246598 | 14 | 0 | 0 | 71428571/s | 🟢 A |
| GetAs_duration | get | 25239328 | 46 | 1 | 8 | 21739130/s | 🟢 A |
| GetSliceAs_float64 | get | 25554534 | 47 | 1 | 32 | 21276596/s | 🟢 A |
| CacheHit_repeated | cache | 100000000 | 11 | 0 | 0 | 90909091/s | 🟢 A |
| CacheHit_rotating | cache | 59705478 | 20 | 0 | 1 | 50000000/s | 🟢 A |
| Set_simple | set | 1513994 | 790 | 9 | 792 | 1265823/s | 🟢 A |
| Set_nested | set | 290173 | 4069 | 12 | 10043 | 245761/s | 🟢 A |
| ConcurrentRead_1G | concat | 384132572 | 3 | 0 | 0 | 333333333/s | 🟢 A |
| ConcurrentRead_4G | concat | 377780280 | 3 | 0 | 0 | 333333333/s | 🟢 A |
| ConcurrentRead_8G | concat | 383852178 | 3 | 0 | 0 | 333333333/s | 🟢 A |
| ConcurrentRead_16G | concat | 372811408 | 3 | 0 | 0 | 333333333/s | 🟢 A |
| ConcurrentReadWrite_8R2W | concat | 60566890 | 20 | 0 | 18 | 50000000/s | 🟢 A |
| Init_minimal | init | 56856 | 31826 | 110 | 13768 | 31421/s | 🟢 A |
| Init_small | init | 6888 | 220122 | 1771 | 140590 | 4543/s | 🟡 B |
| Init_medium | init | 745 | 1602236 | 16576 | 1452940 | 624/s | 🟠 C |
| EnvBinding_10 | init | 5169 | 231835 | 2851 | 237984 | 4313/s | 🟡 B |
| EnvBinding_100 | init | 3744 | 368990 | 4665 | 308311 | 2710/s | 🟡 B |
| LargeConfig_1k_access | get | 12315661 | 92 | 1 | 16 | 10869565/s | 🟢 A |
| TypeConv_str_to_int | get | 33494158 | 35 | 0 | 0 | 28571429/s | 🟢 A |
| TypeConv_str_to_bool | get | 59701382 | 19 | 0 | 0 | 52631579/s | 🟢 A |
| TypeConv_str_to_float | get | 26994631 | 45 | 0 | 0 | 22222222/s | 🟢 A |

## 性能分析


### 并发性能分析

- **ConcurrentRead_1G**: 3 ns/op (A) - 8 协程并发读取
- **ConcurrentRead_4G**: 3 ns/op (A) - 32 协程并发读取
- **ConcurrentRead_8G**: 3 ns/op (A) - 64 协程并发读取
- **ConcurrentRead_16G**: 3 ns/op (A) - 128 协程并发读取
- **ConcurrentReadWrite_8R2W**: 20 ns/op (A) - 8读2写并发混合

### 初始化性能分析

- **Init_minimal**: 31826 ns/op (A) - minimal 配置初始化
- **Init_small**: 220122 ns/op (B) - small 配置初始化
- **Init_medium**: 1602236 ns/op (C) - medium 配置初始化
- **EnvBinding_10**: 231835 ns/op (B) - 绑定 10 个环境变量
- **EnvBinding_100**: 368990 ns/op (B) - 绑定 100 个环境变量

### 读取操作分析

- **GetString_simple**: 11 ns/op (A) - 简单字符串获取
- **GetInt_simple**: 11 ns/op (A) - 简单整数获取
- **GetBool_simple**: 11 ns/op (A) - 简单布尔值获取
- **GetFloat_simple**: 11 ns/op (A) - 简单浮点数获取
- **GetString_nested**: 12 ns/op (A) - 嵌套路径获取
- **GetAs_string**: 14 ns/op (A) - 泛型获取 string 类型
- **GetAs_int**: 14 ns/op (A) - 泛型获取 int 类型
- **GetAs_bool**: 15 ns/op (A) - 泛型获取 bool 类型
- **GetAs_float64**: 14 ns/op (A) - 泛型获取 float64 类型
- **GetAs_duration**: 46 ns/op (A) - 泛型获取 duration 类型
- **GetSliceAs_float64**: 47 ns/op (A) - 泛型切片获取
- **LargeConfig_1k_access**: 92 ns/op (A) - 1000节配置随机访问
- **TypeConv_str_to_int**: 35 ns/op (A) - 类型转换: str_to_int
- **TypeConv_str_to_bool**: 19 ns/op (A) - 类型转换: str_to_bool
- **TypeConv_str_to_float**: 45 ns/op (A) - 类型转换: str_to_float

### 缓存性能分析

- **CacheHit_repeated**: 11 ns/op (A) - 重复访问同一键（缓存命中）
- **CacheHit_rotating**: 20 ns/op (A) - 轮换访问多个键

### 写入操作分析

- **Set_simple**: 790 ns/op (A) - 简单键值设置
- **Set_nested**: 4069 ns/op (A) - 嵌套路径设置
