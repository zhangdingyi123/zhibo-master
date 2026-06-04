package engine

import (
	"testing"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

func cap(v int64) *int64 { return &v }

func rulesZeroStart() domain.AuctionRules {
	return domain.AuctionRules{
		StartingPrice:      0,
		BidIncrement:       1000,
		DurationSec:        120,
		ExtendThresholdSec: 10,
		ExtendSec:          30,
	}
}

// 3.1 0 元起拍：首笔 ≥ 0，后续按加价幅度
func TestEvaluateBid_ZeroStartingPrice(t *testing.T) {
	r := rulesZeroStart()
	now := time.Now()

	out, err := EvaluateBid(SessionView{Status: domain.SessionStatusPending, Rules: r}, 0, now)
	if err != nil {
		t.Fatalf("first bid 0: %v", err)
	}
	if out.AcceptedAmount != 0 || out.Settled {
		t.Fatalf("first bid: amount=%d settled=%v", out.AcceptedAmount, out.Settled)
	}
	if out.Status != domain.SessionStatusRunning {
		t.Fatalf("status want running got %s", out.Status)
	}

	s := SessionView{
		Status: domain.SessionStatusRunning, CurrentPrice: 0, BidCount: 1,
		Rules: r, EndAt: out.EndAt,
	}
	_, err = EvaluateBid(s, 500, now)
	if err != domain.ErrBidTooLow {
		t.Fatalf("second bid below increment: got %v", err)
	}
	out2, err := EvaluateBid(s, 1000, now)
	if err != nil || out2.AcceptedAmount != 1000 {
		t.Fatalf("second bid 1000: err=%v amount=%d", err, out2.AcceptedAmount)
	}
}

// 3.2 加价幅度边界
func TestEvaluateBid_IncrementBoundary(t *testing.T) {
	capPrice := int64(100000)
	r := domain.AuctionRules{
		StartingPrice: 10000, BidIncrement: 500, CapPrice: &capPrice,
		DurationSec: 60, ExtendThresholdSec: 10, ExtendSec: 15,
	}
	end := time.Now().Add(time.Minute)
	s := SessionView{
		Status: domain.SessionStatusRunning, CurrentPrice: 10000, BidCount: 1,
		Rules: r, EndAt: &end,
	}
	now := time.Now()

	if _, err := EvaluateBid(s, 10499, now); err != domain.ErrBidTooLow {
		t.Fatalf("below min want ErrBidTooLow got %v", err)
	}
	out, err := EvaluateBid(s, 10500, now)
	if err != nil || out.AcceptedAmount != 10500 {
		t.Fatalf("exact min increment: %v %d", err, out.AcceptedAmount)
	}
}

// 3.3 封顶立即成交，不延时
func TestEvaluateBid_CapSettlesWithoutExtend(t *testing.T) {
	r := rulesZeroStart()
	r.CapPrice = cap(5000)
	now := time.Now()
	end := now.Add(5 * time.Second) // 在延时窗口内
	s := SessionView{
		Status: domain.SessionStatusRunning, CurrentPrice: 3000, BidCount: 2,
		Rules: r, EndAt: &end,
	}

	out, err := EvaluateBid(s, 5000, now)
	if err != nil {
		t.Fatalf("cap bid: %v", err)
	}
	if !out.Settled || out.Status != domain.SessionStatusSettled {
		t.Fatalf("want settled, got settled=%v status=%s", out.Settled, out.Status)
	}
	if out.AcceptedAmount != 5000 {
		t.Fatalf("accepted=%d want 5000", out.AcceptedAmount)
	}
	// 封顶不应延长 endAt（保持原值或 nil 变化 — 与输入 end 相同时间点语义）
	if out.EndAt != nil && out.EndAt.After(end.Add(time.Second)) {
		t.Fatalf("cap hit must not extend endAt: was %v extended to %v", end, *out.EndAt)
	}
}

func TestEvaluateBid_CapClampsOverBid(t *testing.T) {
	r := rulesZeroStart()
	r.CapPrice = cap(5000)
	end := time.Now().Add(time.Minute)
	s := SessionView{
		Status: domain.SessionStatusRunning, CurrentPrice: 4000, BidCount: 1,
		Rules: r, EndAt: &end,
	}
	out, err := EvaluateBid(s, 5000, time.Now())
	if err != nil || out.AcceptedAmount != 5000 || !out.Settled {
		t.Fatalf("cap settle: %+v err=%v", out, err)
	}
}

// 3.4 自动延时：结束前 N 秒内有出价 → 延长
func TestMaybeExtendEnd_WithinThreshold(t *testing.T) {
	r := rulesZeroStart()
	now := time.Now()
	endAt := now.Add(8 * time.Second)

	newEnd := MaybeExtendEnd(endAt, now, r)
	if newEnd == nil {
		t.Fatal("nil end")
	}
	want := endAt.Add(30 * time.Second)
	if !newEnd.Equal(want) {
		t.Fatalf("extended end want %v got %v", want, *newEnd)
	}
}

func TestMaybeExtendEnd_OutsideThreshold(t *testing.T) {
	r := rulesZeroStart()
	now := time.Now()
	endAt := now.Add(20 * time.Second)

	newEnd := MaybeExtendEnd(endAt, now, r)
	if !newEnd.Equal(endAt) {
		t.Fatalf("should not extend: got %v", *newEnd)
	}
}

func TestEvaluateBid_ExtendOnRunningBid(t *testing.T) {
	r := rulesZeroStart()
	now := time.Now()
	endAt := now.Add(8 * time.Second)
	s := SessionView{
		Status: domain.SessionStatusRunning, CurrentPrice: 0, BidCount: 1,
		Rules: r, EndAt: &endAt,
	}
	out, err := EvaluateBid(s, 1000, now)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if out.EndAt == nil || !out.EndAt.After(endAt) {
		t.Fatalf("expected extended endAt, was %v", out.EndAt)
	}
}

// 3.5 已取消/终态不可出价
func TestEvaluateBid_NotBiddable(t *testing.T) {
	r := rulesZeroStart()
	now := time.Now()
	for _, st := range []domain.SessionStatus{
		domain.SessionStatusCancelled,
		domain.SessionStatusSettled,
		domain.SessionStatusFailed,
	} {
		_, err := EvaluateBid(SessionView{Status: st, Rules: r}, 1000, now)
		if err != domain.ErrSessionNotBiddable {
			t.Fatalf("status %s: want ErrSessionNotBiddable got %v", st, err)
		}
	}
}

// 竞拍已结束
func TestEvaluateBid_AuctionEnded(t *testing.T) {
	r := rulesZeroStart()
	now := time.Now()
	past := now.Add(-time.Second)
	s := SessionView{
		Status: domain.SessionStatusRunning, Rules: r, EndAt: &past,
	}
	_, err := EvaluateBid(s, 1000, now)
	if err != domain.ErrAuctionEnded {
		t.Fatalf("want ErrAuctionEnded got %v", err)
	}
}

func TestEvaluateBid_ExceedsCap(t *testing.T) {
	r := rulesZeroStart()
	r.CapPrice = cap(5000)
	s := SessionView{Status: domain.SessionStatusPending, Rules: r}
	_, err := EvaluateBid(s, 5001, time.Now())
	if err != domain.ErrBidExceedsCap {
		t.Fatalf("want ErrBidExceedsCap got %v", err)
	}
}
