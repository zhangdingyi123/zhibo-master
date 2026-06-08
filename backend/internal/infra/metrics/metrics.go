package metrics

import (
	"encoding/json"
	"sync/atomic"
)

// 全局计数器（5.6 可观测）
var (
	BidAttempts        atomic.Uint64
	BidSuccess         atomic.Uint64
	BidFailures        atomic.Uint64
	CacheHits          atomic.Uint64
	CacheMisses        atomic.Uint64
	CacheWriteAttempts atomic.Uint64
	CacheWriteSuccess  atomic.Uint64
	CacheWriteFailures atomic.Uint64
	CacheWriteRetries  atomic.Uint64
)

func RecordBidAttempt()  { BidAttempts.Add(1) }
func RecordBidSuccess()  { BidSuccess.Add(1) }
func RecordBidFailure() { BidFailures.Add(1) }
func RecordCacheHit()    { CacheHits.Add(1) }
func RecordCacheMiss()   { CacheMisses.Add(1) }

func RecordCacheWriteAttempt() { CacheWriteAttempts.Add(1) }
func RecordCacheWriteSuccess() { CacheWriteSuccess.Add(1) }
func RecordCacheWriteFailure() { CacheWriteFailures.Add(1) }
func RecordCacheWriteRetry()   { CacheWriteRetries.Add(1) }

// Snapshot 当前指标快照
type Snapshot struct {
	BidAttempts         uint64  `json:"bidAttempts"`
	BidSuccess          uint64  `json:"bidSuccess"`
	BidFailures         uint64  `json:"bidFailures"`
	BidFailureRate      float64 `json:"bidFailureRate"`
	CacheHits           uint64  `json:"cacheHits"`
	CacheMisses         uint64  `json:"cacheMisses"`
	CacheWriteAttempts  uint64  `json:"cacheWriteAttempts"`
	CacheWriteSuccess   uint64  `json:"cacheWriteSuccess"`
	CacheWriteFailures  uint64  `json:"cacheWriteFailures"`
	CacheWriteRetries   uint64  `json:"cacheWriteRetries"`
	CacheWriteFailRate  float64 `json:"cacheWriteFailRate"`
	WSConnections       int     `json:"wsConnections"`
	WSRooms             int     `json:"wsRooms"`
}

// WSCollector WebSocket 连接统计
type WSCollector interface {
	Stats() (connections int, rooms int)
}

func Collect(ws WSCollector) Snapshot {
	attempts := BidAttempts.Load()
	failures := BidFailures.Load()
	var bidFailRate float64
	if attempts > 0 {
		bidFailRate = float64(failures) / float64(attempts)
	}
	writeAttempts := CacheWriteAttempts.Load()
	writeFailures := CacheWriteFailures.Load()
	var writeFailRate float64
	if writeAttempts > 0 {
		writeFailRate = float64(writeFailures) / float64(writeAttempts)
	}
	s := Snapshot{
		BidAttempts:        attempts,
		BidSuccess:         BidSuccess.Load(),
		BidFailures:        failures,
		BidFailureRate:     bidFailRate,
		CacheHits:          CacheHits.Load(),
		CacheMisses:        CacheMisses.Load(),
		CacheWriteAttempts: writeAttempts,
		CacheWriteSuccess:  CacheWriteSuccess.Load(),
		CacheWriteFailures: writeFailures,
		CacheWriteRetries:  CacheWriteRetries.Load(),
		CacheWriteFailRate: writeFailRate,
	}
	if ws != nil {
		s.WSConnections, s.WSRooms = ws.Stats()
	}
	return s
}

func (s Snapshot) JSON() ([]byte, error) {
	return json.Marshal(s)
}
