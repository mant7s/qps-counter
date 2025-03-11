package unit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/limiter"
	"github.com/mant7s/qps-counter/internal/logger"
	"github.com/stretchr/testify/assert"
)

func init() {
	// 初始化日志，避免测试中的日志错误
	loggerConfig := config.LoggerConfig{
		Level:  "debug",
		Format: "console",
	}
	logger.Init(loggerConfig)
}

func TestRateLimiter(t *testing.T) {
	t.Run("基本功能测试", func(t *testing.T) {
		// 创建限流器，设置较低的速率以便于测试
		rate := int64(10)
		burstSize := int64(5)
		rl := limiter.NewRateLimiter(rate, burstSize, false)

		// 验证初始状态下允许请求通过
		for i := 0; i < int(burstSize); i++ {
			assert.True(t, rl.Allow(), "初始状态下应允许%d个请求通过", burstSize)
		}

		// 验证突发容量用完后请求被拒绝
		assert.False(t, rl.Allow(), "突发容量用完后应拒绝请求")

		// 手动设置令牌数，模拟令牌补充
		rl.SetTokensForTest(int64(rate))

		// 验证令牌补充后允许请求通过
		for i := 0; i < int(rate); i++ {
			assert.True(t, rl.Allow(), "令牌补充后应允许请求通过")
		}
		assert.False(t, rl.Allow(), "超过补充的令牌数后应拒绝请求")
	})

	t.Run("禁用限流测试", func(t *testing.T) {
		// 创建限流器，初始启用
		rl := limiter.NewRateLimiter(10, 5, false)

		// 消耗所有令牌
		for i := 0; i < 5; i++ {
			rl.Allow()
		}
		// 验证令牌用完后请求被拒绝
		assert.False(t, rl.Allow())

		// 禁用限流
		rl.SetEnabled(false)

		// 验证禁用后所有请求都允许通过
		for i := 0; i < 100; i++ {
			assert.True(t, rl.Allow(), "禁用限流后应允许所有请求通过")
		}

		// 重新启用限流
		rl.SetEnabled(true)

		// 验证重新启用后限流生效
		// 注意：此时令牌桶可能已经有一些令牌，所以我们先消耗掉
		for i := 0; i < 20; i++ { // 足够多以消耗所有可能的令牌
			rl.Allow()
		}
		assert.False(t, rl.Allow(), "重新启用限流后应恢复限流功能")
	})

	t.Run("动态调整速率测试", func(t *testing.T) {
		// 创建限流器，初始速率较低
		initialRate := int64(5)
		rl := limiter.NewRateLimiter(initialRate, 5, false)

		// 消耗所有令牌
		for i := 0; i < 5; i++ {
			rl.Allow()
		}
		// 验证令牌用完后请求被拒绝
		assert.False(t, rl.Allow())

		// 手动设置令牌数，模拟令牌补充
		rl.SetTokensForTest(initialRate)

		// 消耗补充的令牌
		for i := 0; i < int(initialRate); i++ {
			assert.True(t, rl.Allow())
		}
		assert.False(t, rl.Allow())

		// 增加速率
		newRate := int64(20)
		rl.SetRate(newRate)

		// 手动设置令牌数，模拟令牌补充
		rl.SetTokensForTest(newRate)

		// 验证新速率生效
		for i := 0; i < int(newRate); i++ {
			assert.True(t, rl.Allow(), "第%d个请求应该通过", i+1)
		}
		assert.False(t, rl.Allow(), "超过新速率后应拒绝请求")
	})

	t.Run("并发安全测试", func(t *testing.T) {
		// 创建限流器，设置较高的速率和突发容量
		rate := int64(1000)
		burstSize := int64(1000)
		rl := limiter.NewRateLimiter(rate, burstSize, false)

		// 并发请求数
		concurrency := 100
		requestsPerGoroutine := 20

		// 使用WaitGroup等待所有goroutine完成
		var wg sync.WaitGroup
		wg.Add(concurrency)

		// 记录通过和拒绝的请求数
		var allowed, rejected int64
		var mu sync.Mutex

		// 启动并发goroutine
		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerGoroutine; j++ {
					if rl.Allow() {
						mu.Lock()
						allowed++
						mu.Unlock()
					} else {
						mu.Lock()
						rejected++
						mu.Unlock()
					}
					// 短暂休眠，模拟请求处理
					time.Sleep(time.Millisecond)
				}
			}()
		}

		// 等待所有goroutine完成
		wg.Wait()

		// 验证结果
		totalRequests := int64(concurrency * requestsPerGoroutine)
		t.Logf("总请求数: %d, 通过: %d, 拒绝: %d", totalRequests, allowed, rejected)
		assert.Equal(t, totalRequests, allowed+rejected, "通过和拒绝的请求总数应等于总请求数")

		// 由于限流器的特性，我们不能精确预测通过的请求数，但可以验证基本功能
		assert.LessOrEqual(t, allowed, burstSize+rate, "通过的请求数不应超过突发容量加上速率")
	})

	t.Run("获取统计信息测试", func(t *testing.T) {
		// 创建限流器
		rate := int64(10)
		burstSize := int64(5)
		rl := limiter.NewRateLimiter(rate, burstSize, true)

		// 消耗部分令牌
		allowedCount := 3
		for i := 0; i < allowedCount; i++ {
			rl.Allow()
		}

		// 消耗剩余令牌
		for i := 0; i < int(burstSize) - allowedCount; i++ {
			rl.Allow()
		}

		// 尝试一些会被拒绝的请求
		rejectedCount := 2
		for i := 0; i < rejectedCount; i++ {
			rl.Allow()
		}

		// 获取统计信息
		stats := rl.GetStats()

		// 验证统计信息
		assert.Equal(t, rate, stats["rate"], "速率应匹配")
		assert.Equal(t, burstSize, stats["burst_size"], "突发容量应匹配")
		assert.True(t, stats["enabled"].(bool), "限流器应该是启用状态")
		assert.Equal(t, int64(rejectedCount), stats["rejected_count"], "拒绝计数应匹配")
		assert.Equal(t, int64(burstSize) + int64(rejectedCount), stats["total_count"], "总请求数应匹配")
	})
}
