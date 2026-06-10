package ws

import "context"

// RoomBroadcaster 跨实例房间事件广播（Kafka 等）；本机 WS 连接仍由 Hub 维护
type RoomBroadcaster interface {
	Publish(ctx context.Context, roomID string, envelopeJSON []byte) error
	StartSubscriber(ctx context.Context, handler func(roomID string, envelopeJSON []byte)) error
}
