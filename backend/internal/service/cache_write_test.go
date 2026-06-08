package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zhibo/backend/internal/infra/metrics"
)

func TestWriteCacheWithRetry_SuccessFirstTry(t *testing.T) {
	before := metrics.CacheWriteSuccess.Load()
	writeCacheWithRetry(context.Background(), "test", 1, "room_a", func() error { return nil })
	if metrics.CacheWriteSuccess.Load() != before+1 {
		t.Fatalf("expected success counter +1")
	}
}

func TestWriteCacheWithRetry_RetriesThenSucceeds(t *testing.T) {
	retriesBefore := metrics.CacheWriteRetries.Load()
	attempts := 0
	writeCacheWithRetry(context.Background(), "test", 2, "room_b", func() error {
		attempts++
		if attempts < 2 {
			return errors.New("redis timeout")
		}
		return nil
	})
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if metrics.CacheWriteRetries.Load() != retriesBefore+1 {
		t.Fatalf("expected one retry recorded")
	}
}

func TestWriteCacheWithRetry_Exhausted(t *testing.T) {
	failBefore := metrics.CacheWriteFailures.Load()
	writeCacheWithRetry(context.Background(), "test", 3, "room_c", func() error {
		return errors.New("redis down")
	})
	if metrics.CacheWriteFailures.Load() != failBefore+1 {
		t.Fatalf("expected failure counter +1")
	}
}
