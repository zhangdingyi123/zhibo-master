package service

import (
	"context"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// RoomNotifier 场次实时事件通知（WebSocket 等）
type RoomNotifier interface {
	OnBid(ctx context.Context, result *PlaceBidResult, prevEndAt *time.Time)
	OnCancelled(ctx context.Context, session *domain.AuctionSession, reason string)
}

// NoopRoomNotifier 空实现
type NoopRoomNotifier struct{}

func (NoopRoomNotifier) OnBid(context.Context, *PlaceBidResult, *time.Time) {}
func (NoopRoomNotifier) OnCancelled(context.Context, *domain.AuctionSession, string) {}
