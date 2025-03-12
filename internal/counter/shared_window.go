package counter

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mant7s/qps-counter/internal/config"
)

type ShardedWindow struct {
	config     *config.CounterConfig
	shards     []*shard
	stopChan   chan struct{}
	totalCount atomic.Int64 // 添加一个原子计数器来跟踪总请求数
}

type shard struct {
	slots     []*slot
	slotMutex []sync.RWMutex // 使用 RWMutex 替代 Mutex
	shardLock sync.RWMutex   // 添加一个用于保护整个 slots 数组的锁
}

type slot struct {
	timestamp int64
	count     int64
}

func NewSharded(cfg *config.CounterConfig) Counter {
	shardNum := runtime.NumCPU() * 4
	slotNum := cfg.SlotNum

	sw := &ShardedWindow{
		config:   cfg,
		shards:   make([]*shard, shardNum),
		stopChan: make(chan struct{}),
	}

	for i := range sw.shards {
		sw.shards[i] = &shard{
			slots:     make([]*slot, slotNum),
			slotMutex: make([]sync.RWMutex, slotNum),
		}
		for j := range sw.shards[i].slots {
			sw.shards[i].slots[j] = &slot{}
		}
	}

	go sw.cleanupWorker()
	return sw
}

func (sw *ShardedWindow) Incr() {
	// 使用请求时间哈希选择分片
	now := time.Now().UnixNano()
	precisionNano := int64(sw.config.Precision)

	slotTime := now - (now % precisionNano)
	// 使用固定的哈希算法确保分片均匀
	shardID := (now / precisionNano) % int64(len(sw.shards))
	slotID := (now / precisionNano) % int64(sw.config.SlotNum)

	s := sw.shards[shardID]
	s.shardLock.RLock()
	defer s.shardLock.RUnlock()

	s.slotMutex[slotID].Lock()
	defer s.slotMutex[slotID].Unlock()

	// 更新时间戳但不重置计数
	if s.slots[slotID].timestamp < slotTime {
		s.slots[slotID].timestamp = slotTime
	}

	// 增加计数
	s.slots[slotID].count++

	// 同时增加总计数
	sw.totalCount.Add(1)
}

func (sw *ShardedWindow) CurrentQPS() int64 {
	now := time.Now().UnixNano()
	windowStart := now - int64(sw.config.WindowSize)

	var total int64
	for shardID := range sw.shards {
		shard := sw.shards[shardID]
		shard.shardLock.RLock()
		for slotID := range shard.slots {
			// 使用读锁来允许并发读取
			shard.slotMutex[slotID].RLock()
			if shard.slots[slotID].timestamp >= windowStart {
				total += shard.slots[slotID].count
			}
			shard.slotMutex[slotID].RUnlock()
		}
		shard.shardLock.RUnlock()
	}

	// 计算每秒的请求数
	return total * int64(time.Second) / int64(sw.config.WindowSize)
}

func (sw *ShardedWindow) Stop() {
	close(sw.stopChan)
}

func (sw *ShardedWindow) cleanupWorker() {
	ticker := time.NewTicker(sw.config.Precision)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sw.cleanupExpired()
		case <-sw.stopChan:
			return
		}
	}
}

func (sw *ShardedWindow) cleanupExpired() {
	now := time.Now().UnixNano()
	windowStart := now - int64(sw.config.WindowSize)

	// 重置totalCount计数器，避免无限增长
	var newTotal int64

	for shardID := range sw.shards {
		shard := sw.shards[shardID]
		// 创建新的槽位数组以避免内存泄漏
		newSlots := make([]*slot, len(shard.slots))

		shard.shardLock.Lock()
		for slotID := range shard.slots {
			shard.slotMutex[slotID].Lock()

			if shard.slots[slotID].timestamp >= windowStart {
				// 只保留未过期的数据
				newSlots[slotID] = &slot{
					timestamp: shard.slots[slotID].timestamp,
					count:     shard.slots[slotID].count,
				}
				// 累加有效计数到新的总计数
				newTotal += shard.slots[slotID].count
			} else {
				// 为过期槽位创建新的空对象
				newSlots[slotID] = &slot{}
			}

			shard.slotMutex[slotID].Unlock()
		}

		// 替换整个槽位数组
		shard.slots = newSlots
		shard.shardLock.Unlock()
	}

	// 更新总计数
	sw.totalCount.Store(newTotal)
}
