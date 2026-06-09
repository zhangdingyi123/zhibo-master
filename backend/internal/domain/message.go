package domain

import "time"

type MessageCategory string

const (
	MessageCategoryAuction MessageCategory = "auction"
	MessageCategoryOrder   MessageCategory = "order"
	MessageCategorySystem  MessageCategory = "system"
)

type MessageEventType string

const (
	MessageOutbid      MessageEventType = "outbid"
	MessageExtended    MessageEventType = "extended"
	MessageSettledWin  MessageEventType = "settled_win"
	MessageSettled     MessageEventType = "settled"
	MessageCancelled   MessageEventType = "cancelled"
	MessageOrderShipped   MessageEventType = "order_shipped"
	MessageOrderCancelled MessageEventType = "order_cancelled"
	MessageOrderRefunded  MessageEventType = "order_refunded"
)

type UserMessage struct {
	ID        uint64          `json:"id"`
	UserID    uint64          `json:"userId"`
	EventType MessageEventType `json:"eventType"`
	Category  MessageCategory `json:"category"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Payload   map[string]any  `json:"payload,omitempty"`
	IsRead    bool            `json:"isRead"`
	CreatedAt time.Time       `json:"createdAt"`
}
