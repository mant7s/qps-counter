package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/mant7s/qps-counter/internal/api"
	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/mant7s/qps-counter/internal/logger"
	"github.com/mant7s/qps-counter/internal/metrics"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	logger.Init(cfg.Logger)
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Fatal("Failed to sync logger:", err)
		}
	}()

	// 创建增强的优雅关闭管理器，使用配置的超时时间
	gracefulShutdown := counter.NewEnhancedGracefulShutdown(cfg.Shutdown.Timeout, cfg.Shutdown.MaxWait)

	qpsCounter := counter.NewCounter(&cfg.Counter)
	defer qpsCounter.Stop()

	// 创建自适应分片管理器，设置最小分片数为CPU核心数，最大分片数为CPU核心数的8倍
	minShards := runtime.NumCPU()
	maxShards := runtime.NumCPU() * 8
	adaptiveManager := counter.NewAdaptiveShardingManager(qpsCounter, &cfg.Counter, minShards, maxShards)
	defer adaptiveManager.Stop()

	// 创建限流器，使用配置的参数
	rateLimiter := limiter.NewRateLimiter(cfg.Limiter.Rate, cfg.Limiter.Burst, cfg.Limiter.Adaptive)
	// 根据配置决定是否启用限流器
	rateLimiter.SetEnabled(cfg.Limiter.Enabled)

	// 初始化指标收集器
	metricsCollector := metrics.NewMetrics(qpsCounter)
	// 根据配置决定是否启用指标收集
	if cfg.Metrics.Enabled {
		metricsCollector.Start(cfg.Metrics.Interval)
		defer metricsCollector.Stop()
	}

	router := api.NewRouter(qpsCounter, gracefulShutdown, rateLimiter, metricsCollector, cfg.Metrics.Endpoint, cfg.Metrics.Enabled)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Fatal("Server start failed", zap.Error(err))
		}
	}()

	logger.Info("服务已启动", zap.Int("port", cfg.Server.Port), zap.String("metrics", "/metrics"))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
	defer cancel()

	// 启动优雅关闭流程
	if err := gracefulShutdown.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown error", zap.Error(err))
	}

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
}
