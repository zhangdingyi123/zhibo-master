package ws

import (
	"sync"
	"time"
)

// bidRateLimiter 每用户每房间出价限流（4.7）
type bidRateLimiter struct {
	mu      sync.Mutex
	entries map[uint64]time.Time
	minGap  time.Duration
}

func newBidRateLimiter(minGap time.Duration) *bidRateLimiter {
	if minGap <= 0 {
		minGap = 300 * time.Millisecond
	}
	return &bidRateLimiter{
		entries: make(map[uint64]time.Time),
		minGap:  minGap,
	}
}

func (l *bidRateLimiter) allow(userID uint64) bool {
	if userID == 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if last, ok := l.entries[userID]; ok && now.Sub(last) < l.minGap {
		return false
	}
	l.entries[userID] = now
	return true
}
