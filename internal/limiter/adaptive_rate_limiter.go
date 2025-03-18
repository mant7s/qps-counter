package limiter

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mant7s/qps-counter/internal/logger"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// AdaptiveRateLimiter 提供基于系统资源的自适应限流功能
type AdaptiveRateLimiter struct {
	limiter       *rate.Limiter
	baseRate      float64       // 基础限流速率
	cpuThreshold  float64       // CPU使用率阈值
	memThreshold  uint64        // 内存使用阈值
	adjustFactor  float64       // 调整系数
	enabled       atomic.Bool   // 是否启用限流
	mu            sync.RWMutex  // 保护并发访问
	stopChan      chan struct{} // 停止信号
	rejectedCount atomic.Int64  // 被拒绝的请求计数
	totalCount    atomic.Int64  // 总请求计数
}

// NewAdaptiveRateLimiter 创建一个新的自适应限流器
func NewAdaptiveRateLimiter(baseRate float64, burst int) *AdaptiveRateLimiter {
	arl := &AdaptiveRateLimiter{
		limiter:      rate.NewLimiter(rate.Limit(baseRate), burst),
		baseRate:     baseRate,
		cpuThreshold: 70.0,    // CPU使用率超过70%开始限流
		memThreshold: 1 << 30, // 内存阈值1GB
		adjustFactor: 0.8,     // 调整因子
		stopChan:     make(chan struct{}),
	}

	arl.enabled.Store(true)
	go arl.adaptiveWorker()
	return arl
}

// Allow 检查是否允许当前请求通过
func (arl *AdaptiveRateLimiter) Allow() bool {
	if !arl.enabled.Load() {
		return true
	}

	arl.totalCount.Add(1)
	allowed := arl.limiter.Allow()
	if !allowed {
		rejected := arl.rejectedCount.Add(1)
		if rejected%100 == 0 { // 每100次拒绝记录一次日志
			logger.Warn("请求被限流器拒绝",
				zap.Int64("rejected_count", rejected),
				zap.Int64("total_count", arl.totalCount.Load()),
				zap.Float64("current_limit", float64(arl.limiter.Limit())),
			)
		}
	}
	return allowed
}

// adaptiveWorker 周期性检查系统资源并调整限流参数
func (arl *AdaptiveRateLimiter) adaptiveWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			arl.adjustRate()
		case <-arl.stopChan:
			return
		}
	}
}

// adjustRate 根据系统资源使用情况调整限流速率
func (arl *AdaptiveRateLimiter) adjustRate() {
	// 获取系统资源使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算调整系数
	adjustment := 1.0
	if memStats.Alloc > arl.memThreshold {
		// 当内存使用超过阈值时，降低限流速率
		adjustment *= arl.adjustFactor
	}

	// 应用新的限流速率
	newRate := arl.baseRate * adjustment
	arl.mu.Lock()
	arl.limiter.SetLimit(rate.Limit(newRate))
	arl.mu.Unlock()

	logger.Info("限流器参数已调整",
		zap.Float64("new_rate", newRate),
		zap.Uint64("memory_usage", memStats.Alloc),
	)
}

// Stop 停止自适应限流器
func (arl *AdaptiveRateLimiter) Stop() {
	close(arl.stopChan)
}

// GetStats 获取限流器统计信息
func (arl *AdaptiveRateLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"base_rate":      arl.baseRate,
		"current_limit":  float64(arl.limiter.Limit()),
		"enabled":        arl.enabled.Load(),
		"rejected_count": arl.rejectedCount.Load(),
		"total_count":    arl.totalCount.Load(),
	}
}
