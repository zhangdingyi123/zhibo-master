package domain

import "time"

// Bid 出价记录
type Bid struct {
	ID        uint64    `json:"id"`
	SessionID uint64    `json:"sessionId"`
	UserID    uint64    `json:"userId"`
	Amount    int64     `json:"amount"`    // 分
	RequestID string    `json:"requestId"` // 幂等键
	Seq       uint32    `json:"seq"`
	IsWinning bool      `json:"isWinning"`
	CreatedAt time.Time `json:"createdAt"`
}
