package redis

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhibo/backend/internal/domain"
)

// RankMember 排行榜缓存项（含展示字段，避免读库补全）
type RankMember struct {
	UserID   uint64 `json:"userId"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Amount   int64  `json:"amount"`
	Seq      uint32 `json:"seq"`
	Rank     int    `json:"rank"`
}

// SetSnapshotBytes 写入场次快照 JSON
func (c *Client) SetSnapshotBytes(ctx context.Context, roomID string, sessionID uint64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	pipe := c.rdb.Pipeline()
	pipe.Set(ctx, domain.RoomSnapshotKey(roomID), data, time.Duration(domain.RoomSnapshotTTL)*time.Second)
	if sessionID > 0 {
		pipe.Set(ctx, domain.SessionSnapshotKey(sessionID), data, time.Duration(domain.RoomSnapshotTTL)*time.Second)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// GetSnapshotByRoom 按房间读快照 JSON
func (c *Client) GetSnapshotByRoom(ctx context.Context, roomID string) ([]byte, error) {
	return c.getSnapshotBytes(ctx, domain.RoomSnapshotKey(roomID))
}

// GetSnapshotBySession 按场次 ID 读快照 JSON
func (c *Client) GetSnapshotBySession(ctx context.Context, sessionID uint64) ([]byte, error) {
	return c.getSnapshotBytes(ctx, domain.SessionSnapshotKey(sessionID))
}

func (c *Client) getSnapshotBytes(ctx context.Context, key string) ([]byte, error) {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return b, err
}

// SetCountdownEnd 权威结束时间（毫秒）
func (c *Client) SetCountdownEnd(ctx context.Context, roomID string, endAtMs int64) error {
	return c.rdb.Set(ctx, domain.RoomCountdownKey(roomID), endAtMs, time.Duration(domain.RoomCountdownTTL)*time.Second).Err()
}

// AddParticipant 记录参与用户
func (c *Client) AddParticipant(ctx context.Context, roomID string, userID uint64) error {
	key := domain.RoomParticipantsKey(roomID)
	pipe := c.rdb.Pipeline()
	pipe.SAdd(ctx, key, userID)
	pipe.Expire(ctx, key, time.Duration(domain.RoomSnapshotTTL)*time.Second)
	_, err := pipe.Exec(ctx)
	return err
}

// rankScore ZSET 分数：价高优先，同价 seq 小（先出价）优先
func rankScore(amount int64, seq uint32) float64 {
	return float64(amount)*1e6 + float64(1_000_000-seq)
}

// UpsertRank 更新用户最高出价到排行榜 ZSET
func (c *Client) UpsertRank(ctx context.Context, roomID string, userID uint64, amount int64, seq uint32) error {
	key := domain.RoomRankKey(roomID)
	member := strconv.FormatUint(userID, 10)
	pipe := c.rdb.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: rankScore(amount, seq), Member: member})
	pipe.Expire(ctx, key, time.Duration(domain.RoomSnapshotTTL)*time.Second)
	_, err := pipe.Exec(ctx)
	return err
}

// SetRankTop 缓存 TopN 完整条目（读路径热数据）
func (c *Client) SetRankTop(ctx context.Context, roomID string, items []RankMember) error {
	if len(items) == 0 {
		return c.rdb.Del(ctx, domain.RoomRankTopKey(roomID)).Err()
	}
	b, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, domain.RoomRankTopKey(roomID), b, time.Duration(domain.RoomSnapshotTTL)*time.Second).Err()
}

// GetRankTop 读取 TopN 缓存
func (c *Client) GetRankTop(ctx context.Context, roomID string) ([]RankMember, error) {
	b, err := c.rdb.Get(ctx, domain.RoomRankTopKey(roomID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []RankMember
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// MarkRoomAbsent 空值缓存，防穿透（5.3）
func (c *Client) MarkRoomAbsent(ctx context.Context, roomID string) error {
	return c.rdb.Set(ctx, domain.RoomNullKey(roomID), "1", time.Duration(domain.NullCacheTTL)*time.Second).Err()
}

// IsRoomAbsent 是否命中空值缓存
func (c *Client) IsRoomAbsent(ctx context.Context, roomID string) (bool, error) {
	n, err := c.rdb.Exists(ctx, domain.RoomNullKey(roomID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// InvalidateRoom 场次结束/取消时清理房间缓存
func (c *Client) InvalidateRoom(ctx context.Context, roomID string, sessionID uint64) error {
	keys := []string{
		domain.RoomSnapshotKey(roomID),
		domain.RoomRankKey(roomID),
		domain.RoomRankTopKey(roomID),
		domain.RoomCountdownKey(roomID),
		domain.RoomParticipantsKey(roomID),
		domain.RoomNullKey(roomID),
	}
	if sessionID > 0 {
		keys = append(keys, domain.SessionSnapshotKey(sessionID))
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// SetCountdownFromSession 写入结束时间戳
func (c *Client) SetCountdownFromSession(ctx context.Context, roomID string, endAt *time.Time) error {
	if endAt == nil {
		return nil
	}
	return c.SetCountdownEnd(ctx, roomID, endAt.UnixMilli())
}
