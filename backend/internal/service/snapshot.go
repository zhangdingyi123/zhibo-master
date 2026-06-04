package service

import (
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// SessionSnapshot 场次实时快照（服务端权威）
type SessionSnapshot struct {
	SessionID          uint64               `json:"sessionId"`
	RoomID             string               `json:"roomId"`
	Status             domain.SessionStatus `json:"status"`
	CurrentPrice       int64                `json:"currentPrice"`
	BidCount           uint32               `json:"bidCount"`
	ParticipantCount   uint32               `json:"participantCount"`
	MinNextBid         int64                `json:"minNextBid"`
	Rules              domain.AuctionRules  `json:"rules"`
	EndAtMs            *int64               `json:"endAtMs,omitempty"`
	RemainingMs        int64                `json:"remainingMs"`
	ServerTimeMs       int64                `json:"serverTimeMs"`
	WinnerID           *uint64              `json:"winnerId,omitempty"`
}

func BuildSnapshot(session *domain.AuctionSession, now time.Time) *SessionSnapshot {
	hasBids := session.BidCount > 0
	snap := &SessionSnapshot{
		SessionID:        session.ID,
		RoomID:           session.RoomID,
		Status:           session.Status,
		CurrentPrice:     session.CurrentPrice,
		BidCount:         session.BidCount,
		ParticipantCount: session.ParticipantCount,
		MinNextBid:       session.Rules.MinNextBid(session.CurrentPrice, hasBids),
		Rules:            session.Rules,
		ServerTimeMs:     now.UnixMilli(),
		WinnerID:         session.WinnerID,
	}
	if session.EndAt != nil {
		ms := session.EndAt.UnixMilli()
		snap.EndAtMs = &ms
		if session.Status == domain.SessionStatusRunning {
			remaining := session.EndAt.Sub(now).Milliseconds()
			if remaining < 0 {
				remaining = 0
			}
			snap.RemainingMs = remaining
		}
	}
	return snap
}
