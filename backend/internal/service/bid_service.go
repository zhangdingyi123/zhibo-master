package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/engine"
	"github.com/zhibo/backend/internal/repository"
)

type BidService struct {
	db       *sql.DB
	sessions *repository.SessionRepo
	bids     *repository.BidRepo
	products *repository.ProductRepo
	orders   *repository.OrderRepo
	locker   SessionLocker
	cache    RoomCache
	notify   RoomNotifier
}

func NewBidService(
	db *sql.DB,
	sessions *repository.SessionRepo,
	bids *repository.BidRepo,
	products *repository.ProductRepo,
	orders *repository.OrderRepo,
	locker SessionLocker,
) *BidService {
	if locker == nil {
		locker = NoopLocker{}
	}
	return &BidService{db: db, sessions: sessions, bids: bids, products: products, orders: orders, locker: locker, notify: NoopRoomNotifier{}}
}

// SetRoomNotifier 注入实时推送（WebSocket）
func (s *BidService) SetRoomNotifier(n RoomNotifier) {
	if n != nil {
		s.notify = n
	}
}

// SetRoomCache 出价成功后写 Redis（5.2 写穿）
func (s *BidService) SetRoomCache(c RoomCache) {
	s.cache = c
}

type PlaceBidInput struct {
	Amount    int64
	RequestID string
}

type PlaceBidResult struct {
	Bid      domain.Bid            `json:"bid"`
	Session  domain.AuctionSession `json:"session"`
	Snapshot *SessionSnapshot      `json:"snapshot"`
	Settled  bool                  `json:"settled"`
	Order    *domain.Order         `json:"order,omitempty"`
}

func (s *BidService) PlaceBid(ctx context.Context, userID, sessionID uint64, in PlaceBidInput) (*PlaceBidResult, error) {
	requestID := strings.TrimSpace(in.RequestID)
	if requestID == "" {
		return nil, domain.ErrRequestIDRequired
	}

	// 幂等：已处理则直接返回
	if existing, err := s.bids.GetByRequestID(ctx, sessionID, requestID); err == nil {
		session, err := s.sessions.GetByID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		return s.buildBidResult(ctx, existing, session, nil)
	}

	var result *PlaceBidResult
	err := s.locker.WithSessionLock(ctx, sessionID, func(lockCtx context.Context) error {
		var err error
		result, err = s.placeBidLocked(lockCtx, userID, sessionID, in.Amount, requestID)
		return err
	})
	return result, err
}

func (s *BidService) placeBidLocked(ctx context.Context, userID, sessionID uint64, amount int64, requestID string) (*PlaceBidResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if existing, err := s.bids.GetByRequestIDTx(ctx, tx, sessionID, requestID); err == nil {
		_ = tx.Commit()
		session, _ := s.sessions.GetByID(ctx, sessionID)
		return s.buildBidResult(ctx, existing, session, nil)
	}

	session, err := s.sessions.GetByIDForUpdate(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	prevEndAt := session.EndAt

	now := time.Now()
	outcome, err := engine.EvaluateBid(engine.SessionViewFrom(session), amount, now)
	if err != nil {
		return nil, err
	}

	if session.Status == domain.SessionStatusPending && outcome.StartedAt != nil {
		if err := s.products.UpdateStatusTx(ctx, tx, session.ProductID, domain.ProductStatusAuctioning); err != nil {
			return nil, err
		}
	}

	amount = outcome.AcceptedAmount
	settled := outcome.Settled

	seq, err := s.bids.NextSeq(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}

	hadBid, err := s.bids.UserHasBid(ctx, tx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	bid := &domain.Bid{
		SessionID: sessionID,
		UserID:    userID,
		Amount:    amount,
		RequestID: requestID,
		Seq:       seq,
	}
	if err := s.bids.Create(ctx, tx, bid); err != nil {
		return nil, err
	}

	session.CurrentPrice = amount
	session.BidCount++
	if !hadBid {
		session.ParticipantCount++
	}
	session.Status = outcome.Status
	if outcome.StartedAt != nil {
		session.StartedAt = outcome.StartedAt
	}
	if outcome.EndAt != nil {
		session.EndAt = outcome.EndAt
	}

	var order *domain.Order
	if settled {
		if err := s.sessions.MarkSettledTx(ctx, tx, sessionID, userID, amount); err != nil {
			return nil, err
		}
		winner := userID
		session.WinnerID = &winner
		t := now
		session.SettledAt = &t
		if err := s.products.UpdateStatusTx(ctx, tx, session.ProductID, domain.ProductStatusSold); err != nil {
			return nil, err
		}
		order = &domain.Order{
			OrderNo:   generateOrderNo(sessionID),
			SessionID: sessionID,
			ProductID: session.ProductID,
			BuyerID:   userID,
			SellerID:  session.AnchorID,
			Amount:    amount,
			Status:    domain.OrderStatusPendingPay,
		}
		if err := s.orders.CreateTx(ctx, tx, order); err != nil {
			return nil, err
		}
	} else {
		if err := s.sessions.ApplyBid(ctx, tx, repository.ApplyBidParams{
			SessionID:        sessionID,
			Version:          session.Version,
			CurrentPrice:     session.CurrentPrice,
			BidCount:         session.BidCount,
			ParticipantCount: session.ParticipantCount,
			Status:           session.Status,
			StartedAt:        outcome.StartedAt,
			EndAt:            outcome.EndAt,
		}); err != nil {
			return nil, err
		}
		session.Version++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit bid: %w", err)
	}

	session, err = s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	bid, err = s.bids.GetByRequestID(ctx, sessionID, requestID)
	if err != nil {
		return nil, err
	}
	if settled && order != nil {
		order, _ = s.orders.GetBySessionID(ctx, sessionID)
	}
	result, err := s.buildBidResult(ctx, bid, session, order)
	if err == nil {
		if s.cache != nil {
			_ = s.cache.OnBid(ctx, session, userID, bid.Amount, bid.Seq, !hadBid)
		}
		if s.notify != nil {
			s.notify.OnBid(ctx, result, prevEndAt)
		}
	}
	return result, err
}

func (s *BidService) buildBidResult(ctx context.Context, bid *domain.Bid, session *domain.AuctionSession, order *domain.Order) (*PlaceBidResult, error) {
	settled := session.Status == domain.SessionStatusSettled
	return &PlaceBidResult{
		Bid:      *bid,
		Session:  *session,
		Snapshot: BuildSnapshot(session, time.Now()),
		Settled:  settled,
		Order:    order,
	}, nil
}
