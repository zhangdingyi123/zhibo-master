package domain

import "time"

// LiveRoomStatus 直播房间状态
type LiveRoomStatus string

const (
	LiveRoomStatusIdle  LiveRoomStatus = "idle"
	LiveRoomStatusLive  LiveRoomStatus = "live"
	LiveRoomStatusEnded LiveRoomStatus = "ended"
)

// LiveRoom 一场直播（可串联多个竞拍场次）
type LiveRoom struct {
	ID               uint64         `json:"id"`
	AnchorID         uint64         `json:"anchorId"`
	Title            string         `json:"title"`
	RoomID           string         `json:"roomId"`
	Status           LiveRoomStatus `json:"status"`
	CurrentSessionID *uint64        `json:"currentSessionId,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// DefaultLiveRoomID 根据直播房间 ID 生成稳定 roomId
func DefaultLiveRoomID(liveRoomID uint64) string {
	return "room_live_" + formatUint(liveRoomID)
}
