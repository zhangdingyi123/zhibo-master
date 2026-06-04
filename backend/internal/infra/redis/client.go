package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhibo/backend/internal/config"
	"github.com/zhibo/backend/internal/domain"
)

// Client Redis 封装（分布式锁、后续缓存）
type Client struct {
	rdb *redis.Client
}

func Open(cfg config.Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

// WithSessionLock 场次出价分布式锁（SET NX + Lua 安全释放）
func (c *Client) WithSessionLock(ctx context.Context, sessionID uint64, fn func(context.Context) error) error {
	key := domain.SessionLockKey(sessionID)
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	ttl := time.Duration(domain.SessionLockTTL) * time.Second

	ok, err := c.rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return fmt.Errorf("session lock: %w", err)
	}
	if !ok {
		return domain.ErrSessionLockBusy
	}
	defer func() {
		script := redis.NewScript(`
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			end
			return 0
		`)
		_, _ = script.Run(context.Background(), c.rdb, []string{key}, token).Result()
	}()

	return fn(ctx)
}
