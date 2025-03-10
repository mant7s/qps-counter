package unit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/counter"
)

// createCounter 创建指定类型的计数器用于测试
func createCounter(t *testing.T, cfg *config.CounterConfig, counterType string) counter.Counter {
	// 保存原始类型
	originalType := cfg.Type
	// 设置测试需要的类型
	cfg.Type = counterType
	// 创建计数器
	c := counter.NewCounter(cfg)
	// 恢复原始类型
	cfg.Type = originalType
	return c
}

func TestCounter(t *testing.T) {
	cfg := &config.CounterConfig{
		// 使用1秒的窗口大小，简化计算
		WindowSize: 1 * time.Second,
		SlotNum:    20,
		Precision:  100 * time.Millisecond,
	}

	// 定义要测试的计数器类型
	counterTypes := []string{counter.ShardedType, counter.LockFreeType}

	for _, cType := range counterTypes {
		t.Run("concurrency safety for "+cType, func(t *testing.T) {
			// 创建指定类型的计数器
			c := createCounter(t, cfg, cType)
			defer c.Stop()

			const (
				workers   = 100
				perWorker = 62 // 每个工作者增加62次
				total     = perWorker * workers
			)

			// 确保在窗口开始时进行测试
			start := time.Now().Truncate(cfg.WindowSize).Add(cfg.WindowSize)
			time.Sleep(time.Until(start))

			var wg sync.WaitGroup
			wg.Add(workers)
			for i := 0; i < workers; i++ {
				go func() {
					defer wg.Done()
					for j := 0; j < perWorker; j++ {
						c.Incr()
					}
				}()
			}
			wg.Wait()

			// 增加等待时间，确保计数器有足够时间更新
			time.Sleep(5 * cfg.Precision)

			reportedQPS := c.CurrentQPS()
			assert.Equal(t, int64(total), reportedQPS, "Expected reported QPS to be %d, got %d", total, reportedQPS)
		})
	}
}
