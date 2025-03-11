package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/mant7s/qps-counter/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(counter counter.Counter, gracefulShutdown *counter.EnhancedGracefulShutdown, rateLimiter *limiter.RateLimiter, metricsCollector *metrics.Metrics, metricsEndpoint string, metricsEnabled bool) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	handler := NewHandler(counter, gracefulShutdown, rateLimiter)
	router.POST("/collect", handler.Collect)
	router.GET("/qps", handler.Query)
	router.GET("/stats", handler.GetStats)
	router.POST("/limiter/rate", handler.SetLimiterRate)
	router.POST("/limiter/toggle", handler.ToggleLimiter)
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 添加Prometheus指标暴露端点
	if metricsCollector != nil && metricsEnabled {
		if metricsEndpoint == "" {
			metricsEndpoint = "/metrics"
		}
		router.GET(metricsEndpoint, gin.WrapH(promhttp.HandlerFor(metricsCollector.Registry(), promhttp.HandlerOpts{})))
	}

	return router
}
