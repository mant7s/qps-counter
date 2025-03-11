package counter

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/mant7s/qps-counter/internal/config"
	"github.com/mant7s/qps-counter/internal/logger"
	"go.uber.org/zap"
)

// EnhancedAdaptiveShardingManager 增强的分片管理器，考虑内存使用情况
type EnhancedAdaptiveShardingManager struct {
	*BaseComponent   // 嵌入基础组件
	counter        Counter
	config         *config.CounterConfig
	lastQPS        atomic.Int64
	minShards      int
	maxShards      int
	currentShards  atomic.Int32
	
	// 增强功能
	memoryThreshold uint64        // 内存使用阈值（字节）
	lastMemoryUsage atomic.Uint64 // 上次内存使用量
	memoryWeight    float64       // 内存因素权重
	qpsWeight       float64       // QPS因素权重
	adjustInterval  time.Duration // 调整间隔
}

// NewEnhancedAdaptiveShardingManager 创建一个新的增强自适应分片管理器
func NewEnhancedAdaptiveShardingManager(
	counter Counter, 
	cfg *config.CounterConfig, 
	minShards, maxShards int,
	memoryThreshold uint64,
	adjustInterval time.Duration,
) *EnhancedAdaptiveShardingManager {
	if minShards <= 0 {
		minShards = runtime.NumCPU()
	}
	if maxShards <= 0 {
		maxShards = runtime.NumCPU() * 8
	}
	if adjustInterval <= 0 {
		adjustInterval = 10 * time.Second
	}
	if memoryThreshold == 0 {
		// 默认内存阈值设为1GB
		memoryThreshold = 1 << 30
	}

	asm := &EnhancedAdaptiveShardingManager{
		BaseComponent:   NewBaseComponent(),
		counter:        counter,
		config:         cfg,
		minShards:      minShards,
		maxShards:      maxShards,
		currentShards:  atomic.Int32{},
		memoryThreshold: memoryThreshold,
		memoryWeight:   0.4,  // 内存因素权重40%
		qpsWeight:      0.6,  // QPS因素权重60%
		adjustInterval: adjustInterval,
	}

	// 初始设置为最小分片数
	asm.currentShards.Store(int32(minShards))
	asm.UpdateTime() // 使用基础组件的方法更新时间

	// 启动自适应调整协程
	go asm.adaptiveWorker()

	return asm
}

// adaptiveWorker 周期性检查负载并调整分片数量
func (asm *EnhancedAdaptiveShardingManager) adaptiveWorker() {
	ticker := time.NewTicker(asm.adjustInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			asm.adjustShards()
		case <-asm.StopChan(): // 使用基础组件的方法获取停止通道
			return
		}
	}
}

