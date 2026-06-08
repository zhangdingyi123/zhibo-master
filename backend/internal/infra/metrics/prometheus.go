package metrics

import (
	"fmt"
	"strings"
)

// PrometheusText 输出 Prometheus 文本格式，供 Grafana / Alertmanager 抓取
func PrometheusText(ws WSCollector) string {
	s := Collect(ws)
	var b strings.Builder
	writeCounter := func(name, help string, v uint64) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, v)
	}
	writeGauge := func(name, help string, v float64) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s gauge\n%s %g\n", name, help, name, name, v)
	}

	writeCounter("zhibo_bid_attempts_total", "Total bid attempts (REST + WS)", s.BidAttempts)
	writeCounter("zhibo_bid_success_total", "Successful bids", s.BidSuccess)
	writeCounter("zhibo_bid_failures_total", "Failed bids", s.BidFailures)
	writeGauge("zhibo_bid_failure_rate", "Bid failure rate (failures/attempts)", s.BidFailureRate)

	writeCounter("zhibo_cache_hits_total", "Snapshot cache hits", s.CacheHits)
	writeCounter("zhibo_cache_misses_total", "Snapshot cache misses", s.CacheMisses)
	writeCounter("zhibo_cache_write_attempts_total", "Cache write attempts (on_bid, refresh, etc.)", s.CacheWriteAttempts)
	writeCounter("zhibo_cache_write_success_total", "Successful cache writes", s.CacheWriteSuccess)
	writeCounter("zhibo_cache_write_failures_total", "Cache write failures after retries", s.CacheWriteFailures)
	writeCounter("zhibo_cache_write_retries_total", "Cache write retry count", s.CacheWriteRetries)
	writeGauge("zhibo_cache_write_failure_rate", "Cache write failure rate", s.CacheWriteFailRate)

	writeGauge("zhibo_ws_connections", "Active WebSocket connections", float64(s.WSConnections))
	writeGauge("zhibo_ws_rooms", "Active WebSocket rooms", float64(s.WSRooms))

	return b.String()
}
