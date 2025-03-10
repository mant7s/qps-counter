package counter

import (
	"github.com/mant7s/qps-counter/internal/config"
	_ "time"
)

type Counter interface {
	Incr()
	CurrentQPS() int64
	Stop()
}

type Type string

const (
	ShardedType  = "sharded"
	LockFreeType = "lockfree"
)

// NewCounter 配置驱动创建
func NewCounter(cfg *config.CounterConfig) Counter {
	switch cfg.Type {
	case LockFreeType:
		return NewLockFree(cfg)
	default:
		return NewSharded(cfg)
	}
}
