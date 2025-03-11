package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"net/http"
)

type QPSHandler struct {
	counter         counter.Counter
	gracefulShutdown *counter.EnhancedGracefulShutdown
	rateLimiter      *limiter.RateLimiter
}

func NewHandler(c counter.Counter, gs *counter.EnhancedGracefulShutdown, rl *limiter.RateLimiter) *QPSHandler {
	return &QPSHandler{
		counter:         c,
		gracefulShutdown: gs,
		rateLimiter:      rl,
	}
}

func (handler *QPSHandler) Collect(c *gin.Context) {
	// 检查服务是否正在关闭中
	if !handler.gracefulShutdown.StartRequest() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "服务正在关闭中"})
		return
	}
	// 确保请求结束时调用EndRequest
	defer handler.gracefulShutdown.EndRequest()
	
	// 检查是否被限流
	if !handler.rateLimiter.Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求被限流"})
		return
	}
	
	var req struct {
		Count int64 `json:"count"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for i := int64(0); i < req.Count; i++ {
		handler.counter.Incr()
	}

	c.Status(http.StatusAccepted)
}

func (handler *QPSHandler) Query(c *gin.Context) {
	qps := handler.counter.CurrentQPS()
	c.JSON(http.StatusOK, gin.H{"qps": qps})
}

// GetStats 获取系统状态信息
func (handler *QPSHandler) GetStats(c *gin.Context) {
	// 获取QPS计数器状态
	qps := handler.counter.CurrentQPS()
	
	// 获取限流器状态
	limiterStats := handler.rateLimiter.GetStats()
	
	// 获取优雅关闭状态
	shutdownStatus := handler.gracefulShutdown.Status()
	shutdownActiveRequests := handler.gracefulShutdown.ActiveRequests()
	
	c.JSON(http.StatusOK, gin.H{
		"qps": qps,
		"limiter": limiterStats,
		"shutdown": map[string]interface{}{
			"status": shutdownStatus,
			"active_requests": shutdownActiveRequests,
		},
	})
}

// SetLimiterRate 设置限流器速率
func (handler *QPSHandler) SetLimiterRate(c *gin.Context) {
	var req struct {
		Rate int64 `json:"rate" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的速率参数"})
		return
	}
	
	if req.Rate <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "速率必须大于0"})
		return
	}
	
	handler.rateLimiter.SetRate(req.Rate)
	c.JSON(http.StatusOK, gin.H{"message": "限流速率已更新", "new_rate": req.Rate})
}

// ToggleLimiter 启用或禁用限流器
func (handler *QPSHandler) ToggleLimiter(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的参数"})
		return
	}
	
	handler.rateLimiter.SetEnabled(req.Enabled)
	c.JSON(http.StatusOK, gin.H{"message": "限流器状态已更新", "enabled": req.Enabled})
}
