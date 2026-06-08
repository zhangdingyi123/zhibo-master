package service

import (
	"context"
	"log"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

const (
	settlementBatchSize  = 50
	settlementScanInterval = 200 * time.Millisecond // 与 WS countdown.tick 同频，归零后最多 ~200ms 落锤
)

// RunSettlementWorker 扫描倒计时已结束的 running 场次并落锤成交
func (s *AuctionService) RunSettlementWorker(ctx context.Context) {
	ticker := time.NewTicker(settlementScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := s.settleExpiredRunning(ctx, time.Now())
			if err != nil {
				log.Printf("settlement worker: %v", err)
			} else if n > 0 {
				log.Printf("settlement worker: settled %d sessions", n)
			}
		}
	}
}

func (s *AuctionService) settleExpiredRunning(ctx context.Context, now time.Time) (int, error) {
	ids, err := s.sessions.ListExpiredRunning(ctx, now, settlementBatchSize)
	if err != nil {
		return 0, err
	}
	settled := 0
	for _, id := range ids {
		if err := s.settleExpiredSession(ctx, id, now); err != nil {
			log.Printf("settlement worker: session %d: %v", id, err)
			continue
		}
		settled++
	}
	return settled, nil
}

func (s *AuctionService) settleExpiredSession(ctx context.Context, sessionID uint64, now time.Time) error {
	return s.locker.WithSessionLock(ctx, sessionID, func(lockCtx context.Context) error {
		session, err := s.sessions.GetByID(lockCtx, sessionID)
		if err != nil {
			return err
		}
		if session.Status != domain.SessionStatusRunning {
			return nil
		}
		if session.EndAt == nil || now.Before(*session.EndAt) {
			return nil
		}

		if session.BidCount == 0 {
			reason := "倒计时结束无有效出价"
			if err := s.sessions.MarkFailed(lockCtx, sessionID, reason); err != nil {
				return err
			}
			session, err = s.sessions.GetByID(lockCtx, sessionID)
			if err == nil && s.cache != nil {
				writeCacheWithRetry(lockCtx, "refresh_failed", session.ID, session.RoomID, func() error {
					return s.cache.RefreshFromSession(lockCtx, session)
				})
			}
			return nil
		}

		winnerID, err := s.resolveWinner(lockCtx, sessionID)
		if err != nil {
			return err
		}
		if winnerID == 0 {
			reason := "倒计时结束无法确定胜者"
			return s.sessions.MarkFailed(lockCtx, sessionID, reason)
		}

		_, err = s.completeSettlementUnlocked(lockCtx, session, winnerID, session.CurrentPrice)
		return err
	})
}

func (s *AuctionService) resolveWinner(ctx context.Context, sessionID uint64) (uint64, error) {
	winnerID, err := s.bids.GetWinningUserID(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	if winnerID != nil {
		return *winnerID, nil
	}
	top, err := s.bids.ListTopBySession(ctx, sessionID, 1)
	if err != nil {
		return 0, err
	}
	if len(top) == 0 {
		return 0, nil
	}
	return top[0].UserID, nil
}

// completeSettlementUnlocked 在已持有场次锁且状态仍为 running 时落锤（worker 内部）
func (s *AuctionService) completeSettlementUnlocked(
	ctx context.Context,
	session *domain.AuctionSession,
	winnerID uint64,
	finalPrice int64,
) (*domain.Order, error) {
	if session.Status == domain.SessionStatusSettled {
		return s.orders.CreateFromSettledSession(ctx, session)
	}
	if err := session.TransitionTo(domain.SessionStatusSettled); err != nil {
		return nil, err
	}
	if err := s.sessions.MarkSettled(ctx, session.ID, winnerID, finalPrice); err != nil {
		return nil, err
	}
	_ = s.products.UpdateStatus(ctx, session.ProductID, session.AnchorID, domain.ProductStatusSold)

	session, err := s.sessions.GetByID(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	order, err := s.orders.CreateFromSettledSession(ctx, session)
	if err != nil {
		return nil, err
	}
	s.afterSettled(ctx, session, order)
	return order, nil
}
