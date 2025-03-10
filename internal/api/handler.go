package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mant7s/qps-counter/internal/counter"
	"net/http"
)

type QPSHandler struct {
	counter         counter.Counter
	gracefulShutdown *counter.GracefulShutdown
}

func NewHandler(c counter.Counter, gs *counter.GracefulShutdown) *QPSHandler {
	return &QPSHandler{
		counter:         c,
		gracefulShutdown: gs,
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
