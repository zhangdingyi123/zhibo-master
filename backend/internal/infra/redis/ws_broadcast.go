package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/zhibo/backend/internal/domain"
)

const roomEventBuffer = 256

// IncrRoomEventSeq 分配房间事件序号（多实例共享）
func (c *Client) IncrRoomEventSeq(ctx context.Context, roomID string) (uint64, error) {
	n, err := c.rdb.Incr(ctx, domain.RoomEventSeqKey(roomID)).Result()
	if err != nil {
		return 0, err
	}
	return uint64(n), nil
}

// CurrentRoomEventSeq 当前房间事件序号
func (c *Client) CurrentRoomEventSeq(ctx context.Context, roomID string) (uint64, error) {
	n, err := c.rdb.Get(ctx, domain.RoomEventSeqKey(roomID)).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	if n < 0 {
		return 0, nil
	}
	return uint64(n), nil
}

// StoreRoomEvent 写入事件环形缓冲（多实例共享，供重连补偿）
func (c *Client) StoreRoomEvent(ctx context.Context, roomID string, envelopeJSON []byte) error {
	if len(envelopeJSON) == 0 {
		return fmt.Errorf("empty envelope")
	}
	listKey := domain.RoomEventListKey(roomID)
	pipe := c.rdb.Pipeline()
	pipe.LPush(ctx, listKey, envelopeJSON)
	pipe.LTrim(ctx, listKey, 0, roomEventBuffer-1)
	_, err := pipe.Exec(ctx)
	return err
}

// PublishRoomBroadcast Redis Pub/Sub 广播（无 Kafka 时的降级）
func (c *Client) PublishRoomBroadcast(ctx context.Context, roomID string, envelopeJSON []byte) error {
	return c.rdb.Publish(ctx, domain.RoomBroadcastChannel(roomID), envelopeJSON).Err()
}

type storedEnvelope struct {
	Seq uint64 `json:"seq"`
}

// RoomEventsSince 返回 seq > after 的原始事件 JSON（从 LIST 读出后按 seq 过滤）
func (c *Client) RoomEventsSince(ctx context.Context, roomID string, after uint64) ([][]byte, error) {
	raw, err := c.rdb.LRange(ctx, domain.RoomEventListKey(roomID), 0, roomEventBuffer-1).Result()
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	// LIST 头部最新；按 seq 升序返回便于客户端回放
	type item struct {
		seq  uint64
		data []byte
	}
	items := make([]item, 0, len(raw))
	for _, s := range raw {
		var env storedEnvelope
		if err := json.Unmarshal([]byte(s), &env); err != nil || env.Seq == 0 {
			continue
		}
		if env.Seq > after {
			items = append(items, item{seq: env.Seq, data: []byte(s)})
		}
	}
	// 简单冒泡排序（缓冲 ≤256）
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].seq < items[i].seq {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	out := make([][]byte, len(items))
	for i, it := range items {
		out[i] = it.data
	}
	return out, nil
}

// RoomBroadcastHandler 收到跨实例广播
type RoomBroadcastHandler func(roomID string, envelopeJSON []byte)

// StartRoomBroadcastSubscriber 订阅 zhibo:room:*:broadcast，在独立 goroutine 中回调 handler
func (c *Client) StartRoomBroadcastSubscriber(ctx context.Context, handler RoomBroadcastHandler) error {
	pubsub := c.rdb.PSubscribe(ctx, domain.RoomBroadcastPattern())
	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				roomID := roomIDFromBroadcastChannel(msg.Channel)
				if roomID == "" || handler == nil {
					continue
				}
				handler(roomID, []byte(msg.Payload))
			}
		}
	}()
	return nil
}

func roomIDFromBroadcastChannel(channel string) string {
	// zhibo:room:{roomId}:broadcast
	const prefix = "zhibo:room:"
	const suffix = ":broadcast"
	if !strings.HasPrefix(channel, prefix) || !strings.HasSuffix(channel, suffix) {
		return ""
	}
	inner := strings.TrimPrefix(channel, prefix)
	return strings.TrimSuffix(inner, suffix)
}
