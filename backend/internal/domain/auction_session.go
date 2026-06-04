package domain

import "time"

// AuctionSession 竞拍场次
type AuctionSession struct {
	ID        uint64        `json:"id"`
	ProductID uint64        `json:"productId"`
	AnchorID  uint64        `json:"anchorId"`
	RoomID    string        `json:"roomId"`
	Status    SessionStatus `json:"status"`

	Rules AuctionRules `json:"rules"`

	CurrentPrice       int64  `json:"currentPrice"`
	BidCount           uint32 `json:"bidCount"`
	ParticipantCount   uint32 `json:"participantCount"`
	WinnerID           *uint64 `json:"winnerId,omitempty"`
	Version            uint32 `json:"version"`

	ScheduledStartAt *time.Time `json:"scheduledStartAt,omitempty"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	EndAt            *time.Time `json:"endAt,omitempty"`
	SettledAt        *time.Time `json:"settledAt,omitempty"`
	CancelReason     *string    `json:"cancelReason,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// HasBids 是否已有出价
func (s *AuctionSession) HasBids() bool {
	return s.BidCount > 0
}

// TransitionTo 尝试迁移状态，非法迁移返回 ErrInvalidStateTransition
func (s *AuctionSession) TransitionTo(next SessionStatus) error {
	if !CanTransition(s.Status, next) {
		return ErrInvalidStateTransition
	}
	s.Status = next
	return nil
}

// DefaultRoomID 根据场次 ID 生成默认房间号
func DefaultRoomID(sessionID uint64) string {
	return "room_sess_" + formatUint(sessionID)
}

func formatUint(id uint64) string {
	if id == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	return string(buf[i:])
}
