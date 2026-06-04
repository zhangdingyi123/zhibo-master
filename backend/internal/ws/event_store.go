package ws

import "sync"

const defaultEventBuffer = 256

// EventStore 房间事件环形缓冲，供重连增量补偿
type EventStore struct {
	mu     sync.RWMutex
	events []RoomEvent
	seq    uint64
	cap    int
}

func NewEventStore(capacity int) *EventStore {
	if capacity < 32 {
		capacity = defaultEventBuffer
	}
	return &EventStore{cap: capacity}
}

func (s *EventStore) Append(ev RoomEvent) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	ev.Seq = s.seq
	if len(s.events) >= s.cap {
		s.events = s.events[1:]
	}
	s.events = append(s.events, ev)
	return s.seq
}

func (s *EventStore) CurrentSeq() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seq
}

// Since 返回 seq > after 的事件（用于重连补偿）
func (s *EventStore) Since(after uint64) []RoomEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if after >= s.seq {
		return nil
	}
	out := make([]RoomEvent, 0)
	for _, ev := range s.events {
		if ev.Seq > after {
			out = append(out, ev)
		}
	}
	return out
}
