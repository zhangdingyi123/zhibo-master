package engine

import (
	"testing"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// 3.8 成交判定：达到封顶唯一胜者、状态 settled
func TestEvaluateBid_SettlementUniqueWinner(t *testing.T) {
	r := rulesZeroStart()
	r.CapPrice = cap(2000)
	now := time.Now()

	out, err := EvaluateBid(SessionView{Status: domain.SessionStatusPending, Rules: r}, 2000, now)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Settled || out.Status != domain.SessionStatusSettled {
		t.Fatalf("want settled at cap, got settled=%v status=%s", out.Settled, out.Status)
	}
	if out.AcceptedAmount != 2000 {
		t.Fatalf("final price %d", out.AcceptedAmount)
	}
}
