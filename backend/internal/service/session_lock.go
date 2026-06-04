package service

import "context"

// SessionLocker 场次出价互斥（Redis 分布式锁；开发环境可用 NoopLocker）
type SessionLocker interface {
	WithSessionLock(ctx context.Context, sessionID uint64, fn func(context.Context) error) error
}

// NoopLocker 无分布式锁（仅依赖 DB 行锁 + 乐观锁）
type NoopLocker struct{}

func (NoopLocker) WithSessionLock(ctx context.Context, sessionID uint64, fn func(context.Context) error) error {
	return fn(ctx)
}
