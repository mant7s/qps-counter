package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mant7s/qps-counter/internal/api"
	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/stretchr/testify/assert"
)

func TestAPIEndpoints(t *testing.T) {
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
	}

	qpsCounter := counter.NewCounter(&cfg.Counter)
	defer qpsCounter.Stop()
	
	// 创建优雅关闭管理器用于测试
	gracefulShutdown := counter.NewGracefulShutdown(5 * time.Second)
	
	// 使用api.NewRouter创建测试路由，与实际应用保持一致
	router := api.NewRouter(qpsCounter, gracefulShutdown)

	// 设置测试模式
	gin.SetMode(gin.TestMode)

	t.Run("collect endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/collect", strings.NewReader(`{"count":10}`))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	// 添加短暂延迟，确保计数器有时间更新
	time.Sleep(200 * time.Millisecond)

	t.Run("query endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/qps", nil)

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"qps":10}`, w.Body.String())
	})
}
