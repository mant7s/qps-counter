package limiter

import (
	"sync"
	"time"

	"github.com/mant7s/qps-counter/internal/logger"
	"go.uber.org/zap"
)

// RateLimiter 提供基于令牌桶算法的限流功能
type RateLimiter struct {
	rate         int64         // 每秒允许的请求数
	burstSize    int64         // 突发请求容量
	tokens       int64         // 当前可用令牌数
	lastRefill   time.Time     // 上次填充令牌的时间
	enabled      bool          // 是否启用限流
	mu           sync.Mutex    // 保护并发访问
	adaptive     bool          // 是否启用自适应限流
	rejectedCount int64        // 被拒绝的请求计数
	totalCount    int64        // 总请求计数
}

// NewRateLimiter 创建一个新的限流器
func NewRateLimiter(rate, burstSize int64, adaptive bool) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burstSize:  burstSize,
		tokens:     burstSize, // 初始填满令牌
		lastRefill: time.Now(),
		enabled:    true,
		adaptive:   adaptive,
	}
}

// Allow 检查是否允许当前请求通过
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.enabled {
		return true
	}

	rl.totalCount++

	// 计算从上次填充到现在应该添加的令牌数
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	newTokens := int64(elapsed * float64(rl.rate))

	if newTokens > 0 {
		rl.tokens += newTokens
		if rl.tokens > rl.burstSize {
			rl.tokens = rl.burstSize
		}
		rl.lastRefill = now
	}

	// 如果有可用令牌，则允许请求通过
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	// 记录被拒绝的请求
	rl.rejectedCount++
	if rl.rejectedCount%100 == 0 { // 每100次拒绝记录一次日志，避免日志过多
		logger.Warn("请求被限流器拒绝", 
			zap.Int64("rejected_count", rl.rejectedCount),
			zap.Int64("total_count", rl.totalCount),
			zap.Float64("reject_rate", float64(rl.rejectedCount)/float64(rl.totalCount)),
		)
	}

	return false
}

// SetRate 动态调整限流速率
func (rl *RateLimiter) SetRate(newRate int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.rate = newRate
	logger.Info("限流器速率已调整", zap.Int64("new_rate", newRate))
}

// SetEnabled 启用或禁用限流器
func (rl *RateLimiter) SetEnabled(enabled bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.enabled = enabled
	logger.Info("限流器状态已更改", zap.Bool("enabled", enabled))
}

// GetStats 获取限流器统计信息
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return map[string]interface{}{
		"rate":          rl.rate,
		"burst_size":    rl.burstSize,
		"current_tokens": rl.tokens,
		"enabled":       rl.enabled,
		"rejected_count": rl.rejectedCount,
		"total_count":   rl.totalCount,
		"reject_rate":   float64(rl.rejectedCount) / float64(max(rl.totalCount, 1)),
	}
}

// 辅助函数，返回两个int64中的较大值
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// SetTokensForTest 设置当前可用令牌数，仅用于测试
func (rl *RateLimiter) SetTokensForTest(tokens int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.tokens = tokens
}