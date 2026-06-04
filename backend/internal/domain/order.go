package domain

import "time"

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPendingPay OrderStatus = "pending_pay"
	OrderStatusPaid       OrderStatus = "paid"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusClosed     OrderStatus = "closed"
)

// Order 成交订单
type Order struct {
	ID        uint64      `json:"id"`
	OrderNo   string      `json:"orderNo"`
	SessionID uint64      `json:"sessionId"`
	ProductID uint64      `json:"productId"`
	BuyerID   uint64      `json:"buyerId"`
	SellerID  uint64      `json:"sellerId"`
	Amount    int64       `json:"amount"`
	Status    OrderStatus `json:"status"`
	PaidAt    *time.Time  `json:"paidAt,omitempty"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}
