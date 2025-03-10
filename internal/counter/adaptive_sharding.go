package counter

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/logger"
)

// AdaptiveShardingManager 管理分片数量的自适应调整
type AdaptiveShardingManager struct {
	counter        Counter
	config         *config.CounterConfig
	lastQPS        atomic.Int64
	lastAdjustTime atomic.Int64
	stopChan       chan struct{}
	minShards      int
	maxShards      int
	currentShards  atomic.Int32
}

// NewAdaptiveShardingManager 创建一个新的自适应分片管理器
func NewAdaptiveShardingManager(counter Counter, cfg *config.CounterConfig, minShards, maxShards int) *AdaptiveShardingManager {
	if minShards <= 0 {
		minShards = 4
	}
	if maxShards <= 0 {
		maxShards = 64
	}

	asm := &AdaptiveShardingManager{
		counter:       counter,
		config:        cfg,
		stopChan:      make(chan struct{}),
		minShards:     minShards,
		maxShards:     maxShards,
		currentShards: atomic.Int32{},
	}

	// 初始设置为最小分片数
	asm.currentShards.Store(int32(minShards))
	asm.lastAdjustTime.Store(time.Now().Unix())

	// 启动自适应调整协程
	go asm.adaptiveWorker()

	return asm
}

// adaptiveWorker 周期性检查负载并调整分片数量
func (asm *AdaptiveShardingManager) adaptiveWorker() {
	// 每10秒检查一次负载情况
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			asm.adjustShards()
		case <-asm.stopChan:
			return
		}
	}
}

// adjustShards 根据当前QPS调整分片数量
func (asm *AdaptiveShardingManager) adjustShards() {
	currentQPS := asm.counter.CurrentQPS()
	lastQPS := asm.lastQPS.Swap(currentQPS)
	currentShards := asm.currentShards.Load()

	// 计算QPS变化率
	var qpsChangeRate float64
	if lastQPS > 0 {
		qpsChangeRate = float64(currentQPS-lastQPS) / float64(lastQPS)
	}

	// 根据QPS变化率调整分片数量
	var newShards int32
	if qpsChangeRate > 0.3 && currentShards < int32(asm.maxShards) {
		// QPS增长超过30%，增加分片数
		newShards = currentShards + int32(float64(currentShards)*0.5)
		if newShards > int32(asm.maxShards) {
			newShards = int32(asm.maxShards)
		}
	} else if qpsChangeRate < -0.3 && currentShards > int32(asm.minShards) {
		// QPS下降超过30%，减少分片数
		newShards = currentShards - int32(float64(currentShards)*0.3)
		if newShards < int32(asm.minShards) {
			newShards = int32(asm.minShards)
		}
	} else {
		// QPS变化不大，保持当前分片数
		return
	}

	// 更新分片数量
	if newShards != currentShards {
		asm.currentShards.Store(newShards)
		asm.lastAdjustTime.Store(time.Now().Unix())
		logger.Info(fmt.Sprintf("自适应调整分片数量: %d -> %d, 当前QPS: %d", currentShards, newShards, currentQPS))
	}
}

// Stop 停止自适应分片管理器
func (asm *AdaptiveShardingManager) Stop() {
	close(asm.stopChan)
}

// GetCurrentShards 获取当前分片数量
func (asm *AdaptiveShardingManager) GetCurrentShards() int32 {
	return asm.currentShards.Load()
}
