package domain

import "time"

type RoomComment struct {
	ID        uint64    `json:"id"`
	RoomID    string    `json:"roomId"`
	UserID    uint64    `json:"userId"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Content   string    `json:"content"`
	IsHidden  bool      `json:"isHidden,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
