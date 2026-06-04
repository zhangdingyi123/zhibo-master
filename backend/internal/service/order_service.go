package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

type OrderService struct {
	orders *repository.OrderRepo
}

func NewOrderService(orders *repository.OrderRepo) *OrderService {
	return &OrderService{orders: orders}
}

type ListOrdersInput struct {
	Status   *domain.OrderStatus
	Page     int
	PageSize int
}

type ListOrdersResult struct {
	Items    []domain.Order `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

func (s *OrderService) Get(ctx context.Context, sellerID, orderID uint64) (*domain.Order, error) {
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o.SellerID != sellerID {
		return nil, domain.ErrForbidden
	}
	return o, nil
}

func (s *OrderService) GetForBuyer(ctx context.Context, buyerID, orderID uint64) (*domain.Order, error) {
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o.BuyerID != buyerID {
		return nil, domain.ErrForbidden
	}
	return o, nil
}

func (s *OrderService) GetBySessionForBuyer(ctx context.Context, buyerID, sessionID uint64) (*domain.Order, error) {
	o, err := s.orders.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if o.BuyerID != buyerID {
		return nil, domain.ErrForbidden
	}
	return o, nil
}

func (s *OrderService) List(ctx context.Context, sellerID uint64, in ListOrdersInput) (*ListOrdersResult, error) {
	sid := sellerID
	items, total, err := s.orders.List(ctx, repository.OrderFilter{
		SellerID: &sid,
		Status:   in.Status,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
	if err != nil {
		return nil, err
	}
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	return &ListOrdersResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *OrderService) ListForBuyer(ctx context.Context, buyerID uint64, in ListOrdersInput) (*ListOrdersResult, error) {
	bid := buyerID
	items, total, err := s.orders.List(ctx, repository.OrderFilter{
		BuyerID:  &bid,
		Status:   in.Status,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
	if err != nil {
		return nil, err
	}
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	return &ListOrdersResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// MockPay 模拟支付：仅 pending_pay 可支付
func (s *OrderService) MockPay(ctx context.Context, buyerID, orderID uint64) (*domain.Order, error) {
	o, err := s.GetForBuyer(ctx, buyerID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.OrderStatusPendingPay {
		return nil, domain.ErrInvalidStateTransition
	}
	if err := s.orders.MarkPaid(ctx, orderID); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, orderID)
}

// CreateFromSettledSession 成交后创建订单（幂等，供规则引擎调用）
func (s *OrderService) CreateFromSettledSession(ctx context.Context, session *domain.AuctionSession) (*domain.Order, error) {
	if session.WinnerID == nil {
		return nil, domain.ErrSettlementNoWinner
	}
	existing, err := s.orders.GetBySessionID(ctx, session.ID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	o := &domain.Order{
		OrderNo:   generateOrderNo(session.ID),
		SessionID: session.ID,
		ProductID: session.ProductID,
		BuyerID:   *session.WinnerID,
		SellerID:  session.AnchorID,
		Amount:    session.CurrentPrice,
		Status:    domain.OrderStatusPendingPay,
	}
	if err := s.orders.Create(ctx, o); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, o.ID)
}

func generateOrderNo(sessionID uint64) string {
	return fmt.Sprintf("ZB%s%06d", time.Now().Format("20060102"), sessionID%1000000)
}
