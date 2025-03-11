package unit_test

import (
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

// mockCounter 创建一个模拟的计数器，用于测试
type mockCounter struct {
	qps int64
}

func (m *mockCounter) Incr() {
	// 空实现
}

func (m *mockCounter) CurrentQPS() int64 {
	return m.qps
}

func (m *mockCounter) Stop() {
	// 空实现
}

func (m *mockCounter) SetQPS(qps int64) {
	m.qps = qps
}

func TestEnhancedAdaptiveShardingManager(t *testing.T) {
	// 创建配置
	cfg := &config.CounterConfig{
		Type:       "sharded",
		WindowSize: 1 * time.Second,
		SlotNum:    10,
		Precision:  100 * time.Millisecond,
	}

	// 创建模拟计数器
	mock := &mockCounter{qps: 1000}

	// 设置较短的调整间隔以加速测试
	adjustInterval := 100 * time.Millisecond
	minShards := 2
	maxShards := 8
	memoryThreshold := uint64(100 * 1024 * 1024) // 100MB，设置较小的阈值以便于测试

	t.Run("基本功能测试", func(t *testing.T) {
		// 创建增强的自适应分片管理器
		asm := counter.NewEnhancedAdaptiveShardingManager(
			mock,
			cfg,
			minShards,
			maxShards,
			memoryThreshold,
			adjustInterval,
		)
		defer asm.Stop()

		// 验证初始状态
		assert.Equal(t, int32(minShards), asm.GetCurrentShards())

		// 获取状态信息
		stats := asm.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, minShards, stats["min_shards"])
		assert.Equal(t, maxShards, stats["max_shards"])
		assert.Equal(t, int64(1000), stats["current_qps"])
		assert.Equal(t, memoryThreshold, stats["memory_threshold"])
	})

	t.Run("QPS增加时分片数增加测试", func(t *testing.T) {
		// 创建模拟计数器，初始QPS较低
		mock := &mockCounter{qps: 1000}

		// 创建增强的自适应分片管理器
		asm := counter.NewEnhancedAdaptiveShardingManager(
			mock,
			cfg,
			minShards,
			maxShards,
			memoryThreshold,
			adjustInterval,
		)
		defer asm.Stop()

		// 等待初始调整完成
		time.Sleep(adjustInterval * 2)

		// 模拟QPS大幅增加
		mock.SetQPS(5000) // 增加500%

		// 等待调整发生
		time.Sleep(adjustInterval * 2)

		// 验证分片数增加
		currentShards := asm.GetCurrentShards()
		assert.Greater(t, int(currentShards), minShards, "QPS增加后分片数应该增加")
	})

	t.Run("QPS减少时分片数减少测试", func(t *testing.T) {
		// 创建模拟计数器，初始QPS较高
		mock := &mockCounter{qps: 5000}

		// 创建增强的自适应分片管理器，初始设置为最大分片数
		asm := counter.NewEnhancedAdaptiveShardingManager(
			mock,
			cfg,
			minShards,
			maxShards,
			memoryThreshold,
			adjustInterval,
		)
		defer asm.Stop()

		// 手动设置初始分片数为最大值
		// 注意：这里假设有一个SetCurrentShards方法，如果没有，可以通过其他方式实现
		// 例如，可以通过反射或者添加一个测试专用的方法
		// 这里我们通过调整QPS来间接实现
		time.Sleep(adjustInterval * 2) // 等待初始调整
		mock.SetQPS(10000)             // 设置一个非常高的QPS
		time.Sleep(adjustInterval * 2) // 等待调整到较高分片数

		// 现在模拟QPS大幅下降
		mock.SetQPS(1000) // 减少90%

		// 等待调整发生
		time.Sleep(adjustInterval * 3)

		// 验证分片数减少
		currentShards := asm.GetCurrentShards()
		assert.Less(t, int(currentShards), maxShards, "QPS减少后分片数应该减少")
	})

	t.Run("内存使用超过阈值时分片数减少测试", func(t *testing.T) {
		// 跳过此测试，如果无法控制内存使用
		if testing.Short() {
			t.Skip("跳过内存相关测试")
		}

		// 创建模拟计数器
		mock := &mockCounter{qps: 1000}

		// 设置一个非常低的内存阈值，确保会触发内存调整
		lowMemoryThreshold := uint64(1 * 1024 * 1024) // 1MB

		// 创建增强的自适应分片管理器
		asm := counter.NewEnhancedAdaptiveShardingManager(
			mock,
			cfg,
			minShards,
			maxShards,
			lowMemoryThreshold,
			adjustInterval,
		)
		defer asm.Stop()

		// 手动设置初始分片数为较高值
		time.Sleep(adjustInterval * 2) // 等待初始调整
		mock.SetQPS(10000)             // 设置一个非常高的QPS
		time.Sleep(adjustInterval * 2) // 等待调整到较高分片数

		// 分配一些内存，确保超过阈值
		memoryHog := make([]byte, 10*1024*1024) // 分配10MB
		for i := range memoryHog {
			memoryHog[i] = byte(i % 256) // 确保内存被实际使用
		}

		// 等待调整发生
		time.Sleep(adjustInterval * 3)

		// 验证分片数减少到最小值
		currentShards := asm.GetCurrentShards()
		assert.Equal(t, int32(minShards), currentShards, "内存使用超过阈值后分片数应该减少到最小值")

		// 防止memoryHog被过早GC
		_ = memoryHog
	})

	t.Run("设置内存阈值和权重测试", func(t *testing.T) {
		// 创建模拟计数器
		mock := &mockCounter{qps: 1000}

		// 创建增强的自适应分片管理器
		asm := counter.NewEnhancedAdaptiveShardingManager(
			mock,
			cfg,
			minShards,
			maxShards,
			memoryThreshold,
			adjustInterval,
		)
		defer asm.Stop()

		// 测试设置新的内存阈值
		newThreshold := uint64(200 * 1024 * 1024) // 200MB
		asm.SetMemoryThreshold(newThreshold)

		// 测试设置新的权重
		asm.SetWeights(0.7, 0.3) // 70% QPS权重，30%内存权重

		// 获取状态并验证设置是否生效
		stats := asm.GetStats()
		assert.Equal(t, newThreshold, stats["memory_threshold"])
	})
}
