# QPS Counter

[![Go Report Card](https://goreportcard.com/badge/github.com/mant7s/qps-counter)](https://goreportcard.com/report/github.com/mant7s/qps-counter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mant7s/qps-counter)](https://github.com/mant7s/qps-counter)

High-precision QPS (Queries Per Second) statistics system, suitable for real-time request frequency statistics in high-concurrency scenarios. A high-performance counter implemented in Go language, supporting accurate statistics in million-level QPS scenarios.

*[English](README.en.md) | [中文](README.md)*

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

## 🏗 Architecture Design
```
+-------------------+     +-----------------------+
|   HTTP Endpoint   | ⇒  |  Adaptive Sharding    |
+-------------------+     +-----------------------+
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
| Engine Type | Concurrency | Avg Latency | P99 Latency | QPS     |
|-------------|------------|------------|------------|--------|
| Lock-Free   | 10k        | 1.2ms      | 3.5ms      | 950k   |
| Sharded     | 50k        | 3.8ms      | 9.2ms      | 4.2M   |

High-load scenario test results:
| Engine Type | Concurrency | Avg Latency | P99 Latency | QPS     |
|-------------|------------|------------|------------|--------|
| Lock-Free   | 100k       | 1.2ms      | 3.5ms      | 1.23M  |
| Sharded     | 500k       | 3.8ms      | 9.2ms      | 4.75M  |

## 🛡️ Health Check and Monitoring

### Health Check Endpoint
```http
GET /healthz
Response:
{
  "status": "OK"
}
```

## 🚀 Quick Start

### Installation
```bash
# Clone repository
$ git clone https://github.com/mant7s/qps-counter.git
$ cd qps-counter

# Copy configuration file
$ cp config/config.example.yaml config/config.yaml

# Build
$ make build

# Run
$ ./bin/qps-counter
```

### Docker Deployment
```bash
# Deploy with Docker
$ git clone https://github.com/mant7s/qps-counter.git
$ cd qps-counter
$ cp config/config.example.yaml config/config.yaml
$ cd deployments
$ docker-compose up -d --scale qps-counter=3

# Verify deployment
$ curl http://localhost:8080/healthz
```

## 📚 API Documentation

### Increment Counter
```http
POST /collect
Request body:
{
  "count": 1
}
Response: 202 Accepted
```

### Get Current QPS
```http
GET /qps
Response: 
{
  "qps": 12345
}
```

## 🔧 Development Guide

### Project Structure
```
├── cmd/          # Entry programs
├── config/       # Configuration files
├── internal/     # Internal packages
│   ├── api/      # API handlers
│   ├── config/   # Configuration management
│   ├── counter/  # Counter implementation
│   ├── limiter/  # Rate limiting component
│   ├── logger/   # Logging component
│   └── metrics/  # Monitoring metrics component
├── deployments/  # Deployment configurations
└── tests/        # Test code
```

### Running Tests
```bash
# Run unit tests
$ make test

# Run benchmark tests
$ make benchmark
```

## 🤝 Contribution Guidelines
1. Fork the project and create a branch
2. Add test cases
3. Submit a Pull Request
4. Follow Go code standards (use gofmt)

## 📄 License
MIT License

## 📞 Contact
- Project Maintainer: [mant7s](https://github.com/mant7s)
- Issue Reporting: [Issues](https://github.com/mant7s/qps-counter/issues)