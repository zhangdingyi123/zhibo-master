package engine

import (
	"time"

	"github.com/zhibo/backend/internal/domain"
)

// SessionView 规则引擎输入（场次快照，不含用户身份）
type SessionView struct {
	Status       domain.SessionStatus
	CurrentPrice int64
	BidCount     uint32
	Rules        domain.AuctionRules
	EndAt        *time.Time
}

// BidOutcome 单次出价规则计算结果（持久化由 BidService 执行）
type BidOutcome struct {
	AcceptedAmount int64
	Settled        bool
	Status         domain.SessionStatus
	StartedAt      *time.Time
	EndAt          *time.Time
}

// EvaluateBid 校验并计算出价效果：0 元起拍、加价幅度、封顶成交、延时窗口。
func EvaluateBid(s SessionView, amount int64, now time.Time) (BidOutcome, error) {
	if s.Status.IsTerminal() {
		return BidOutcome{}, domain.ErrSessionNotBiddable
	}
	if s.Status != domain.SessionStatusPending && s.Status != domain.SessionStatusRunning {
		return BidOutcome{}, domain.ErrSessionNotBiddable
	}
	if s.Status == domain.SessionStatusRunning && s.EndAt != nil && !now.Before(*s.EndAt) {
		return BidOutcome{}, domain.ErrAuctionEnded
	}

	if s.Rules.CapPrice != nil && amount > *s.Rules.CapPrice {
		return BidOutcome{}, domain.ErrBidExceedsCap
	}

	hasBids := s.BidCount > 0
	minBid := s.Rules.MinNextBid(s.CurrentPrice, hasBids)
	if amount < minBid {
		return BidOutcome{}, domain.ErrBidTooLow
	}

	accepted := amount
	settled := s.Rules.IsCapReached(accepted)
	if settled && s.Rules.CapPrice != nil {
		accepted = *s.Rules.CapPrice
	}

	out := BidOutcome{
		AcceptedAmount: accepted,
		Settled:        settled,
	}

	if s.Status == domain.SessionStatusPending {
		out.Status = domain.SessionStatusRunning
		started := now
		out.StartedAt = &started
		end := now.Add(time.Duration(s.Rules.DurationSec) * time.Second)
		out.EndAt = &end
	} else {
		out.Status = domain.SessionStatusRunning
		out.EndAt = s.EndAt
		if !settled && out.EndAt != nil {
			out.EndAt = MaybeExtendEnd(*out.EndAt, now, s.Rules)
		}
	}

	if settled {
		out.Status = domain.SessionStatusSettled
	}
	return out, nil
}

// MaybeExtendEnd 结束前 extendThresholdSec 内有出价则延长 extendSec（封顶成交不调用）
func MaybeExtendEnd(endAt, now time.Time, rules domain.AuctionRules) *time.Time {
	remaining := endAt.Sub(now)
	threshold := time.Duration(rules.ExtendThresholdSec) * time.Second
	if remaining > 0 && remaining <= threshold {
		newEnd := endAt.Add(time.Duration(rules.ExtendSec) * time.Second)
		return &newEnd
	}
	return &endAt
}

// SessionViewFrom 从领域场次构造引擎输入
func SessionViewFrom(s *domain.AuctionSession) SessionView {
	return SessionView{
		Status:       s.Status,
		CurrentPrice: s.CurrentPrice,
		BidCount:     s.BidCount,
		Rules:        s.Rules,
		EndAt:        s.EndAt,
	}
}
