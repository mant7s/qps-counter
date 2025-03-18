# QPS Counter

[![Go Report Card](https://goreportcard.com/badge/github.com/mant7s/qps-counter)](https://goreportcard.com/report/github.com/mant7s/qps-counter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mant7s/qps-counter)](https://github.com/mant7s/qps-counter)

QPS统计系统，用于并发场景下的实时请求频率统计。基于Go语言实现的计数器，支持大规模QPS场景下的统计。

*[English](README.md)*

## ✨ 核心特性
- 🚀 双引擎架构（Lock-Free/Sharded），支持大规模QPS实时统计
- 🔄 分片策略（基于CPU核心数的动态分片，10秒间隔QPS监控）
- ⚡ 时间窗口滑动算法（1s窗口，100ms粒度）
- 🧠 负载均衡（QPS变化率超30%时调整）
- 🛡️ 完善的关闭机制（请求完整性保障，超时控制，强制关闭）
- 🔒 令牌桶限流保护（可调整速率，支持突发流量，动态限流）
- 📊 Prometheus监控集成（QPS、内存、CPU、请求延迟等指标）
- ✅ 健康检查端点支持（/healthz）
- 📈 资源使用监控（内存阈值调整，分片数量调整）
- ⚙️ 性能优化（原子操作、细粒度锁、请求计数与统计）
- 🌐 HTTP服务器双模式支持（标准net/http和fasthttp）

## 🏗 架构设计
```
+-------------------+     +-----------------------+
|   HTTP Server     | ⇒  |  Dynamic Sharding     |
| (net/http,fasthttp)|    +-----------------------+
+-------------------+     
      ↓                               ↓
+---------------+        +------------------------+
| Lock-Free引擎 |        | Sharded计数器集群       |
| (CAS原子操作)  |        | (动态分片)              |
+---------------+        +------------------------+
                                ⇓
+------------------------------------------------+
|           分片管理器                           |
|  • 10秒间隔监控QPS变化率（±30%触发调整）        |
|  • 分片数调整（最小CPU核心数，最大CPU核心数*8）   |
|  • 内存使用监控（调整分片以控制内存使用）         |
+------------------------------------------------+
                                ⇓
+------------------+  +------------------+  +------------------+
|    限流保护层    |  |    监控指标层    |  |   关闭机制       |
| (令牌桶+动态)    |  | (Prometheus集成) |  | (请求完整性保障) |
+------------------+  +------------------+  +------------------+
```

## 🔍 技术实现

### Lock-Free引擎
基于原子操作（CAS）实现的无锁计数器，适用于中等流量场景：
- 使用`atomic.Int64`实现无锁计数，减少并发下的锁竞争
- 时间窗口滑动算法，保证统计实时性
- 自动清理过期数据，避免内存泄漏

### Sharded计数器
分片设计的计数器，适用于大规模并发场景：
- 基于CPU核心数的分片，默认为`runtime.NumCPU() * 4`
- 细粒度锁设计，每个时间槽独立锁，提高并行性
- 哈希算法分散请求到各分片

### 分片管理
- 监控QPS变化率，当变化超过±30%时调整分片
- 增长时增加50%分片数，下降时减少30%分片数
- 分片数范围在CPU核心数到CPU核心数*8之间
- 内存使用监控，根据阈值调整分片数量
- 结合QPS变化率(60%)和内存使用情况(40%)调整

### 令牌桶限流器
- 基于令牌桶算法实现限流，处理突发流量
- 可调整限流速率，适应系统负载
- 动态限流模式，根据系统资源使用调整参数
- 统计被拒绝请求，提供限流指标

### 监控指标系统
- 集成Prometheus，提供系统运行指标
- 监控QPS、内存使用、CPU使用率、Goroutine数量
- 请求延迟分布统计，支持P99等性能分析
- 可配置的指标收集间隔

### 关闭机制
- 请求完整性保障，确保进行中的请求完成处理
- 多级超时控制，包括软超时和硬超时
- 状态报告，提供关闭过程的可观测性
- 强制关闭保护，防止系统无法退出

## ⚙️ 配置说明
```yaml
server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s
  server_type: fasthttp  # HTTP服务器类型（standard/fasthttp）

counter:
  type: "lockfree"     # 计数器类型（lockfree/sharded）
  window_size: 1s      # 统计时间窗口
  slot_num: 10         # 窗口分片数量
  precision: 100ms     # 统计精度
```