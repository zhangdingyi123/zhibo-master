package domain

import "fmt"

// Redis Key 命名规范（阶段 1.4）
// 前缀 zhibo: 便于多环境隔离；roomId 与 auction_sessions.room_id 一致。

const redisPrefix = "zhibo"

// RoomSnapshotKey 场次快照（Hash）：当前价、人数、状态、endAt 等
// TTL：场次结束后保留 24h 供重连补偿
func RoomSnapshotKey(roomID string) string {
	return fmt.Sprintf("%s:room:%s:snapshot", redisPrefix, roomID)
}

// RoomRankKey 出价排行榜（ZSET）：member=userId, score=amount（同价按时间戳 tie-break 在 value 中编码）
func RoomRankKey(roomID string) string {
	return fmt.Sprintf("%s:room:%s:rank", redisPrefix, roomID)
}

// RoomCountdownKey 权威倒计时结束时间戳（String，Unix 毫秒）
func RoomCountdownKey(roomID string) string {
	return fmt.Sprintf("%s:room:%s:countdown", redisPrefix, roomID)
}

// RoomParticipantsKey 参与用户集合（SET）
func RoomParticipantsKey(roomID string) string {
	return fmt.Sprintf("%s:room:%s:participants", redisPrefix, roomID)
}

// SessionLockKey 场次出价分布式锁（String，SET NX EX）
func SessionLockKey(sessionID uint64) string {
	return fmt.Sprintf("%s:lock:session:%d", redisPrefix, sessionID)
}

// BidIdempotentKey 出价幂等（String，requestId -> bidId）
func BidIdempotentKey(sessionID uint64, requestID string) string {
	return fmt.Sprintf("%s:bid:idem:%d:%s", redisPrefix, sessionID, requestID)
}

// RoomEventSeqKey 房间事件序号（INCR），用于 WS 增量推送与重连补偿
func RoomEventSeqKey(roomID string) string {
	return fmt.Sprintf("%s:room:%s:seq", redisPrefix, roomID)
}

// SessionSnapshotKey 按场次 ID 的快照（与 RoomSnapshotKey 双写，便于按 ID 查询）
func SessionSnapshotKey(sessionID uint64) string {
	return fmt.Sprintf("%s:session:%d:snapshot", redisPrefix, sessionID)
}

// RoomRankTopKey TopN 排行榜 JSON 缓存（含昵称头像，降低 JOIN 读压）
func RoomRankTopKey(roomID string) string {
	return fmt.Sprintf("%s:room:%s:rank_top", redisPrefix, roomID)
}

// RoomNullKey 空值标记，防止缓存穿透
func RoomNullKey(roomID string) string {
	return fmt.Sprintf("%s:room:%s:null", redisPrefix, roomID)
}

// SessionBidSeqKey 场次内出价序号（INCR）
func SessionBidSeqKey(sessionID uint64) string {
	return fmt.Sprintf("%s:session:%d:bid_seq", redisPrefix, sessionID)
}

// RedisTTL 建议过期时间（秒）
const (
	RoomSnapshotTTL   = 86400 // 24h
	SessionLockTTL    = 5     // 锁 5s
	BidIdempotentTTL  = 3600  // 幂等 1h
	RoomCountdownTTL  = 86400
	NullCacheTTL      = 60    // 空值缓存 60s
)
