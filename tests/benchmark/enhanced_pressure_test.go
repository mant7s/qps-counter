package benchmark_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mant7s/qps-counter/internal/api"
	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/mant7s/qps-counter/internal/logger"
	"github.com/mant7s/qps-counter/internal/metrics"
	"github.com/stretchr/testify/assert"
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

func TestEnhancedPressure(t *testing.T) {
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

	// 创建限流器
	rateLimiter := limiter.NewRateLimiter(20000, 5000, true)

	// 创建指标收集器
	metricsCollector := metrics.NewMetrics(qpsCounter)

	// 创建路由
	router := api.NewRouter(qpsCounter, gracefulShutdown, rateLimiter, metricsCollector, "/metrics", true)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 测试不同场景
	testCases := []struct {
		name      string
		rate      int
		duration  time.Duration
		limitRate int64
		enabled   bool
	}{
		{"无限流", 10000, 3 * time.Second, 20000, false},
		{"低限流", 10000, 3 * time.Second, 5000, true},
		{"高限流", 10000, 3 * time.Second, 15000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 配置限流器
			rateLimiter.SetRate(tc.limitRate)
			rateLimiter.SetEnabled(tc.enabled)

			// 配置压测参数
			rate := vegeta.Rate{Freq: tc.rate, Per: time.Second}
			duration := tc.duration
			targeter := vegeta.NewStaticTargeter(vegeta.Target{
				Method: "POST",
				URL:    ts.URL + "/collect",
				Body:   []byte(`{"count":1}`),
			})

			attacker := vegeta.NewAttacker()
			var metrics vegeta.Metrics

			// 执行压测
			for res := range attacker.Attack(targeter, rate, duration, tc.name) {
				metrics.Add(res)
			}
			metrics.Close()

			// 计算错误率和成功率
			errorCount := len(metrics.Errors)
			errorRate := float64(errorCount) / float64(metrics.Requests)
			successRate := 1 - errorRate

			// 输出测试结果
			t.Logf("%s - 总请求数: %d, 成功率: %.2f%%, 错误数: %d", 
				tc.name, metrics.Requests, successRate*100, errorCount)
			t.Logf("平均响应时间: %s, 99%%响应时间: %s", 
				metrics.Latencies.Mean, metrics.Latencies.P99)

			// 验证限流是否生效
			if tc.enabled && tc.limitRate < int64(tc.rate) {
				// 如果启用了限流且限流速率小于请求速率，应该有一定比例的请求被拒绝
				assert.Greater(t, errorRate, 0.0, "限流应该导致一些请求被拒绝")
			}

			// 获取系统状态
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/stats", nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
			t.Logf("系统状态: %s", w.Body.String())

			// 等待系统恢复
			time.Sleep(1 * time.Second)
		})
	}

	// 测试优雅关闭
	t.Run("优雅关闭测试", func(t *testing.T) {
		// 创建一个新的测试环境
		testGS := counter.NewEnhancedGracefulShutdown(1*time.Second, 2*time.Second)
		testCounter := counter.NewLockFree(cfg)
		testLimiter := limiter.NewRateLimiter(10000, 2000, true)
		// 创建指标收集器
		testMetrics := metrics.NewMetrics(testCounter)
		testRouter := api.NewRouter(testCounter, testGS, testLimiter, testMetrics, "/metrics", true)
		testServer := httptest.NewServer(testRouter)
		defer testServer.Close()
		defer testCounter.Stop()

		// 启动一些请求
		rate := vegeta.Rate{Freq: 1000, Per: time.Second}
		duration := 500 * time.Millisecond
		targeter := vegeta.NewStaticTargeter(vegeta.Target{
			Method: "POST",
			URL:    testServer.URL + "/collect",
			Body:   []byte(`{"count":1}`),
		})

		// 启动压测
		attacker := vegeta.NewAttacker()
		results := attacker.Attack(targeter, rate, duration, "shutdown-test")

		// 在请求进行中启动关闭
		time.Sleep(200 * time.Millisecond)
		go testGS.Shutdown(context.Background())

		// 收集结果
		var metrics vegeta.Metrics
		for res := range results {
			metrics.Add(res)
		}
		metrics.Close()

		// 等待关闭完成
		time.Sleep(1500 * time.Millisecond)

		// 验证关闭状态
		assert.Contains(t, []string{"graceful_shutdown_complete", "delayed_shutdown_complete"}, testGS.Status())
		assert.False(t, testGS.IsForceShutdown())

		// 尝试发送新请求，应该被拒绝
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/collect", nil)
		testRouter.ServeHTTP(w, req)
		assert.Equal(t, 503, w.Code) // 服务不可用
	})
}