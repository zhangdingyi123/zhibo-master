package ws

import (
	"encoding/json"

	"github.com/zhibo/backend/internal/service"
)

// 客户端 → 服务端
const (
	ClientSubscribe = "subscribe"
	ClientPing      = "ping"
	ClientBid       = "bid"
)

// 服务端 → 客户端
const (
	ServerConnected = "connected"
	ServerSync      = "sync"
	ServerEvent     = "event"
	ServerPong      = "pong"
	ServerError     = "error"
)

// 房间事件类型（4.4）
const (
	EventBidNew          = "bid.new"
	EventRankUpdate      = "rank.update"
	EventCountdownTick   = "countdown.tick"
	EventAuctionExtended = "auction.extended"
	EventAuctionSettled  = "auction.settled"
	EventAuctionCancelled = "auction.cancelled"
	EventSessionSwitch    = "session.switch"
	EventCommentNew       = "comment.new"
	EventLikeUpdate       = "like.update"
	EventCommentHidden    = "comment.hidden"
)

// Envelope 统一消息信封
type Envelope struct {
	Type      string          `json:"type"`
	ClientID  string          `json:"clientId,omitempty"`
	RoomID    string          `json:"roomId,omitempty"`
	Seq       uint64          `json:"seq,omitempty"`
	LastSeq   uint64          `json:"lastSeq,omitempty"`
	Timestamp int64           `json:"ts,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// ConnectedPayload 连接成功
type ConnectedPayload struct {
	RoomID     string `json:"roomId"`
	SessionID  uint64 `json:"sessionId"`
	CurrentSeq uint64 `json:"currentSeq"`
	UserID     uint64 `json:"userId,omitempty"`
}

// SyncPayload 重连补偿（4.6）
type SyncPayload struct {
	Snapshot *service.SessionSnapshot `json:"snapshot"`
	Events   []RoomEvent              `json:"events"`
}

// RoomEvent 带序号的房间事件
type RoomEvent struct {
	Seq     uint64          `json:"seq"`
	Type    string          `json:"type"`
	Ts      int64           `json:"ts"`
	Payload json.RawMessage `json:"payload"`
}

// SubscribePayload 订阅房间
type SubscribePayload struct {
	RoomID   string `json:"roomId"`
	ClientID string `json:"clientId"`
	LastSeq  uint64 `json:"lastSeq"`
}

// PingPayload 心跳
type PingPayload struct {
	ClientID string `json:"clientId"`
	LastSeq  uint64 `json:"lastSeq"`
}

// BidPayload WS 出价
type BidPayload struct {
	Amount    int64  `json:"amount"`
	RequestID string `json:"requestId"`
}

// ErrorPayload 错误
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func marshalPayload(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
