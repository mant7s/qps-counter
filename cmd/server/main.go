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
	"time"

	"github.com/mant7s/qps-counter/internal/api"
	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
	"github.com/mant7s/qps-counter/internal/logger"
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

	// 创建优雅关闭管理器，设置超时时间为5秒
	gracefulShutdown := counter.NewGracefulShutdown(5 * time.Second)

	qpsCounter := counter.NewCounter(&cfg.Counter)
	defer qpsCounter.Stop()

	// 创建自适应分片管理器，设置最小分片数为CPU核心数，最大分片数为CPU核心数的8倍
	minShards := runtime.NumCPU()
	maxShards := runtime.NumCPU() * 8
	adaptiveManager := counter.NewAdaptiveShardingManager(qpsCounter, &cfg.Counter, minShards, maxShards)
	defer adaptiveManager.Stop()

	router := api.NewRouter(qpsCounter, gracefulShutdown)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 启动优雅关闭流程
	if err := gracefulShutdown.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown error", zap.Error(err))
	}

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
}
