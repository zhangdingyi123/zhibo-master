package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRoomCache_SnapshotAndNull(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := &Client{rdb: rdb}
	ctx := context.Background()

	payload := []byte(`{"sessionId":1,"roomId":"room_sess_1","status":"running","currentPrice":5000}`)
	if err := c.SetSnapshotBytes(ctx, "room_sess_1", 1, payload); err != nil {
		t.Fatal(err)
	}
	got, err := c.GetSnapshotByRoom(ctx, "room_sess_1")
	if err != nil || len(got) == 0 {
		t.Fatalf("snapshot: %s err=%v", got, err)
	}

	if err := c.MarkRoomAbsent(ctx, "room_missing"); err != nil {
		t.Fatal(err)
	}
	ok, _ := c.IsRoomAbsent(ctx, "room_missing")
	if !ok {
		t.Fatal("expected absent marker")
	}

	if err := c.InvalidateRoom(ctx, "room_sess_1", 1); err != nil {
		t.Fatal(err)
	}
	got, _ = c.GetSnapshotByRoom(ctx, "room_sess_1")
	if got != nil {
		t.Fatal("expected cache cleared")
	}
}

func TestRoomCache_RankScore(t *testing.T) {
	s1 := rankScore(10000, 1)
	s2 := rankScore(10000, 2)
	s3 := rankScore(9000, 1)
	if s1 <= s2 {
		t.Fatalf("same amount earlier seq should rank higher: %v %v", s1, s2)
	}
	if s1 <= s3 {
		t.Fatalf("higher amount should rank higher: %v %v", s1, s3)
	}
}
