package service

import (
	"testing"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

func TestBuildSnapshot_RemainingMs(t *testing.T) {
	end := time.Now().Add(30 * time.Second)
	session := &domain.AuctionSession{
		ID:           1,
		RoomID:       "room_sess_1",
		Status:       domain.SessionStatusRunning,
		CurrentPrice: 10000,
		BidCount:     2,
		Rules: domain.AuctionRules{
			StartingPrice: 0,
			BidIncrement:  1000,
			DurationSec:   120,
		},
		EndAt: &end,
	}

	now := time.Now()
	snap := BuildSnapshot(session, now)
	if snap.RemainingMs <= 0 || snap.RemainingMs > 30000 {
		t.Fatalf("unexpected remainingMs: %d", snap.RemainingMs)
	}
	if snap.MinNextBid != 11000 {
		t.Fatalf("minNextBid want 11000 got %d", snap.MinNextBid)
	}
}
