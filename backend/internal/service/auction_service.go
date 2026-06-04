package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

type AuctionService struct {
	products *repository.ProductRepo
	sessions *repository.SessionRepo
	orders   *OrderService
	cache    RoomCache
	notify   RoomNotifier
}

func NewAuctionService(products *repository.ProductRepo, sessions *repository.SessionRepo, orders *OrderService) *AuctionService {
	return &AuctionService{products: products, sessions: sessions, orders: orders, notify: NoopRoomNotifier{}}
}

// SetRoomNotifier 注入实时推送
func (s *AuctionService) SetRoomNotifier(n RoomNotifier) {
	if n != nil {
		s.notify = n
	}
}

// SetRoomCache 取消/规则变更时失效缓存
func (s *AuctionService) SetRoomCache(c RoomCache) {
	s.cache = c
}

type PublishAuctionInput struct {
	Rules            domain.AuctionRules
	ScheduledStartAt *time.Time
}

type UpdateRulesInput struct {
	Rules            domain.AuctionRules
	ScheduledStartAt *time.Time
}

func (s *AuctionService) Publish(ctx context.Context, anchorID, productID uint64, in PublishAuctionInput) (*domain.AuctionSession, error) {
	if err := in.Rules.Validate(); err != nil {
		return nil, err
	}

	p, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	switch p.Status {
	case domain.ProductStatusOffShelf, domain.ProductStatusSold, domain.ProductStatusAuctioning:
		return nil, domain.ErrProductNotPublishable
	}

	active, err := s.sessions.HasActiveByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, domain.ErrActiveSessionExists
	}

	var scheduled sql.NullTime
	if in.ScheduledStartAt != nil {
		scheduled = sql.NullTime{Time: *in.ScheduledStartAt, Valid: true}
	}

	session, err := s.sessions.Create(ctx, repository.CreateSessionInput{
		ProductID:        productID,
		AnchorID:         anchorID,
		Rules:            in.Rules,
		ScheduledStartAt: scheduled,
	})
	if err != nil {
		return nil, err
	}

	if p.Status == domain.ProductStatusDraft {
		_ = s.products.UpdateStatus(ctx, productID, anchorID, domain.ProductStatusListed)
	}

	return session, nil
}

func (s *AuctionService) Get(ctx context.Context, anchorID, sessionID uint64) (*domain.AuctionSession, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	return session, nil
}

func (s *AuctionService) UpdateRules(ctx context.Context, anchorID, sessionID uint64, in UpdateRulesInput) (*domain.AuctionSession, error) {
	if err := in.Rules.Validate(); err != nil {
		return nil, err
	}
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	if !session.Status.CanModifyRules() {
		return nil, domain.ErrRulesNotEditable
	}
	if session.HasBids() {
		return nil, domain.ErrSessionHasBids
	}

	var scheduled sql.NullTime
	if in.ScheduledStartAt != nil {
		scheduled = sql.NullTime{Time: *in.ScheduledStartAt, Valid: true}
	}

	if err := s.sessions.UpdateRules(ctx, sessionID, anchorID, in.Rules, scheduled); err != nil {
		return nil, err
	}
	return s.sessions.GetByID(ctx, sessionID)
}

func (s *AuctionService) Cancel(ctx context.Context, anchorID, sessionID uint64, reason string) (*domain.AuctionSession, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, domain.ErrCancelReasonRequired
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	if !session.Status.CanCancelByAnchor() {
		return nil, domain.ErrSessionNotCancellable
	}

	if err := s.sessions.Cancel(ctx, sessionID, anchorID, reason); err != nil {
		return nil, err
	}

	// 商品状态回滚：竞拍中 → 已上架
	p, err := s.products.GetByID(ctx, session.ProductID)
	if err == nil && p.Status == domain.ProductStatusAuctioning {
		_ = s.products.UpdateStatus(ctx, session.ProductID, anchorID, domain.ProductStatusListed)
	}

	session, err = s.sessions.GetByID(ctx, sessionID)
	if err == nil {
		if s.cache != nil {
			_ = s.cache.Invalidate(ctx, session.RoomID, session.ID)
			_ = s.cache.RefreshFromSession(ctx, session)
		}
		if s.notify != nil {
			s.notify.OnCancelled(ctx, session, reason)
		}
	}
	return session, err
}

// CompleteSettlement 场次成交并自动生成订单（规则引擎阶段 3 调用）
func (s *AuctionService) CompleteSettlement(ctx context.Context, sessionID uint64, winnerID uint64, finalPrice int64) (*domain.Order, error) {
	if winnerID == 0 {
		return nil, domain.ErrSettlementNoWinner
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 已成交则返回已有订单
	if session.Status == domain.SessionStatusSettled {
		return s.orders.CreateFromSettledSession(ctx, session)
	}

	if err := session.TransitionTo(domain.SessionStatusSettled); err != nil {
		return nil, err
	}

	if err := s.sessions.MarkSettled(ctx, sessionID, winnerID, finalPrice); err != nil {
		return nil, err
	}

	_ = s.products.UpdateStatus(ctx, session.ProductID, session.AnchorID, domain.ProductStatusSold)

	session, err = s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.orders.CreateFromSettledSession(ctx, session)
}

func sessionToProgress(s *domain.AuctionSession, order *domain.Order) *AuctionProgress {
	if s == nil {
		return nil
	}
	p := &AuctionProgress{
		SessionID:        s.ID,
		RoomID:           s.RoomID,
		Status:           s.Status,
		CurrentPrice:     s.CurrentPrice,
		BidCount:         s.BidCount,
		ParticipantCount: s.ParticipantCount,
		ScheduledStartAt: s.ScheduledStartAt,
		StartedAt:        s.StartedAt,
		EndAt:            s.EndAt,
		SettledAt:        s.SettledAt,
		WinnerID:         s.WinnerID,
		CancelReason:     s.CancelReason,
		Order:            order,
	}
	return p
}

func pickSessionForView(active, latest *domain.AuctionSession) *domain.AuctionSession {
	if active != nil {
		return active
	}
	return latest
}
