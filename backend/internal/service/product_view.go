package service

import (
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// ProductView 商品 + 竞拍进度/成交信息（管理端列表/详情）
type ProductView struct {
	domain.Product
	Auction *AuctionProgress `json:"auction,omitempty"`
}

// AuctionProgress 场次进度摘要
type AuctionProgress struct {
	SessionID          uint64               `json:"sessionId"`
	RoomID             string               `json:"roomId"`
	Status             domain.SessionStatus `json:"status"`
	CurrentPrice       int64                `json:"currentPrice"`
	BidCount           uint32               `json:"bidCount"`
	ParticipantCount   uint32               `json:"participantCount"`
	ScheduledStartAt   *time.Time           `json:"scheduledStartAt,omitempty"`
	StartedAt          *time.Time           `json:"startedAt,omitempty"`
	EndAt              *time.Time           `json:"endAt,omitempty"`
	SettledAt          *time.Time           `json:"settledAt,omitempty"`
	WinnerID           *uint64              `json:"winnerId,omitempty"`
	CancelReason       *string              `json:"cancelReason,omitempty"`
	Order              *domain.Order        `json:"order,omitempty"`
}
