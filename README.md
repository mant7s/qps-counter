# QPS Counter

[![Go Report Card](https://goreportcard.com/badge/github.com/mant7s/qps-counter)](https://goreportcard.com/report/github.com/mant7s/qps-counter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mant7s/qps-counter)](https://github.com/mant7s/qps-counter)

高精度QPS统计系统，适用于高并发场景的实时请求频率统计。基于Go语言实现的高性能计数器，支持百万级QPS场景下的精确统计。

*[中文](README.md) | [English](README.en.md)*

## ✨ 核心特性
- 🚀 双引擎架构（Lock-Free/Sharded），支持百万级QPS实时统计
- 🔄 智能分片策略（基于CPU核心数的动态分片，10秒间隔QPS监控）
- ⚡ 时间窗口滑动算法（1s窗口，100ms精度）
- 🧠 自适应负载均衡（QPS变化率超30%自动调整）
- 🛡️ 增强的优雅关闭机制（请求完整性保障，超时控制，强制关闭）
- 🔒 令牌桶限流保护（可动态调整速率，支持突发流量，自适应限流）
- 📊 Prometheus监控集成（QPS、内存、CPU、请求延迟等指标）
- ✅ 健康检查端点支持（/healthz）
- 📈 资源使用监控指标（内存阈值自适应，自动分片调整）
- ⚙️ 高性能设计（原子操作、细粒度锁、请求计数与统计）
- 🌐 HTTP服务器双模式支持（标准net/http和高性能fasthttp）

## 🏗 架构设计
```
+-------------------+     +-----------------------+
|   HTTP Server     | ⇒  |  Adaptive Sharding    |
| (net/http,fasthttp)|    +-----------------------+
+-------------------+     
      ↓                               ↓
+---------------+        +------------------------+
| Lock-Free引擎 |        | Sharded计数器集群       |
| (CAS原子操作)  |        | (动态分片)              |
+---------------+        +------------------------+
                                ⇓
+------------------------------------------------+
|           动态分片管理器                       |
|  • 10秒间隔监控QPS变化率（±30%触发调整）        |
|  • 分片数自动伸缩（最小CPU核心数，最大CPU核心数*8）|
|  • 内存使用监控（自动调整分片以优化内存使用）     |
+------------------------------------------------+
                                ⇓
+------------------+  +------------------+  +------------------+
|    限流保护层    |  |    监控指标层    |  |   优雅关闭机制   |
| (令牌桶+自适应)  |  | (Prometheus集成) |  | (请求完整性保障) |
+------------------+  +------------------+  +------------------+
```

## 🔍 技术实现

### Lock-Free引擎
基于原子操作（CAS）实现的无锁计数器，适用于中等流量场景：
- 使用`atomic.Int64`实现无锁计数，避免高并发下的锁竞争
- 时间窗口滑动算法，保证统计精度和实时性
- 自动清理过期数据，避免内存泄漏

### Sharded计数器
分片设计的高性能计数器，适用于超高并发场景：
- 基于CPU核心数的自动分片，默认为`runtime.NumCPU() * 4`
- 细粒度锁设计，每个时间槽独立锁，最大化并行性
- 哈希算法确保请求均匀分布到各分片

### 自适应分片管理
- 实时监控QPS变化率，当变化超过±30%时触发分片调整
- 增长时增加50%分片数，下降时减少30%分片数
- 分片数范围控制在CPU核心数到CPU核心数*8之间，避免资源浪费
- 内存使用监控，当接近阈值时自动调整分片数量
- 综合考量QPS变化率(60%)和内存使用情况(40%)进行智能调整

### 令牌桶限流器
- 基于令牌桶算法实现高效限流，支持突发流量处理
- 动态调整限流速率，适应系统负载变化
- 自适应限流模式，根据系统资源使用情况自动调整限流参数
- 精确统计被拒绝请求，提供限流指标监控

### 监控指标系统
- 集成Prometheus，提供丰富的系统运行指标
- 实时监控QPS、内存使用、CPU使用率、Goroutine数量
- 请求延迟分布统计，支持P99等性能分析
- 可配置的指标收集间隔，优化性能与精度平衡

### 增强的优雅关闭机制
- 请求完整性保障，确保进行中的请求能够完成处理
- 多级超时控制，包括软超时和硬超时机制
- 实时状态报告，提供关闭过程的可观测性
- 强制关闭保护，防止系统长时间无法退出

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

limiter:
  enabled: true        # 是否启用限流
  rate: 1000000        # 每秒允许的请求数
  burst: 10000         # 突发请求容量
  adaptive: true       # 是否启用自适应限流

metrics:
  enabled: true        # 是否启用指标收集
  interval: 5s         # 指标收集间隔
  endpoint: "/metrics" # 指标暴露端点

shutdown:
  timeout: 30s         # 优雅关闭超时时间
  max_wait: 60s        # 最大等待时间

logger:
  level: info
  format: json
  file_path: "/var/log/qps-counter/app.log"
  max_size: 100
  max_backups: 3
  max_age: 7
```

## 📈 性能指标
| 服务器类型  | 并发量 | 平均延迟 | P99延迟 | QPS     |
|------------|--------|---------|--------|--------|
| standard   | 10k    | 1.8ms   | 4.5ms  | 850k   |
| fasthttp   | 10k    | 1.2ms   | 3.5ms  | 950k   |

高负载场景测试结果：
| 服务器类型  | 并发量 | 平均延迟 | P99延迟 | QPS     |
|------------|--------|---------|--------|--------|
| standard   | 100k   | 2.5ms   | 6.5ms  | 1.05M  |
| fasthttp   | 100k   | 1.2ms   | 3.5ms  | 1.23M  |

## 🚀 快速开始

### 安装
```bash
go get github.com/mant7s/qps-counter
```

### 基本使用
```go
package main

import (
    "github.com/mant7s/qps-counter/counter"
    "log"
)

func main() {
    // 创建计数器实例
    cfg := counter.DefaultConfig()
    counter, err := counter.NewCounter(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // 增加计数
    counter.Increment()

    // 获取当前QPS
    qps := counter.GetQPS()
    log.Printf("Current QPS: %d", qps)
}
```

## 📊 监控指标

系统通过`/metrics`端点暴露Prometheus格式的监控指标：

- `qps_counter_requests_total`: 总请求计数
- `qps_counter_current_qps`: 当前QPS值
- `qps_counter_memory_usage_bytes`: 内存使用量
- `qps_counter_cpu_usage_percent`: CPU使用率
- `qps_counter_goroutines`: Goroutine数量
- `qps_counter_request_duration_seconds`: 请求处理时间分布

## 🔍 API文档

详细的API文档请参考[API文档](docs/api.md)。

## 🛠 开发指南

### 环境要求
- Go 1.18+
- Make

### 本地开发
1. 克隆仓库
```bash
git clone https://github.com/mant7s/qps-counter.git
cd qps-counter
```

2. 安装依赖
```bash
go mod download
```

3. 运行测试
```bash
make test
```

4. 构建项目
```bash
make build
```

## 🤝 贡献指南

欢迎贡献代码！请确保：

1. Fork项目并创建特性分支
2. 添加测试用例
3. 提交PR前运行`make test`确保测试通过
4. 遵循项目的代码规范

## 📄 许可证

本项目采用MIT许可证 - 详见[LICENSE](LICENSE)文件

## 📞 联系方式

- 作者：Mant7s
- GitHub：[@mant7s](https://github.com/mant7s)