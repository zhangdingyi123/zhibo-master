package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/infra/metrics"
	redisc "github.com/zhibo/backend/internal/infra/redis"
	"github.com/zhibo/backend/internal/repository"
	"golang.org/x/sync/singleflight"
)

func recordCacheHit()  { metrics.RecordCacheHit() }
func recordCacheMiss() { metrics.RecordCacheMiss() }

// RoomCache 房间热数据缓存（5.1）；nil 实现表示直读 DB
type RoomCache interface {
	GetSnapshotByRoom(ctx context.Context, roomID string) (*SessionSnapshot, error)
	GetSnapshotBySession(ctx context.Context, sessionID uint64) (*SessionSnapshot, error)
	RefreshFromSession(ctx context.Context, session *domain.AuctionSession) error
	OnBid(ctx context.Context, session *domain.AuctionSession, userID uint64, amount int64, seq uint32, newParticipant bool) error
	SetRankTop(ctx context.Context, roomID string, items []redisc.RankMember) error
	GetRankTop(ctx context.Context, roomID string) ([]redisc.RankMember, error)
	Invalidate(ctx context.Context, roomID string, sessionID uint64) error
}

// RedisRoomCache Redis 读写 + 击穿保护
type RedisRoomCache struct {
	rdb      *redisc.Client
	sessions *repository.SessionRepo
	sf       singleflight.Group
}

func NewRedisRoomCache(rdb *redisc.Client, sessions *repository.SessionRepo) *RedisRoomCache {
	return &RedisRoomCache{rdb: rdb, sessions: sessions}
}

func (c *RedisRoomCache) GetSnapshotByRoom(ctx context.Context, roomID string) (*SessionSnapshot, error) {
	if absent, _ := c.rdb.IsRoomAbsent(ctx, roomID); absent {
		return nil, domain.ErrNotFound
	}
	raw, err := c.rdb.GetSnapshotByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if snap := decodeSnapshot(raw); snap != nil {
		recordCacheHit()
		enrichSnapshotTiming(snap)
		return snap, nil
	}
	recordCacheMiss()
	return c.loadSnapshotByRoom(ctx, roomID)
}

func (c *RedisRoomCache) GetSnapshotBySession(ctx context.Context, sessionID uint64) (*SessionSnapshot, error) {
	raw, err := c.rdb.GetSnapshotBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if snap := decodeSnapshot(raw); snap != nil {
		recordCacheHit()
		enrichSnapshotTiming(snap)
		return snap, nil
	}
	recordCacheMiss()
	session, err := c.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return c.GetSnapshotByRoom(ctx, session.RoomID)
}

func decodeSnapshot(raw []byte) *SessionSnapshot {
	if len(raw) == 0 {
		return nil
	}
	var snap SessionSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil
	}
	return &snap
}

func encodeSnapshot(snap *SessionSnapshot) ([]byte, error) {
	return json.Marshal(snap)
}

func (c *RedisRoomCache) loadSnapshotByRoom(ctx context.Context, roomID string) (*SessionSnapshot, error) {
	key := "snap:" + roomID
	v, err, _ := c.sf.Do(key, func() (any, error) {
		if absent, _ := c.rdb.IsRoomAbsent(ctx, roomID); absent {
			return nil, domain.ErrNotFound
		}
		raw, err := c.rdb.GetSnapshotByRoom(ctx, roomID)
		if err != nil {
			return nil, err
		}
		if snap := decodeSnapshot(raw); snap != nil {
			enrichSnapshotTiming(snap)
			return snap, nil
		}
		session, err := c.sessions.GetByRoomID(ctx, roomID)
		if err != nil {
			if err == domain.ErrNotFound {
				_ = c.rdb.MarkRoomAbsent(ctx, roomID)
			}
			return nil, err
		}
		snap := BuildSnapshot(session, time.Now())
		if b, err := encodeSnapshot(snap); err == nil {
			_ = c.rdb.SetSnapshotBytes(ctx, roomID, session.ID, b)
		}
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*SessionSnapshot), nil
}

func (c *RedisRoomCache) RefreshFromSession(ctx context.Context, session *domain.AuctionSession) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	snap := BuildSnapshot(session, time.Now())
	b, err := encodeSnapshot(snap)
	if err != nil {
		return err
	}
	if err := c.rdb.SetSnapshotBytes(ctx, session.RoomID, session.ID, b); err != nil {
		return err
	}
	return c.rdb.SetCountdownFromSession(ctx, session.RoomID, session.EndAt)
}

func (c *RedisRoomCache) OnBid(ctx context.Context, session *domain.AuctionSession, userID uint64, amount int64, seq uint32, newParticipant bool) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	snap := BuildSnapshot(session, time.Now())
	if b, err := encodeSnapshot(snap); err != nil {
		return err
	} else if err := c.rdb.SetSnapshotBytes(ctx, session.RoomID, session.ID, b); err != nil {
		return err
	}
	_ = c.rdb.SetCountdownFromSession(ctx, session.RoomID, session.EndAt)
	if err := c.rdb.UpsertRank(ctx, session.RoomID, userID, amount, seq); err != nil {
		return err
	}
	if newParticipant {
		_ = c.rdb.AddParticipant(ctx, session.RoomID, userID)
	}
	// TopN JSON 在 Notifier 写库后回填；此处使旧 Top 失效
	return c.rdb.SetRankTop(ctx, session.RoomID, nil)
}

func (c *RedisRoomCache) SetRankTop(ctx context.Context, roomID string, items []redisc.RankMember) error {
	return c.rdb.SetRankTop(ctx, roomID, items)
}

func (c *RedisRoomCache) GetRankTop(ctx context.Context, roomID string) ([]redisc.RankMember, error) {
	return c.rdb.GetRankTop(ctx, roomID)
}

func (c *RedisRoomCache) Invalidate(ctx context.Context, roomID string, sessionID uint64) error {
	return c.rdb.InvalidateRoom(ctx, roomID, sessionID)
}

// enrichSnapshotTiming 缓存中的 remainingMs / serverTimeMs 按请求时刻重算
func enrichSnapshotTiming(snap *SessionSnapshot) {
	now := time.Now()
	snap.ServerTimeMs = now.UnixMilli()
	if snap.EndAtMs != nil && snap.Status == domain.SessionStatusRunning {
		remaining := *snap.EndAtMs - snap.ServerTimeMs
		if remaining < 0 {
			remaining = 0
		}
		snap.RemainingMs = remaining
	}
}
