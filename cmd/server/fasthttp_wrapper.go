package main

import (
	"context"
	"github.com/valyala/fasthttp"
)

// FastHTTPServerWrapper 包装FastHTTP服务器以实现Server接口
type FastHTTPServerWrapper struct {
	server *fasthttp.Server
}

// ListenAndServe 实现Server接口的ListenAndServe方法
func (w *FastHTTPServerWrapper) ListenAndServe() error {
	return w.server.ListenAndServe(w.server.Name)
}

// Shutdown 实现Server接口的Shutdown方法
func (w *FastHTTPServerWrapper) Shutdown(ctx context.Context) error {
	return w.server.ShutdownWithContext(ctx)
}