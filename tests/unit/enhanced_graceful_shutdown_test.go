package unit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
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

func TestEnhancedGracefulShutdown(t *testing.T) {
	// 创建增强的优雅关闭管理器，设置较短的超时时间以加速测试
	timeout := 500 * time.Millisecond
	maxWait := 1 * time.Second
	gs := counter.NewEnhancedGracefulShutdown(timeout, maxWait)

	t.Run("基本功能测试", func(t *testing.T) {
		// 验证初始状态
		assert.Equal(t, "running", gs.Status())
		assert.Equal(t, int64(0), gs.ActiveRequests())
		assert.False(t, gs.IsForceShutdown())

		// 测试请求处理
		assert.True(t, gs.StartRequest())
		assert.Equal(t, int64(1), gs.ActiveRequests())
		gs.EndRequest()
		assert.Equal(t, int64(0), gs.ActiveRequests())
	})

	t.Run("优雅关闭测试 - 无活跃请求", func(t *testing.T) {
		// 创建新的关闭管理器
		gs := counter.NewEnhancedGracefulShutdown(timeout, maxWait)

		// 启动关闭流程
		ctx := context.Background()
		err := gs.Shutdown(ctx)

		// 验证关闭成功，无错误
		assert.NoError(t, err)
		assert.Equal(t, "graceful_shutdown_complete", gs.Status())
		assert.False(t, gs.IsForceShutdown())
	})

	t.Run("优雅关闭测试 - 有活跃请求但能在超时前完成", func(t *testing.T) {
		// 创建新的关闭管理器
		gs := counter.NewEnhancedGracefulShutdown(timeout, maxWait)

		// 模拟活跃请求
		gs.StartRequest()

		// 在后台启动关闭流程
		var wg sync.WaitGroup
		wg.Add(1)
		var shutdownErr error
		go func() {
			defer wg.Done()
			ctx := context.Background()
			shutdownErr = gs.Shutdown(ctx)
		}()

		// 等待一小段时间后结束请求
		time.Sleep(200 * time.Millisecond)
		gs.EndRequest()

		// 等待关闭完成
		wg.Wait()

		// 验证关闭成功，无错误
		assert.NoError(t, shutdownErr)
		assert.Equal(t, "graceful_shutdown_complete", gs.Status())
		assert.False(t, gs.IsForceShutdown())
	})

	t.Run("优雅关闭测试 - 超时但在最大等待时间内完成", func(t *testing.T) {
		// 创建新的关闭管理器，使用更短的超时
		shortTimeout := 100 * time.Millisecond
		gs := counter.NewEnhancedGracefulShutdown(shortTimeout, maxWait)

		// 模拟活跃请求
		gs.StartRequest()

		// 在后台启动关闭流程
		var wg sync.WaitGroup
		wg.Add(1)
		var shutdownErr error
		go func() {
			defer wg.Done()
			ctx := context.Background()
			shutdownErr = gs.Shutdown(ctx)
		}()

		// 等待超过正常超时时间但小于最大等待时间后结束请求
		time.Sleep(300 * time.Millisecond)
		gs.EndRequest()

		// 等待关闭完成
		wg.Wait()

		// 验证关闭成功，无错误
		assert.NoError(t, shutdownErr)
		assert.Equal(t, "delayed_shutdown_complete", gs.Status())
		assert.False(t, gs.IsForceShutdown())
	})

	t.Run("优雅关闭测试 - 超过最大等待时间强制关闭", func(t *testing.T) {
		// 创建新的关闭管理器，使用更短的超时和最大等待时间
		shortTimeout := 50 * time.Millisecond
		shortMaxWait := 100 * time.Millisecond
		gs := counter.NewEnhancedGracefulShutdown(shortTimeout, shortMaxWait)

		// 模拟活跃请求
		gs.StartRequest()

		// 在后台启动关闭流程
		var wg sync.WaitGroup
		wg.Add(1)
		var shutdownErr error
		go func() {
			defer wg.Done()
			ctx := context.Background()
			shutdownErr = gs.Shutdown(ctx)
		}()

		// 等待超过最大等待时间
		time.Sleep(200 * time.Millisecond)

		// 等待关闭完成
		wg.Wait()

		// 验证强制关闭，有错误
		assert.Error(t, shutdownErr)
		assert.Equal(t, "force_shutdown", gs.Status())
		assert.True(t, gs.IsForceShutdown())

		// 清理：结束请求，避免资源泄漏
		gs.EndRequest()
	})

	t.Run("拒绝新请求测试", func(t *testing.T) {
		// 创建新的关闭管理器
		gs := counter.NewEnhancedGracefulShutdown(timeout, maxWait)

		// 启动关闭流程
		ctx := context.Background()
		go gs.Shutdown(ctx)

		// 等待关闭开始
		time.Sleep(50 * time.Millisecond)

		// 尝试启动新请求，应该被拒绝
		assert.False(t, gs.StartRequest())
	})
}