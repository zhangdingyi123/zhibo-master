package service

import (
	"context"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// CompositeRoomNotifier WS 广播 + 写扩散消息
type CompositeRoomNotifier struct {
	realtime RoomNotifier
	messages *MessageService
}

func NewCompositeRoomNotifier(realtime RoomNotifier, messages *MessageService) *CompositeRoomNotifier {
	return &CompositeRoomNotifier{realtime: realtime, messages: messages}
}

func (n *CompositeRoomNotifier) OnBid(ctx context.Context, result *PlaceBidResult, prevEndAt *time.Time) {
	n.OnBidWithPrevWinner(ctx, result, prevEndAt, nil)
}

func (n *CompositeRoomNotifier) OnBidWithPrevWinner(ctx context.Context, result *PlaceBidResult, prevEndAt *time.Time, prevWinnerID *uint64) {
	if n.realtime != nil {
		n.realtime.OnBid(ctx, result, prevEndAt)
	}
	if n.messages != nil {
		n.messages.FanOutOnBid(ctx, result, prevEndAt, prevWinnerID)
	}
}

func (n *CompositeRoomNotifier) OnCancelled(ctx context.Context, session *domain.AuctionSession, reason string) {
	if n.realtime != nil {
		n.realtime.OnCancelled(ctx, session, reason)
	}
	if n.messages != nil {
		n.messages.FanOutOnCancelled(ctx, session, reason)
	}
}
