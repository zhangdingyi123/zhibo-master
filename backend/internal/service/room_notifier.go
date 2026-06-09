package service

import (
	"context"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// RoomNotifier 场次实时事件通知（WebSocket 等）
type RoomNotifier interface {
	OnBid(ctx context.Context, result *PlaceBidResult, prevEndAt *time.Time)
	OnSettled(ctx context.Context, session *domain.AuctionSession, order *domain.Order)
	OnCancelled(ctx context.Context, session *domain.AuctionSession, reason string)
	OnSessionSwitch(ctx context.Context, liveRoom *domain.LiveRoom, previous *SessionSummary, current *UserAuctionDetail, history []SessionSummary)
}

// BidAwareNotifier 支持写扩散前快照（领先者）
type BidAwareNotifier interface {
	RoomNotifier
	OnBidWithPrevWinner(ctx context.Context, result *PlaceBidResult, prevEndAt *time.Time, prevWinnerID *uint64)
}

// NoopRoomNotifier 空实现
type NoopRoomNotifier struct{}

func (NoopRoomNotifier) OnBid(context.Context, *PlaceBidResult, *time.Time)       {}
func (NoopRoomNotifier) OnSettled(context.Context, *domain.AuctionSession, *domain.Order) {}
func (NoopRoomNotifier) OnCancelled(context.Context, *domain.AuctionSession, string) {}
func (NoopRoomNotifier) OnSessionSwitch(context.Context, *domain.LiveRoom, *SessionSummary, *UserAuctionDetail, []SessionSummary) {
}