// adjustShards 根据当前QPS和内存使用情况调整分片数量
func (asm *EnhancedAdaptiveShardingManager) adjustShards() {
	// 使用基础组件的方法尝试获取锁
	if !asm.TryLock() {
		return
	}
	defer asm.Unlock()

	currentQPS := asm.counter.CurrentQPS()
	lastQPS := asm.lastQPS.Swap(currentQPS)
	currentShards := asm.currentShards.Load()

	// 获取当前内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	currentMemory := memStats.Alloc
	lastMemory := asm.lastMemoryUsage.Swap(currentMemory)

	// 计算QPS变化率
	var qpsChangeRate float64
	if lastQPS > 0 {
		qpsChangeRate = float64(currentQPS-lastQPS) / float64(lastQPS)
	}

	// 计算内存变化率
	var memoryChangeRate float64
	if lastMemory > 0 {
		memoryChangeRate = float64(currentMemory-lastMemory) / float64(lastMemory)
	}

	// 综合评分，决定是否需要调整分片
	qpsScore := qpsChangeRate * asm.qpsWeight
	memoryScore := memoryChangeRate * asm.memoryWeight
	totalScore := qpsScore + memoryScore

	// 记录当前状态
	logger.Debug("分片管理器状态",
		zap.Int64("current_qps", currentQPS),
		zap.Float64("qps_change_rate", qpsChangeRate),
		zap.Uint64("memory_usage", currentMemory),
		zap.Float64("memory_change_rate", memoryChangeRate),
		zap.Float64("total_score", totalScore),
		zap.Int32("current_shards", currentShards),
	)

	// 根据综合评分调整分片数量
	var newShards int32
	if totalScore > 0.3 && currentShards < int32(asm.maxShards) {
		// 负载增加，增加分片数
		newShards = currentShards + int32(float64(currentShards)*0.5)
		if newShards > int32(asm.maxShards) {
			newShards = int32(asm.maxShards)
		}
	} else if totalScore < -0.3 && currentShards > int32(asm.minShards) {
		// 负载减少，减少分片数
		newShards = currentShards - int32(float64(currentShards)*0.3)
		if newShards < int32(asm.minShards) {
			newShards = int32(asm.minShards)
		}
	} else if currentMemory > asm.memoryThreshold && currentShards > int32(asm.minShards) {
		// 内存使用超过阈值，强制减少分片数到最小值以释放内存
		newShards = int32(asm.minShards)
		logger.Warn("内存使用超过阈值，减少分片数",
			zap.Uint64("memory_usage", currentMemory),
			zap.Uint64("threshold", asm.memoryThreshold),
			zap.Int32("new_shards", newShards),
		)
	} else {
		// 负载变化不大，保持当前分片数
		return
	}

	// 更新分片数量
	if newShards != currentShards {
		asm.currentShards.Store(newShards)
		asm.UpdateTime() // 使用基础组件的方法更新时间
		logger.Info(fmt.Sprintf("自适应调整分片数量: %d -> %d", currentShards, newShards),
			zap.Int64("current_qps", currentQPS),
			zap.Uint64("memory_usage", currentMemory),
			zap.Float64("total_score", totalScore),
		)
	}
}

// Stop 停止自适应分片管理器
func (asm *EnhancedAdaptiveShardingManager) Stop() {
	// 使用基础组件的方法停止组件
	asm.BaseComponent.Stop()
}

// GetCurrentShards 获取当前分片数量
func (asm *EnhancedAdaptiveShardingManager) GetCurrentShards() int32 {
	return asm.currentShards.Load()
}

// GetStats 获取分片管理器状态
func (asm *EnhancedAdaptiveShardingManager) GetStats() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"current_shards":   asm.currentShards.Load(),
		"min_shards":       asm.minShards,
		"max_shards":       asm.maxShards,
		"current_qps":      asm.counter.CurrentQPS(),
		"memory_usage":     memStats.Alloc,
		"memory_threshold": asm.memoryThreshold,
		"last_adjust_time": time.Unix(asm.GetLastUpdateTime(), 0), // 使用基础组件的方法获取上次更新时间
	}
}

// SetMemoryThreshold 设置内存使用阈值
func (asm *EnhancedAdaptiveShardingManager) SetMemoryThreshold(threshold uint64) {
	if threshold > 0 {
		asm.memoryThreshold = threshold
		logger.Info("更新内存阈值", zap.Uint64("new_threshold", threshold))
	}
}

// SetWeights 设置QPS和内存因素的权重
func (asm *EnhancedAdaptiveShardingManager) SetWeights(qpsWeight, memoryWeight float64) {
	if qpsWeight >= 0 && memoryWeight >= 0 && qpsWeight+memoryWeight > 0 {
		// 归一化权重
		total := qpsWeight + memoryWeight
		asm.qpsWeight = qpsWeight / total
		asm.memoryWeight = memoryWeight / total
		logger.Info("更新权重配置", 
			zap.Float64("qps_weight", asm.qpsWeight),
			zap.Float64("memory_weight", asm.memoryWeight))
	}
}