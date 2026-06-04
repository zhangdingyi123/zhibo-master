package domain

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to SessionStatus
		want     bool
	}{
		{SessionStatusPending, SessionStatusRunning, true},
		{SessionStatusPending, SessionStatusCancelled, true},
		{SessionStatusPending, SessionStatusSettled, false},
		{SessionStatusRunning, SessionStatusSettled, true},
		{SessionStatusRunning, SessionStatusCancelled, true},
		{SessionStatusRunning, SessionStatusPending, false},
		{SessionStatusSettled, SessionStatusRunning, false},
		{SessionStatusCancelled, SessionStatusFailed, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%s,%s)=%v want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestAuctionRulesValidate(t *testing.T) {
	cap := int64(10000)
	valid := AuctionRules{
		StartingPrice:      0,
		BidIncrement:       100,
		CapPrice:           &cap,
		DurationSec:        120,
		ExtendThresholdSec: 10,
		ExtendSec:          30,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid rules: %v", err)
	}

	invalid := valid
	invalid.ExtendSec = 5
	if err := invalid.Validate(); err != ErrInvalidExtendSec {
		t.Fatalf("expected ErrInvalidExtendSec, got %v", err)
	}
}

func TestMinNextBid(t *testing.T) {
	r := AuctionRules{StartingPrice: 0, BidIncrement: 1000}
	if got := r.MinNextBid(0, false); got != 0 {
		t.Fatalf("first bid min = %d want 0", got)
	}
	if got := r.MinNextBid(5000, true); got != 6000 {
		t.Fatalf("next bid min = %d want 6000", got)
	}
}
