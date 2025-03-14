# QPS Counter

[![Go Report Card](https://goreportcard.com/badge/github.com/mant7s/qps-counter)](https://goreportcard.com/report/github.com/mant7s/qps-counter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mant7s/qps-counter)](https://github.com/mant7s/qps-counter)

High-precision QPS (Queries Per Second) statistics system, suitable for real-time request frequency statistics in high-concurrency scenarios. A high-performance counter implemented in Go language, supporting accurate statistics in million-level QPS scenarios.

*[中文](README.zh_CN.md)*

## ✨ Core Features
- 🚀 Dual-engine architecture (Lock-Free/Sharded), supporting million-level QPS real-time statistics
- 🔄 Intelligent sharding strategy (dynamic sharding based on CPU cores, 10-second interval QPS monitoring)
- ⚡ Time window sliding algorithm (1s window, 100ms precision)
- 🧠 Adaptive load balancing (automatically adjusts when QPS change rate exceeds 30%)
- 🛡️ Enhanced graceful shutdown mechanism (request integrity guarantee, timeout control, forced shutdown)
- 🔒 Token bucket rate limiting (dynamically adjustable rate, burst traffic support, adaptive rate limiting)
- 📊 Prometheus monitoring integration (QPS, memory, CPU, request latency metrics)
- ✅ Health check endpoint support (/healthz)
- 📈 Resource usage monitoring metrics (memory threshold adaptation, automatic shard adjustment)
- ⚙️ High-performance design (atomic operations, fine-grained locks, request counting and statistics)
- 🌐 HTTP server dual-mode support (standard net/http and high-performance fasthttp)

## 🏗 Architecture Design
```
+-------------------+     +-----------------------+
|   HTTP Server     | ⇒  |  Adaptive Sharding    |
| (net/http,fasthttp)|    +-----------------------+
+-------------------+     
      ↓                               ↓
+---------------+        +------------------------+
| Lock-Free Engine |     | Sharded Counter Cluster |
| (CAS Atomic Ops) |     | (Dynamic Sharding)     |
+---------------+        +------------------------+
                                ⇓
+------------------------------------------------+
|           Dynamic Sharding Manager             |
|  • 10s interval monitoring QPS change rate     |
|    (±30% triggers adjustment)                  |
|  • Auto-scaling shards (min: CPU cores,        |
|    max: CPU cores*8)                           |
|  • Memory usage monitoring (auto-adjusts shards |
|    to optimize memory usage)                   |
+------------------------------------------------+
                                ⇓
+------------------+  +------------------+  +------------------+
|  Rate Limiting   |  |    Monitoring    |  | Graceful Shutdown|
| (Token Bucket+  |  | (Prometheus      |  | (Request Integrity|
|  Adaptive)      |  |  Integration)    |  |  Guarantee)      |
+------------------+  +------------------+  +------------------+
```

## 🔍 Technical Implementation

### Lock-Free Engine
Lock-free counter based on atomic operations (CAS), suitable for medium traffic scenarios:
- Uses `atomic.Int64` to implement lock-free counting, avoiding lock contention in high concurrency
- Time window sliding algorithm, ensuring statistical accuracy and real-time performance
- Automatic cleaning of expired data to prevent memory leaks

### Sharded Counter
High-performance counter with sharding design, suitable for ultra-high concurrency scenarios:
- Automatic sharding based on CPU cores, default is `runtime.NumCPU() * 4`
- Fine-grained lock design, independent lock for each time slot, maximizing parallelism
- Hash algorithm ensures requests are evenly distributed across shards

### Adaptive Sharding Management
- Real-time monitoring of QPS change rate, triggering shard adjustment when changes exceed ±30%
- Increases shard count by 50% during growth, reduces by 30% during decline
- Shard count range controlled between CPU cores and CPU cores*8, avoiding resource waste
- Memory usage monitoring, automatically adjusts shards when approaching threshold
- Intelligent adjustment based on combined QPS change rate (60%) and memory usage (40%)

### Token Bucket Rate Limiter
- Efficient rate limiting based on token bucket algorithm, supporting burst traffic
- Dynamic rate adjustment to adapt to system load changes
- Adaptive rate limiting mode that automatically adjusts parameters based on system resource usage
- Precise tracking of rejected requests with monitoring metrics

### Monitoring Metrics System
- Prometheus integration providing rich system operational metrics
- Real-time monitoring of QPS, memory usage, CPU utilization, and Goroutine count
- Request latency distribution statistics supporting P99 performance analysis
- Configurable metrics collection interval optimizing performance and precision balance

### Enhanced Graceful Shutdown
- Request integrity guarantee ensuring in-progress requests complete processing
- Multi-level timeout control with soft and hard timeout mechanisms
- Real-time status reporting providing shutdown process observability
- Forced shutdown protection preventing system from hanging indefinitely

## ⚙️ Configuration
```yaml
server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s
  server_type: fasthttp  # HTTP server type (standard/fasthttp)

counter:
  type: "lockfree"     # Counter type (lockfree/sharded)
  window_size: 1s      # Statistical time window
  slot_num: 10         # Window shard count
  precision: 100ms     # Statistical precision

limiter:
  enabled: true        # Enable rate limiting
  rate: 1000000        # Requests allowed per second
  burst: 10000         # Burst capacity
  adaptive: true       # Enable adaptive rate limiting

metrics:
  enabled: true        # Enable metrics collection
  interval: 5s         # Metrics collection interval
  endpoint: "/metrics" # Metrics exposure endpoint

shutdown:
  timeout: 30s         # Graceful shutdown timeout
  max_wait: 60s        # Maximum wait time

logger:
  level: info
  format: json
  file_path: "/var/log/qps-counter/app.log"
  max_size: 100
  max_backups: 3
  max_age: 7
```

## 📈 Performance Metrics
| Server Type | Concurrency | Avg Latency | P99 Latency | QPS     |
|------------|------------|------------|------------|--------|
| standard   | 10k        | 1.8ms      | 4.5ms      | 850k   |
| fasthttp   | 10k        | 1.2ms      | 3.5ms      | 950k   |

High-load scenario test results:
| Server Type | Concurrency | Avg Latency | P99 Latency | QPS     |
|------------|------------|------------|------------|--------|
| standard   | 100k       | 2.5ms      | 6.5ms      | 1.05M  |
| fasthttp   | 100k       | 1.2ms      | 3.5ms      | 1.23M  |

## 🚀 Quick Start

### Installation
```bash
go get github.com/mant7s/qps-counter
```

### Basic Usage
```go
package main

import (
    "github.com/mant7s/qps-counter/counter"
    "log"
)

func main() {
    // Create counter instance
    cfg := counter.DefaultConfig()
    counter, err := counter.NewCounter(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Increment counter
    counter.Increment()

    // Get current QPS
    qps := counter.GetQPS()
    log.Printf("Current QPS: %d", qps)
}
```

## 📊 Monitoring Metrics

The system exposes Prometheus-format monitoring metrics through the `/metrics` endpoint:

- `qps_counter_requests_total`: Total request count
- `qps_counter_current_qps`: Current QPS value
- `qps_counter_memory_usage_bytes`: Memory usage
- `qps_counter_cpu_usage_percent`: CPU usage
- `qps_counter_goroutines`: Goroutine count
- `qps_counter_request_duration_seconds`: Request processing time distribution

## 🔍 API Documentation

For detailed API documentation, please refer to [API Documentation](docs/api.md).

## 🛠 Development Guide

### Requirements
- Go 1.18+
- Make

### Local Development
1. Clone repository
```bash
git clone https://github.com/mant7s/qps-counter.git
cd qps-counter
```

2. Install dependencies
```bash
go mod download
```

3. Run tests
```bash
make test
```

4. Build project
```bash
make build
```

## 🤝 Contributing Guide

Contributions are welcome! Please ensure:

1. Fork the project and create a feature branch
2. Add test cases
3. Run `make test` before submitting PR to ensure tests pass
4. Follow the project's code standards

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## 📞 Contact

- Author: Mant7s
- GitHub: [@mant7s](https://github.com/mant7s)