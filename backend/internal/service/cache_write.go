package service

import (
	"context"
	"log"
	"time"

	"github.com/zhibo/backend/internal/infra/metrics"
)

const (
	cacheWriteMaxAttempts = 3
	cacheWriteBaseDelay   = 50 * time.Millisecond
)

// writeCacheWithRetry 缓存写穿失败时指数退避重试；耗尽后打指标并告警日志
func writeCacheWithRetry(ctx context.Context, op string, sessionID uint64, roomID string, fn func() error) {
	if fn == nil {
		return
	}
	metrics.RecordCacheWriteAttempt()

	var lastErr error
	for attempt := 1; attempt <= cacheWriteMaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			metrics.RecordCacheWriteSuccess()
			return
		}
		if attempt < cacheWriteMaxAttempts {
			metrics.RecordCacheWriteRetry()
			delay := cacheWriteBaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				metrics.RecordCacheWriteFailure()
				log.Printf("cache write %s cancelled: session=%d room=%s: %v", op, sessionID, roomID, ctx.Err())
				return
			case <-time.After(delay):
			}
		}
	}

	metrics.RecordCacheWriteFailure()
	log.Printf("ALERT cache write %s failed after %d attempts: session=%d room=%s: %v",
		op, cacheWriteMaxAttempts, sessionID, roomID, lastErr)
}
