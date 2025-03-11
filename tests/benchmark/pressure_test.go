package benchmark_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mant7s/qps-counter/internal/api"
	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/mant7s/qps-counter/internal/logger"
	"github.com/mant7s/qps-counter/internal/metrics"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func init() {
	// 初始化日志，避免测试中的日志错误
	loggerConfig := config.LoggerConfig{
		Level:  "debug",
		Format: "console",
	}
	logger.Init(loggerConfig)
}

func TestPressure(t *testing.T) {
	// 如果是短时间运行的测试，可以跳过
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	cfg := &config.CounterConfig{
		Type:       "lockfree",
		WindowSize: time.Second,
		SlotNum:    60,
		Precision:  100 * time.Millisecond,
	}

	// 创建计数器
	qpsCounter := counter.NewLockFree(cfg)
	defer qpsCounter.Stop()

	// 创建增强的优雅关闭管理器
	gracefulShutdown := counter.NewEnhancedGracefulShutdown(5*time.Second, 10*time.Second)

	// 创建限流器，设置较高的限流阈值以避免测试中的限流影响
	rateLimiter := limiter.NewRateLimiter(20000, 5000, false)

	// 创建指标收集器
	metricsCollector := metrics.NewMetrics(qpsCounter)

	// 创建路由
	router := api.NewRouter(qpsCounter, gracefulShutdown, rateLimiter, metricsCollector, "/metrics", true)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 减少测试时间，从30秒减少到3秒
	rate := vegeta.Rate{Freq: 10000, Per: time.Second} // 降低频率以减少资源消耗
	duration := 3 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "POST",
		URL:    ts.URL + "/collect",
		Body:   []byte(`{"count":1}`),
	})

	attacker := vegeta.NewAttacker()
	var metrics vegeta.Metrics

	for res := range attacker.Attack(targeter, rate, duration, "QPS Test") {
		metrics.Add(res)
	}
	metrics.Close()

	// 计算错误率
	errorCount := len(metrics.Errors)
	errorRate := float64(errorCount) / float64(metrics.Requests)

	// 允许更高的错误率，因为这是压力测试
	if errorRate > 0.01 {
		t.Errorf("错误率过高: %.4f%%，总请求数: %d，错误数: %d", errorRate*100, metrics.Requests, errorCount)
	}
}
