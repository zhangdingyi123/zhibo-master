package engine

import (
	"testing"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// 3.6 幂等语义由 BidService + DB uk_session_request 保证；引擎层重复校验应一致
func TestEvaluateBid_SameAmountAfterCap(t *testing.T) {
	r := rulesZeroStart()
	r.CapPrice = cap(3000)
	s := SessionView{Status: domain.SessionStatusPending, Rules: r}
	out, err := EvaluateBid(s, 3000, mustTime("2026-05-26T12:00:00Z"))
	if err != nil || !out.Settled {
		t.Fatalf("cap bid: %+v err=%v", out, err)
	}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
