package counter

import (
	"context"
	"sync"
	"time"

	"github.com/mant7s/qps-counter/internal/logger"
)

// GracefulShutdown 提供优雅关闭功能，确保所有请求都被处理完成
type GracefulShutdown struct {
	shutdownTimeout time.Duration
	shutdownChan    chan struct{}
	doneChan        chan struct{}
	wg              sync.WaitGroup
	shutdownOnce    sync.Once
	shutdownStarted bool
	mu              sync.Mutex
}

// NewGracefulShutdown 创建一个新的优雅关闭管理器
func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		shutdownTimeout: timeout,
		shutdownChan:    make(chan struct{}),
		doneChan:        make(chan struct{}),
	}
}

// StartRequest 标记一个新请求的开始
func (gs *GracefulShutdown) StartRequest() bool {
	gs.mu.Lock()
	if gs.shutdownStarted {
		gs.mu.Unlock()
		return false
	}
	gs.wg.Add(1)
	gs.mu.Unlock()
	return true
}

// EndRequest 标记一个请求的结束
func (gs *GracefulShutdown) EndRequest() {
	gs.wg.Done()
}

// Shutdown 开始优雅关闭过程
func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	gs.shutdownOnce.Do(func() {
		gs.mu.Lock()
		gs.shutdownStarted = true
		gs.mu.Unlock()

		logger.Info("开始优雅关闭服务...")
		close(gs.shutdownChan)

		// 创建一个带超时的上下文
		shutdownCtx, cancel := context.WithTimeout(ctx, gs.shutdownTimeout)
		defer cancel()

		// 等待所有请求完成或超时
		done := make(chan struct{})
		go func() {
			gs.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			logger.Info("所有请求已处理完成，服务关闭")
		case <-shutdownCtx.Done():
			logger.Warn("关闭超时，强制关闭服务")
		}

		close(gs.doneChan)
	})

	return nil
}

// ShutdownChan 返回一个通道，当开始关闭时会被关闭
func (gs *GracefulShutdown) ShutdownChan() <-chan struct{} {
	return gs.shutdownChan
}

// DoneChan 返回一个通道，当关闭完成时会被关闭
func (gs *GracefulShutdown) DoneChan() <-chan struct{} {
	return gs.doneChan
}