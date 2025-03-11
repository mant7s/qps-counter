package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mant7s/qps-counter/internal/api"
	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/mant7s/qps-counter/internal/logger"
	"github.com/mant7s/qps-counter/internal/metrics"
	"github.com/stretchr/testify/assert"
)

func TestEnhancedAPIEndpoints(t *testing.T) {
	// 初始化日志器
	initTestLogger()

	// 初始化测试配置
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Port:        8080,
			ReadTimeout: 5 * time.Second,
		},
		Counter: config.CounterConfig{
			Type:       "sharded",
			WindowSize: 1 * time.Second,
			SlotNum:    10,
			Precision:  100 * time.Millisecond,
		},
		Logger: config.LoggerConfig{
			Level:  "info",
			Format: "console",
		},
	}

	qpsCounter := counter.NewCounter(&cfg.Counter)
	defer qpsCounter.Stop()

	// 创建增强的优雅关闭管理器
	gracefulShutdown := counter.NewEnhancedGracefulShutdown(1*time.Second, 2*time.Second)

	// 创建限流器
	rateLimiter := limiter.NewRateLimiter(100, 50, true)

	// 创建指标收集器
	metricsCollector := metrics.NewMetrics(qpsCounter)

	// 使用api.NewRouter创建测试路由，与实际应用保持一致
	router := api.NewRouter(qpsCounter, gracefulShutdown, rateLimiter, metricsCollector, "/metrics", true)

	// 设置测试模式
	gin.SetMode(gin.TestMode)

	t.Run("collect endpoint with rate limiter", func(t *testing.T) {
		// 测试正常请求
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/collect", strings.NewReader(`{"count":10}`))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusAccepted, w.Code)

		// 测试限流功能
		// 先将限流器速率设置为非常低的值
		rateLimiter.SetRate(1)

		// 消耗所有令牌
		for i := 0; i < 60; i++ {
			rateLimiter.Allow()
		}

		// 此时应该被限流
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/collect", strings.NewReader(`{"count":1}`))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		// 恢复限流器速率
		rateLimiter.SetRate(100)
	})

	// 添加短暂延迟，确保计数器有时间更新
	time.Sleep(200 * time.Millisecond)

	t.Run("query endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/qps", nil)

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		// 由于前面的测试可能会影响QPS值，这里我们只检查响应格式
		assert.Contains(t, w.Body.String(), "qps")
	})

	t.Run("stats endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/stats", nil)

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 验证响应包含预期的字段
		response := w.Body.String()
		assert.Contains(t, response, "qps")
		assert.Contains(t, response, "limiter")
		assert.Contains(t, response, "shutdown")
	})

	t.Run("limiter rate endpoint", func(t *testing.T) {
		// 测试设置限流器速率
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/limiter/rate", strings.NewReader(`{"rate":200}`))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "限流速率已更新")
		assert.Contains(t, w.Body.String(), "200")
	})

	t.Run("limiter toggle endpoint", func(t *testing.T) {
		// 测试禁用限流器
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/limiter/toggle", strings.NewReader(`{"enabled":false}`))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "限流器状态已更新")
		assert.Contains(t, w.Body.String(), "false")

		// 测试启用限流器
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/limiter/toggle", strings.NewReader(`{"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "限流器状态已更新")
		assert.Contains(t, w.Body.String(), "true")
	})

	t.Run("graceful shutdown test", func(t *testing.T) {
		// 创建一个新的关闭管理器用于测试
		testGS := counter.NewEnhancedGracefulShutdown(500*time.Millisecond, 1*time.Second)

		// 模拟一个长时间运行的请求
		testGS.StartRequest()

		// 在后台启动关闭流程
		go func() {
			testGS.Shutdown(context.Background())
		}()

		// 等待关闭开始
		time.Sleep(100 * time.Millisecond)

		// 尝试启动新请求，应该被拒绝
		assert.False(t, testGS.StartRequest(), "关闭过程中不应接受新请求")

		// 结束长时间运行的请求
		testGS.EndRequest()

		// 等待关闭完成
		time.Sleep(600 * time.Millisecond)

		// 验证关闭状态
		assert.Equal(t, "graceful_shutdown_complete", testGS.Status())
	})
}

// 初始化测试用的日志器
func initTestLogger() {
	// 创建一个测试日志配置
	logCfg := config.LoggerConfig{
		Level:  "info",
		Format: "console",
	}
	
	// 初始化logger包
	logger.Init(logCfg)
}