package domain

import (
	"strings"
	"time"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPendingPay OrderStatus = "pending_pay"
	OrderStatusPaid       OrderStatus = "paid"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusClosed     OrderStatus = "closed"
	OrderStatusRefunded   OrderStatus = "refunded"
)

// OrderActor 关单 / 退款操作方
type OrderActor string

const (
	OrderActorBuyer  OrderActor = "buyer"
	OrderActorSeller OrderActor = "seller"
	OrderActorSystem OrderActor = "system"
)

// Order 成交订单
type Order struct {
	ID              uint64      `json:"id"`
	OrderNo         string      `json:"orderNo"`
	SessionID       uint64      `json:"sessionId"`
	ProductID       uint64      `json:"productId"`
	BuyerID         uint64      `json:"buyerId"`
	SellerID        uint64      `json:"sellerId"`
	Amount          int64       `json:"amount"`
	Status          OrderStatus `json:"status"`
	PayExpireAt     *time.Time  `json:"payExpireAt,omitempty"`
	PaidAt          *time.Time  `json:"paidAt,omitempty"`
	ReceiverName    string      `json:"receiverName,omitempty"`
	ReceiverPhone   string      `json:"receiverPhone,omitempty"`
	ReceiverAddress string      `json:"receiverAddress,omitempty"`
	TrackingNo      string      `json:"trackingNo,omitempty"`
	ShippedAt       *time.Time  `json:"shippedAt,omitempty"`
	CompletedAt     *time.Time  `json:"completedAt,omitempty"`
	CancelReason    string      `json:"cancelReason,omitempty"`
	CancelledBy     OrderActor  `json:"cancelledBy,omitempty"`
	CancelledAt     *time.Time  `json:"cancelledAt,omitempty"`
	RefundedAt      *time.Time  `json:"refundedAt,omitempty"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}

// HasShippingAddress 是否已填写收货信息
func (o *Order) HasShippingAddress() bool {
	if o == nil {
		return false
	}
	return strings.TrimSpace(o.ReceiverName) != "" &&
		strings.TrimSpace(o.ReceiverPhone) != "" &&
		strings.TrimSpace(o.ReceiverAddress) != ""
}
