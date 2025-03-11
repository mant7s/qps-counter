package counter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mant7s/qps-counter/internal/logger"
	"go.uber.org/zap"
)

// EnhancedGracefulShutdown 提供增强的优雅关闭功能
type EnhancedGracefulShutdown struct {
	*BaseComponent   // 嵌入基础组件
	shutdownTimeout time.Duration
	doneChan        chan struct{}
	wg              sync.WaitGroup
	shutdownOnce    sync.Once
	shutdownStarted atomic.Bool
	mu              sync.RWMutex
	
	// 增强功能
	activeRequests  atomic.Int64    // 当前活跃请求数
	maxWaitTime     time.Duration   // 最大等待时间
	shutdownTime    atomic.Int64    // 关闭开始时间
	forceShutdown   atomic.Bool     // 是否强制关闭
	shutdownStatus  string          // 关闭状态
	statusLock      sync.RWMutex    // 状态锁
}

// NewEnhancedGracefulShutdown 创建一个新的增强优雅关闭管理器
func NewEnhancedGracefulShutdown(timeout, maxWait time.Duration) *EnhancedGracefulShutdown {
	if maxWait <= 0 {
		maxWait = timeout * 2 // 默认最大等待时间为超时时间的两倍
	}
	
	return &EnhancedGracefulShutdown{
		BaseComponent:   NewBaseComponent(),
		shutdownTimeout: timeout,
		maxWaitTime:     maxWait,
		doneChan:        make(chan struct{}),
		shutdownStatus:  "running",
	}
}

// StartRequest 标记一个新请求的开始，返回是否接受该请求
func (gs *EnhancedGracefulShutdown) StartRequest() bool {
	// 快速检查是否已开始关闭
	if gs.shutdownStarted.Load() {
		return false
	}
	
	// 增加活跃请求计数
	gs.activeRequests.Add(1)
	gs.wg.Add(1)
	
	// 二次检查，如果在增加计数后开始了关闭，需要回滚
	if gs.shutdownStarted.Load() {
		gs.activeRequests.Add(-1)
		gs.wg.Done()
		return false
	}
	
	return true
}

// EndRequest 标记一个请求的结束
func (gs *EnhancedGracefulShutdown) EndRequest() {
	gs.activeRequests.Add(-1)
	gs.wg.Done()
}

// ActiveRequests 返回当前活跃的请求数
func (gs *EnhancedGracefulShutdown) ActiveRequests() int64 {
	return gs.activeRequests.Load()
}

// SetStatus 设置关闭状态
func (gs *EnhancedGracefulShutdown) SetStatus(status string) {
	gs.statusLock.Lock()
	defer gs.statusLock.Unlock()
	gs.shutdownStatus = status
	logger.Info("优雅关闭状态变更", zap.String("status", status))
}

// Status 获取当前关闭状态
func (gs *EnhancedGracefulShutdown) Status() string {
	gs.statusLock.RLock()
	defer gs.statusLock.RUnlock()
	return gs.shutdownStatus
}

// Shutdown 开始优雅关闭过程，带有超时控制
func (gs *EnhancedGracefulShutdown) Shutdown(ctx context.Context) error {
	var shutdownErr error
	
	gs.shutdownOnce.Do(func() {
		// 标记开始关闭
		gs.shutdownStarted.Store(true)
		gs.shutdownTime.Store(time.Now().Unix())
		gs.SetStatus("shutting_down")
		
		logger.Info("开始优雅关闭服务...", 
			zap.Int64("active_requests", gs.ActiveRequests()),
			zap.Duration("timeout", gs.shutdownTimeout),
			zap.Duration("max_wait", gs.maxWaitTime))
		
		// 通知所有监听器服务正在关闭
		gs.Stop() // 使用基础组件的方法关闭停止通道
		
		// 创建一个带超时的上下文
		shutdownCtx, cancel := context.WithTimeout(ctx, gs.shutdownTimeout)
		defer cancel()
		
		// 创建一个带最大等待时间的上下文
		maxWaitCtx, maxWaitCancel := context.WithTimeout(ctx, gs.maxWaitTime)
		defer maxWaitCancel()
		
		// 等待所有请求完成或超时
		done := make(chan struct{})
		go func() {
			gs.wg.Wait()
			close(done)
		}()
		
		// 定期报告剩余请求数
		go gs.reportActiveRequests(done)
		
		// 等待完成或超时
		select {
		case <-done:
			gs.SetStatus("graceful_shutdown_complete")
			logger.Info("所有请求已处理完成，服务正常关闭")
			
		case <-shutdownCtx.Done():
			// 超过正常超时，但仍在最大等待时间内，继续等待但记录警告
			gs.SetStatus("timeout_waiting")
			logger.Warn("关闭超时，等待剩余请求处理完成", 
				zap.Int64("remaining_requests", gs.ActiveRequests()))
			
			// 继续等待直到最大等待时间或全部完成
			select {
			case <-done:
				gs.SetStatus("delayed_shutdown_complete")
				logger.Info("所有请求已处理完成，服务延迟关闭")
				
			case <-maxWaitCtx.Done():
				// 达到最大等待时间，强制关闭
				gs.forceShutdown.Store(true)
				gs.SetStatus("force_shutdown")
				shutdownErr = context.DeadlineExceeded
				logger.Error("达到最大等待时间，强制关闭服务", 
					zap.Int64("abandoned_requests", gs.ActiveRequests()))
			}
		}
		
		// 关闭完成
		close(gs.doneChan)
	})
	
	return shutdownErr
}

// 定期报告活跃请求数
func (gs *EnhancedGracefulShutdown) reportActiveRequests(done chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			active := gs.ActiveRequests()
			if active > 0 {
				logger.Info("等待请求完成", 
					zap.Int64("remaining", active),
					zap.Int64("shutdown_seconds", time.Now().Unix() - gs.shutdownTime.Load()))
			}
		case <-done:
			return
		}
	}
}

// ShutdownChan 返回一个通道，当开始关闭时会被关闭
func (gs *EnhancedGracefulShutdown) ShutdownChan() <-chan struct{} {
	return gs.StopChan() // 使用基础组件的方法获取停止通道
}

// DoneChan 返回一个通道，当关闭完成时会被关闭
func (gs *EnhancedGracefulShutdown) DoneChan() <-chan struct{} {
	return gs.doneChan
}

// IsForceShutdown 返回是否是强制关闭
func (gs *EnhancedGracefulShutdown) IsForceShutdown() bool {
	return gs.forceShutdown.Load()
}

// ShutdownTime 返回关闭开始的时间戳
func (gs *EnhancedGracefulShutdown) ShutdownTime() int64 {
	return gs.shutdownTime.Load()
}