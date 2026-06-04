package ws

import "testing"

func TestEventStore_Since(t *testing.T) {
	s := NewEventStore(4)
	for i := 0; i < 5; i++ {
		s.Append(RoomEvent{Type: EventBidNew})
	}
	if s.CurrentSeq() != 5 {
		t.Fatalf("seq=%d want 5", s.CurrentSeq())
	}
	missed := s.Since(3)
	if len(missed) != 2 {
		t.Fatalf("missed len=%d want 2", len(missed))
	}
	if missed[0].Seq != 4 || missed[1].Seq != 5 {
		t.Fatalf("seqs=%d,%d", missed[0].Seq, missed[1].Seq)
	}
}
