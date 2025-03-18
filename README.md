# QPS Counter

[![Go Report Card](https://goreportcard.com/badge/github.com/mant7s/qps-counter)](https://goreportcard.com/report/github.com/mant7s/qps-counter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mant7s/qps-counter)](https://github.com/mant7s/qps-counter)

QPS (Queries Per Second) statistics system, designed for real-time request frequency statistics in concurrent scenarios. A counter implemented in Go language, supporting statistics in large-scale QPS scenarios.

*[中文](README.zh_CN.md)*

## ✨ Core Features
- 🚀 Dual-engine architecture (Lock-Free/Sharded), supporting large-scale QPS real-time statistics
- 🔄 Sharding strategy (dynamic sharding based on CPU cores, 10-second interval QPS monitoring)
- ⚡ Time window sliding algorithm (1s window, 100ms granularity)
- 🧠 Load balancing (adjusts when QPS change rate exceeds 30%)
- 🛡️ Comprehensive shutdown mechanism (request integrity guarantee, timeout control, forced shutdown)
- 🔒 Token bucket rate limiting (adjustable rate, burst traffic support, dynamic rate limiting)
- 📊 Prometheus monitoring integration (QPS, memory, CPU, request latency metrics)
- ✅ Health check endpoint support (/healthz)
- 📈 Resource usage monitoring (memory threshold adjustment, shard count adjustment)
- ⚙️ Performance optimization (atomic operations, fine-grained locks, request counting and statistics)
- 🌐 HTTP server dual-mode support (standard net/http and fasthttp)

## 🏗 Architecture Design
```
+-------------------+     +-----------------------+
|   HTTP Server     | ⇒  |  Dynamic Sharding     |
| (net/http,fasthttp)|    +-----------------------+
+-------------------+     
      ↓                               ↓
+---------------+        +------------------------+
| Lock-Free Engine |     | Sharded Counter Cluster |
| (CAS Atomic Ops) |     | (Dynamic Sharding)     |
+---------------+        +------------------------+
                                ⇓
+------------------------------------------------+
|           Sharding Manager                     |
|  • 10s interval monitoring QPS change rate     |
|    (±30% triggers adjustment)                  |
|  • Shard count adjustment (min: CPU cores,     |
|    max: CPU cores*8)                           |
|  • Memory usage monitoring (adjusts shards     |
|    to control memory usage)                    |
+------------------------------------------------+
                                ⇓
+------------------+  +------------------+  +------------------+
|  Rate Limiting   |  |    Monitoring    |  |    Shutdown     |
| (Token Bucket+  |  | (Prometheus      |  | (Request Integrity|
|  Dynamic)       |  |  Integration)    |  |  Guarantee)      |
+------------------+  +------------------+  +------------------+
```

## 🔍 Technical Implementation

### Lock-Free Engine
Lock-free counter based on atomic operations (CAS), suitable for medium traffic scenarios:
- Uses `atomic.Int64` to implement lock-free counting, reducing lock contention in concurrent scenarios
- Time window sliding algorithm, ensuring statistical timeliness
- Automatic cleaning of expired data to prevent memory leaks

### Sharded Counter
Sharded counter design, suitable for large-scale concurrent scenarios:
- Sharding based on CPU cores, default is `runtime.NumCPU() * 4`
- Fine-grained lock design, independent lock for each time slot, improving parallelism
- Hash algorithm distributes requests across shards

### Sharding Management
- Monitors QPS change rate, adjusts shards when changes exceed ±30%
- Increases shard count by 50% during growth, reduces by 30% during decline
- Shard count range between CPU cores and CPU cores*8
- Memory usage monitoring, adjusts shard count based on threshold
- Combined adjustment based on QPS change rate (60%) and memory usage (40%)

### Token Bucket Rate Limiter
- Rate limiting based on token bucket algorithm, handling burst traffic
- Adjustable rate to adapt to system load
- Dynamic rate limiting mode, adjusting parameters based on system resource usage
- Tracks rejected requests with monitoring metrics

### Monitoring Metrics System
- Prometheus integration providing system operational metrics
- Monitors QPS, memory usage, CPU utilization, and Goroutine count
- Request latency distribution statistics supporting P99 performance analysis
- Configurable metrics collection interval

### Shutdown Mechanism
- Request integrity guarantee ensuring in-progress requests complete processing
- Multi-level timeout control with soft and hard timeout mechanisms
- Status reporting providing shutdown process observability
- Forced shutdown protection preventing system from hanging

## ⚙️ Configuration
```yaml
server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s
  server_type: fasthttp  # HTTP server type (standard/fasthttp)

counter:
  type: "lockfree"     # Counter type (lockfree/sharded)
  window_size: 1s      # Statistics time window
  slot_num: 10         # Window slot count
  precision: 100ms     # Statistics granularity
```