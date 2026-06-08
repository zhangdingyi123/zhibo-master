package metrics

import (
	"strings"
	"testing"
)

type fakeWS struct{ conn, rooms int }

func (f fakeWS) Stats() (int, int) { return f.conn, f.rooms }

func TestPrometheusText_ContainsMetrics(t *testing.T) {
	RecordBidAttempt()
	RecordCacheWriteAttempt()
	RecordCacheWriteFailure()

	text := PrometheusText(fakeWS{conn: 3, rooms: 1})
	for _, want := range []string{
		"zhibo_bid_attempts_total",
		"zhibo_cache_write_failures_total",
		"zhibo_ws_connections 3",
		"zhibo_ws_rooms 1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}
