package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mant7s/qps-counter/internal/counter"
	"net/http"
)

func NewRouter(counter counter.Counter, gracefulShutdown *counter.GracefulShutdown) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	handler := NewHandler(counter, gracefulShutdown)
	router.POST("/collect", handler.Collect)
	router.GET("/qps", handler.Query)
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	return router
}
