package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

type OrderService struct {
	orders     *repository.OrderRepo
	products   *repository.ProductRepo
	payTimeout time.Duration
}

func NewOrderService(orders *repository.OrderRepo, products *repository.ProductRepo, payTimeout time.Duration) *OrderService {
	if payTimeout <= 0 {
		payTimeout = 30 * time.Minute
	}
	return &OrderService{orders: orders, products: products, payTimeout: payTimeout}
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

// BuyerOrderItem 买家订单列表项（含商品摘要，避免 N+1）
type BuyerOrderItem struct {
	Order   domain.Order `json:"order"`
	Product ProductBrief `json:"product"`
}

type BuyerListOrdersResult struct {
	Items    []BuyerOrderItem `json:"items"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
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

func (s *OrderService) GetForBuyer(ctx context.Context, buyerID, orderID uint64) (*BuyerOrderItem, error) {
	if err := s.closeExpiredPending(ctx); err != nil {
		return nil, err
	}
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o.BuyerID != buyerID {
		return nil, domain.ErrForbidden
	}
	return s.toBuyerOrderItem(ctx, o)
}

func (s *OrderService) GetBySessionForBuyer(ctx context.Context, buyerID, sessionID uint64) (*BuyerOrderItem, error) {
	if err := s.closeExpiredPending(ctx); err != nil {
		return nil, err
	}
	o, err := s.orders.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if o.BuyerID != buyerID {
		return nil, domain.ErrForbidden
	}
	return s.toBuyerOrderItem(ctx, o)
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
	page, pageSize := normalizePage(in.Page, in.PageSize)
	return &ListOrdersResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *OrderService) ListForBuyer(ctx context.Context, buyerID uint64, in ListOrdersInput) (*BuyerListOrdersResult, error) {
	if err := s.closeExpiredPending(ctx); err != nil {
		return nil, err
	}
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
	enriched, err := s.attachProducts(ctx, items)
	if err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(in.Page, in.PageSize)
	return &BuyerListOrdersResult{
		Items:    enriched,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// MockPay 模拟支付：仅 pending_pay 且未超时可支付
func (s *OrderService) MockPay(ctx context.Context, buyerID, orderID uint64) (*BuyerOrderItem, error) {
	if err := s.closeExpiredPending(ctx); err != nil {
		return nil, err
	}
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o.BuyerID != buyerID {
		return nil, domain.ErrForbidden
	}
	if o.Status != domain.OrderStatusPendingPay {
		return nil, domain.ErrInvalidStateTransition
	}
	if s.isPayExpired(o, time.Now()) {
		return nil, domain.ErrOrderPayExpired
	}
	if err := s.orders.MarkPaid(ctx, orderID); err != nil {
		return nil, err
	}
	o, err = s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return s.toBuyerOrderItem(ctx, o)
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

	expireAt := payExpireAt(time.Now(), s.payTimeout)
	o := &domain.Order{
		OrderNo:     generateOrderNo(session.ID),
		SessionID:   session.ID,
		ProductID:   session.ProductID,
		BuyerID:     *session.WinnerID,
		SellerID:    session.AnchorID,
		Amount:      session.CurrentPrice,
		Status:      domain.OrderStatusPendingPay,
		PayExpireAt: &expireAt,
	}
	if err := s.orders.Create(ctx, o); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, o.ID)
}

// RunPayExpiryWorker 定时关闭超时未支付订单
func (s *OrderService) RunPayExpiryWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := s.orders.CloseExpiredPending(ctx, time.Now())
			if err != nil {
				log.Printf("order expiry: %v", err)
			} else if n > 0 {
				log.Printf("order expiry: closed %d pending orders", n)
			}
		}
	}
}

func (s *OrderService) closeExpiredPending(ctx context.Context) error {
	_, err := s.orders.CloseExpiredPending(ctx, time.Now())
	return err
}

func (s *OrderService) isPayExpired(o *domain.Order, now time.Time) bool {
	return o.PayExpireAt != nil && now.After(*o.PayExpireAt)
}

func payExpireAt(from time.Time, timeout time.Duration) time.Time {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return from.Add(timeout)
}

func (s *OrderService) attachProducts(ctx context.Context, orders []domain.Order) ([]BuyerOrderItem, error) {
	if len(orders) == 0 {
		return []BuyerOrderItem{}, nil
	}
	ids := make([]uint64, 0, len(orders))
	seen := make(map[uint64]struct{}, len(orders))
	for _, o := range orders {
		if _, ok := seen[o.ProductID]; ok {
			continue
		}
		seen[o.ProductID] = struct{}{}
		ids = append(ids, o.ProductID)
	}
	productMap, err := s.products.MapBriefByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]BuyerOrderItem, 0, len(orders))
	for _, o := range orders {
		item := BuyerOrderItem{Order: o}
		if p, ok := productMap[o.ProductID]; ok {
			item.Product = ProductBrief{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				CoverURL:    p.CoverURL,
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *OrderService) toBuyerOrderItem(ctx context.Context, o *domain.Order) (*BuyerOrderItem, error) {
	items, err := s.attachProducts(ctx, []domain.Order{*o})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, domain.ErrNotFound
	}
	return &items[0], nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return page, pageSize
}

func generateOrderNo(sessionID uint64) string {
	return fmt.Sprintf("ZB%s%06d", time.Now().Format("20060102"), sessionID%1000000)
}
