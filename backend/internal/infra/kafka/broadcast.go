package kafka

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zhibo/backend/internal/config"
)

const defaultTopic = "zhibo.room.broadcast"

// RoomBroadcaster Kafka 跨实例 WS 广播（每实例独立 Consumer Group，实现 fan-out）
type RoomBroadcaster struct {
	brokers    []string
	topic      string
	instanceID string
	writer     *kafka.Writer
}

func NewRoomBroadcaster(cfg config.Config) (*RoomBroadcaster, error) {
	if len(cfg.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("KAFKA_BROKERS is empty")
	}
	topic := cfg.KafkaTopic
	if topic == "" {
		topic = defaultTopic
	}
	instanceID := cfg.InstanceID
	if instanceID == "" {
		host, err := os.Hostname()
		if err != nil || host == "" {
			instanceID = fmt.Sprintf("ws-%d", time.Now().UnixNano())
		} else {
			instanceID = host
		}
	}

	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.KafkaBrokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           5 * time.Millisecond,
	}

	return &RoomBroadcaster{
		brokers:    cfg.KafkaBrokers,
		topic:      topic,
		instanceID: instanceID,
		writer:     w,
	}, nil
}

func (b *RoomBroadcaster) Publish(ctx context.Context, roomID string, envelopeJSON []byte) error {
	if b == nil || b.writer == nil {
		return fmt.Errorf("kafka broadcaster not initialized")
	}
	return b.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(roomID),
		Value: envelopeJSON,
	})
}

func (b *RoomBroadcaster) StartSubscriber(ctx context.Context, handler func(roomID string, envelopeJSON []byte)) error {
	if b == nil || handler == nil {
		return fmt.Errorf("invalid kafka subscriber")
	}
	// 每实例独立 GroupID → 每个 WS Pod 都收到全量消息（广播语义）
	groupID := "zhibo-ws-" + b.instanceID
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        b.brokers,
		Topic:          b.topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       1 << 20,
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second,
	})

	go func() {
		defer reader.Close()
		log.Printf("kafka: ws subscriber started topic=%s group=%s", b.topic, groupID)
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("kafka: fetch: %v", err)
				time.Sleep(time.Second)
				continue
			}
			roomID := string(msg.Key)
			if roomID == "" {
				_ = reader.CommitMessages(ctx, msg)
				continue
			}
			handler(roomID, msg.Value)
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("kafka: commit: %v", err)
			}
		}
	}()
	return nil
}

func (b *RoomBroadcaster) Close() error {
	if b == nil || b.writer == nil {
		return nil
	}
	return b.writer.Close()
}
