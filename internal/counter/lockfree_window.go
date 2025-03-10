package counter

import (
	"sync/atomic"
	"time"

	"github.com/mant7s/qps-counter/internal/config"
)

type atomicSlot struct {
	timestamp atomic.Int64
	count     atomic.Int64
}

type LockFreeWindow struct {
	config     *config.CounterConfig
	slots      []atomicSlot
	stopChan   chan struct{}
	totalCount atomic.Int64 // 添加一个原子计数器来跟踪总请求数
}

func NewLockFree(cfg *config.CounterConfig) *LockFreeWindow {
	w := &LockFreeWindow{
		config:   cfg,
		slots:    make([]atomicSlot, cfg.SlotNum),
		stopChan: make(chan struct{}),
	}

	go w.cleanupWorker()
	return w
}

func (lfw *LockFreeWindow) Incr() {
	now := time.Now().UnixNano()
	precision := int64(lfw.config.Precision)
	idx := (now / precision) % int64(len(lfw.slots))

	// CAS更新槽位
	for {
		stored := lfw.slots[idx].timestamp.Load()
		if stored/precision == now/precision {
			lfw.slots[idx].count.Add(1)
			lfw.totalCount.Add(1) // 增加总计数
			return
		}

		if stored == 0 || stored < now-precision {
			if lfw.slots[idx].timestamp.CompareAndSwap(stored, now) {
				lfw.slots[idx].count.Store(1)
				lfw.totalCount.Add(1) // 增加总计数
				return
			}
		}
	}
}

func (lfw *LockFreeWindow) CurrentQPS() int64 {
	// 计算窗口内的实际QPS，而不是简单返回累计值
	now := time.Now().UnixNano()
	windowStart := now - int64(lfw.config.WindowSize)

	var total int64
	for i := range lfw.slots {
		ts := lfw.slots[i].timestamp.Load()
		if ts >= windowStart {
			total += lfw.slots[i].count.Load()
		}
	}

	// 计算每秒的请求数
	return total * int64(time.Second) / int64(lfw.config.WindowSize)
}

func (lfw *LockFreeWindow) Stop() {
	close(lfw.stopChan)
}

func (lfw *LockFreeWindow) cleanupWorker() {
	ticker := time.NewTicker(lfw.config.Precision)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lfw.cleanupExpired()
		case <-lfw.stopChan:
			return
		}
	}
}

func (lfw *LockFreeWindow) cleanupExpired() {
	now := time.Now().UnixNano()
	windowStart := now - int64(lfw.config.WindowSize)

	// 清理过期数据，但不替换整个数组
	for i := range lfw.slots {
		ts := lfw.slots[i].timestamp.Load()
		if ts > 0 && ts < windowStart {
			// 只重置过期的槽位
			lfw.slots[i].timestamp.Store(0)
			lfw.slots[i].count.Store(0)
		}
	}

	// 重新计算总计数
	var newTotal int64
	for i := range lfw.slots {
		if lfw.slots[i].timestamp.Load() >= windowStart {
			newTotal += lfw.slots[i].count.Load()
		}
	}

	// 更新总计数
	lfw.totalCount.Store(newTotal)
}
