package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"runtime"
	"sync"
	"time"

	"github.com/mant7s/qps-counter/internal/counter"
)

// Metrics 提供系统监控指标收集和导出功能
type Metrics struct {
	counter       counter.Counter
	registry      *prometheus.Registry
	qpsGauge      prometheus.Gauge
	memoryGauge   prometheus.Gauge
	cpuGauge      prometheus.Gauge
	goroutineGauge prometheus.Gauge
	requestCounter prometheus.Counter
	requestLatency prometheus.Histogram
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewMetrics 创建一个新的指标收集器
func NewMetrics(counter counter.Counter) *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		counter:  counter,
		registry: reg,
		qpsGauge: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Name: "qps_counter_current_qps",
				Help: "当前系统QPS",
			},
		),
		memoryGauge: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Name: "qps_counter_memory_usage_bytes",
				Help: "当前内存使用量（字节）",
			},
		),
		cpuGauge: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Name: "qps_counter_cpu_usage_percent",
				Help: "当前CPU使用率",
			},
		),
		goroutineGauge: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Name: "qps_counter_goroutines",
				Help: "当前goroutine数量",
			},
		),
		requestCounter: promauto.With(reg).NewCounter(
			prometheus.CounterOpts{
				Name: "qps_counter_requests_total",
				Help: "处理的请求总数",
			},
		),
		requestLatency: promauto.With(reg).NewHistogram(
			prometheus.HistogramOpts{
				Name:    "qps_counter_request_duration_seconds",
				Help:    "请求处理时间分布",
				Buckets: prometheus.DefBuckets,
			},
		),
		stopChan: make(chan struct{}),
	}

	return m
}

// Start 启动指标收集
func (m *Metrics) Start(interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second // 默认5秒间隔
	}
	m.wg.Add(1)
	go m.collectMetrics(interval)
}

// Stop 停止指标收集
func (m *Metrics) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

// Registry 返回Prometheus注册表，用于HTTP处理程序
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// RecordRequest 记录一个请求
func (m *Metrics) RecordRequest() func() {
	m.requestCounter.Inc()
	start := time.Now()
	return func() {
		duration := time.Since(start).Seconds()
		m.requestLatency.Observe(duration)
	}
}

// collectMetrics 定期收集系统指标
func (m *Metrics) collectMetrics(interval time.Duration) {
	defer m.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var memStats runtime.MemStats

	for {
		select {
		case <-ticker.C:
			// 更新QPS指标
			m.qpsGauge.Set(float64(m.counter.CurrentQPS()))

			// 更新内存使用指标
			runtime.ReadMemStats(&memStats)
			m.memoryGauge.Set(float64(memStats.Alloc))

			// 更新goroutine数量
			m.goroutineGauge.Set(float64(runtime.NumGoroutine()))

		case <-m.stopChan:
			return
		}
	}
}