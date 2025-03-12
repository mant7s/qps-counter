package api

import (
	"encoding/json"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/valyala/fasthttp"
	"net/http"
)

type FastHTTPHandler struct {
	counter          counter.Counter
	gracefulShutdown *counter.EnhancedGracefulShutdown
	rateLimiter      *limiter.RateLimiter
}

func NewFastHTTPHandler(c counter.Counter, gs *counter.EnhancedGracefulShutdown, rl *limiter.RateLimiter) *FastHTTPHandler {
	return &FastHTTPHandler{
		counter:          c,
		gracefulShutdown: gs,
		rateLimiter:      rl,
	}
}

func (h *FastHTTPHandler) Collect(ctx *fasthttp.RequestCtx) {
	// 检查服务是否正在关闭中
	if !h.gracefulShutdown.StartRequest() {
		ctx.SetStatusCode(http.StatusServiceUnavailable)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "服务正在关闭中"})
		return
	}
	// 确保请求结束时调用EndRequest
	defer h.gracefulShutdown.EndRequest()

	// 检查是否被限流
	if !h.rateLimiter.Allow() {
		ctx.SetStatusCode(http.StatusTooManyRequests)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "请求被限流"})
		return
	}

	var req struct {
		Count int64 `json:"count"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		json.NewEncoder(ctx).Encode(map[string]string{"error": err.Error()})
		return
	}

	for i := int64(0); i < req.Count; i++ {
		h.counter.Incr()
	}

	ctx.SetStatusCode(http.StatusAccepted)
}

func (h *FastHTTPHandler) Query(ctx *fasthttp.RequestCtx) {
	qps := h.counter.CurrentQPS()
	ctx.SetStatusCode(http.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]interface{}{"qps": qps})
}

func (h *FastHTTPHandler) GetStats(ctx *fasthttp.RequestCtx) {
	qps := h.counter.CurrentQPS()
	limiterStats := h.rateLimiter.GetStats()
	shutdownStatus := h.gracefulShutdown.Status()
	shutdownActiveRequests := h.gracefulShutdown.ActiveRequests()

	ctx.SetStatusCode(http.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]interface{}{
		"qps": qps,
		"limiter": limiterStats,
		"shutdown": map[string]interface{}{
			"status":          shutdownStatus,
			"active_requests": shutdownActiveRequests,
		},
	})
}

func (h *FastHTTPHandler) SetLimiterRate(ctx *fasthttp.RequestCtx) {
	var req struct {
		Rate int64 `json:"rate"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "无效的速率参数"})
		return
	}

	if req.Rate <= 0 {
		ctx.SetStatusCode(http.StatusBadRequest)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "速率必须大于0"})
		return
	}

	h.rateLimiter.SetRate(req.Rate)
	ctx.SetStatusCode(http.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]interface{}{
		"message":  "限流速率已更新",
		"new_rate": req.Rate,
	})
}

func (h *FastHTTPHandler) ToggleLimiter(ctx *fasthttp.RequestCtx) {
	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "无效的参数"})
		return
	}

	h.rateLimiter.SetEnabled(req.Enabled)
	ctx.SetStatusCode(http.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]interface{}{
		"message": "限流器状态已更新",
		"enabled": req.Enabled,
	})
}

func (h *FastHTTPHandler) HealthCheck(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(http.StatusOK)
	ctx.SetBodyString("ok")
}