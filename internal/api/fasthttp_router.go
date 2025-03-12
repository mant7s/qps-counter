package api

import (
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/mant7s/qps-counter/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type FastHTTPRouter struct {
	handler *FastHTTPHandler
}

func NewFastHTTPRouter(counter counter.Counter, gracefulShutdown *counter.EnhancedGracefulShutdown, rateLimiter *limiter.RateLimiter, metricsCollector *metrics.Metrics, metricsEndpoint string, metricsEnabled bool) *FastHTTPRouter {
	handler := NewFastHTTPHandler(counter, gracefulShutdown, rateLimiter)
	return &FastHTTPRouter{handler: handler}
}

func (r *FastHTTPRouter) Handler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		method := string(ctx.Method())

		switch {
		case method == "POST" && path == "/collect":
			r.handler.Collect(ctx)
		case method == "GET" && path == "/qps":
			r.handler.Query(ctx)
		case method == "GET" && path == "/stats":
			r.handler.GetStats(ctx)
		case method == "POST" && path == "/limiter/rate":
			r.handler.SetLimiterRate(ctx)
		case method == "POST" && path == "/limiter/toggle":
			r.handler.ToggleLimiter(ctx)
		case method == "GET" && path == "/healthz":
			r.handler.HealthCheck(ctx)
		case method == "GET" && path == "/metrics":
			// 使用适配器将promhttp.Handler转换为fasthttp处理器
			fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())(ctx)
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}
}