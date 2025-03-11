# QPS Counter API 文档

## 概述

本文档描述了QPS Counter服务提供的API接口。QPS Counter是一个高性能的QPS计数和限流服务，支持多种计数策略和自适应分片。

## 基础信息

- 基础URL: `http://localhost:8080`（可通过配置文件修改端口）
- 所有POST请求的Content-Type应为`application/json`

## 接口列表

### 1. 计数接口

**请求**:
```
POST /collect
```

**请求体**:
```json
{
  "count": 1
}
```

**参数说明**:
- `count`: 整数，表示要增加的计数值，默认为1

**响应**:
- 成功: HTTP 202 (Accepted)
- 限流: HTTP 429 (Too Many Requests)
- 服务关闭中: HTTP 503 (Service Unavailable)

### 2. 查询当前QPS

**请求**:
```
GET /qps
```

**响应**:
```json
{
  "qps": 1000
}
```

**参数说明**:
- `qps`: 整数，表示当前系统QPS

### 3. 获取系统状态

**请求**:
```
GET /stats
```

**响应**:
```json
{
  "qps": 1000,
  "limiter": {
    "rate": 10000,
    "burst_size": 20000,
    "current_tokens": 15000,
    "enabled": true,
    "rejected_count": 150,
    "total_count": 10000,
    "reject_rate": 0.015
  },
  "shutdown": {
    "status": "running",
    "active_requests": 5
  }
}
```

### 4. 设置限流器速率

**请求**:
```
POST /limiter/rate
```

**请求体**:
```json
{
  "rate": 5000
}
```

**参数说明**:
- `rate`: 整数，表示新的限流速率（每秒请求数）

**响应**:
```json
{
  "message": "限流速率已更新",
  "new_rate": 5000
}
```

### 5. 启用/禁用限流器

**请求**:
```
POST /limiter/toggle
```

**请求体**:
```json
{
  "enabled": false
}
```

**参数说明**:
- `enabled`: 布尔值，表示是否启用限流器

**响应**:
```json
{
  "message": "限流器状态已更新",
  "enabled": false
}
```

### 6. 健康检查

**请求**:
```
GET /healthz
```

**响应**:
- 成功: HTTP 200，响应体为 "ok"

### 7. Prometheus指标

**请求**:
```
GET /metrics
```

**响应**:
- 成功: HTTP 200，响应体为Prometheus格式的指标数据

## 指标说明

系统暴露以下Prometheus指标：

- `qps_counter_current_qps`: 当前系统QPS
- `qps_counter_memory_usage_bytes`: 当前内存使用量（字节）
- `qps_counter_cpu_usage_percent`: 当前CPU使用率
- `qps_counter_goroutines`: 当前goroutine数量
- `qps_counter_requests_total`: 处理的请求总数
- `qps_counter_request_duration_seconds`: 请求处理时间分布

## 错误处理

所有API错误响应都使用标准HTTP状态码，并在响应体中包含错误详情：

```json
{
  "error": "错误描述信息"
}
```

常见错误状态码：
- 400: 请求参数错误
- 429: 请求被限流
- 503: 服务正在关闭中