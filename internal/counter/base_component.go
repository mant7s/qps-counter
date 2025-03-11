package counter

import (
	"sync/atomic"
	"time"
)

// BaseComponent 提供基础组件功能，用于减少代码重复
type BaseComponent struct {
	stopChan       chan struct{} // 停止信号通道
	lastUpdateTime atomic.Int64 // 上次更新时间
	statusLock     atomic.Bool  // 状态锁，防止并发操作
}

// NewBaseComponent 创建一个新的基础组件
func NewBaseComponent() *BaseComponent {
	bc := &BaseComponent{
		stopChan: make(chan struct{}),
	}
	bc.lastUpdateTime.Store(time.Now().Unix())
	return bc
}

// StopChan 返回停止信号通道
func (bc *BaseComponent) StopChan() <-chan struct{} {
	return bc.stopChan
}

// Stop 停止组件
func (bc *BaseComponent) Stop() {
	select {
	case <-bc.stopChan:
		// 已经关闭，不需要再次关闭
		return
	default:
		close(bc.stopChan)
	}
}

// UpdateTime 更新最后操作时间
func (bc *BaseComponent) UpdateTime() {
	bc.lastUpdateTime.Store(time.Now().Unix())
}

// GetLastUpdateTime 获取上次更新时间
func (bc *BaseComponent) GetLastUpdateTime() int64 {
	return bc.lastUpdateTime.Load()
}

// TryLock 尝试获取状态锁，如果成功返回true
func (bc *BaseComponent) TryLock() bool {
	return bc.statusLock.CompareAndSwap(false, true)
}

// Unlock 释放状态锁
func (bc *BaseComponent) Unlock() {
	bc.statusLock.Store(false)
}