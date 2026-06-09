package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) CreateTx(ctx context.Context, tx *sql.Tx, o *domain.Order) error {
	const q = `INSERT INTO orders (order_no, session_id, product_id, buyer_id, seller_id, amount, status, pay_expire_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, q,
		o.OrderNo, o.SessionID, o.ProductID, o.BuyerID, o.SellerID, o.Amount, string(o.Status), nullTime(o.PayExpireAt),
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	o.ID = uint64(id)
	return nil
}

func (r *OrderRepo) Create(ctx context.Context, o *domain.Order) error {
	const q = `INSERT INTO orders (order_no, session_id, product_id, buyer_id, seller_id, amount, status, pay_expire_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q,
		o.OrderNo, o.SessionID, o.ProductID, o.BuyerID, o.SellerID, o.Amount, string(o.Status), nullTime(o.PayExpireAt),
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("order last insert id: %w", err)
	}
	o.ID = uint64(id)
	return nil
}

const orderSelectCols = `id, order_no, session_id, product_id, buyer_id, seller_id, amount, status,
		pay_expire_at, paid_at, receiver_name, receiver_phone, receiver_address, tracking_no,
		shipped_at, completed_at, cancel_reason, cancelled_by, cancelled_at, refunded_at,
		created_at, updated_at`

func (r *OrderRepo) GetByID(ctx context.Context, id uint64) (*domain.Order, error) {
	const q = `SELECT ` + orderSelectCols + ` FROM orders WHERE id = ?`
	row := r.db.QueryRowContext(ctx, q, id)
	o, err := scanOrderRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

// CountPaidBySessionIDs 统计指定场次中已支付及以上状态的订单数
func (r *OrderRepo) CountPaidBySessionIDs(ctx context.Context, sessionIDs []uint64) (int, error) {
	if len(sessionIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(sessionIDs))
	args := make([]any, len(sessionIDs))
	for i, id := range sessionIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	q := `SELECT COUNT(*) FROM orders WHERE session_id IN (` + strings.Join(placeholders, ",") +
		`) AND status IN ('paid', 'shipped', 'completed')`
	var count int
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count paid orders: %w", err)
	}
	return count, nil
}

func (r *OrderRepo) GetBySessionID(ctx context.Context, sessionID uint64) (*domain.Order, error) {
	const q = `SELECT ` + orderSelectCols + ` FROM orders WHERE session_id = ?`
	row := r.db.QueryRowContext(ctx, q, sessionID)
	o, err := scanOrderRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order by session: %w", err)
	}
	return o, nil
}

type OrderFilter struct {
	SellerID *uint64
	BuyerID  *uint64
	Status   *domain.OrderStatus
	Page     int
	PageSize int
}

func (r *OrderRepo) List(ctx context.Context, f OrderFilter) ([]domain.Order, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	var conds []string
	var args []any
	if f.SellerID != nil {
		conds = append(conds, "seller_id = ?")
		args = append(args, *f.SellerID)
	}
	if f.BuyerID != nil {
		conds = append(conds, "buyer_id = ?")
		args = append(args, *f.BuyerID)
	}
	if len(conds) == 0 {
		return nil, 0, fmt.Errorf("order list filter: seller_id or buyer_id required")
	}
	if f.Status != nil {
		conds = append(conds, "status = ?")
		args = append(args, string(*f.Status))
	}
	where := strings.Join(conds, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	listQ := `SELECT ` + orderSelectCols + ` FROM orders WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), f.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var items []domain.Order
	for rows.Next() {
		o, err := scanOrderFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *o)
	}
	if items == nil {
		items = []domain.Order{}
	}
	return items, total, rows.Err()
}

func (r *OrderRepo) MapLatestByProductIDs(ctx context.Context, productIDs []uint64) (map[uint64]*domain.Order, error) {
	if len(productIDs) == 0 {
		return map[uint64]*domain.Order{}, nil
	}
	placeholders := placeholders(len(productIDs))
	args := uint64sToAny(productIDs)

	q := `SELECT o.id, o.order_no, o.session_id, o.product_id, o.buyer_id, o.seller_id, o.amount, o.status,
		o.pay_expire_at, o.paid_at, o.receiver_name, o.receiver_phone, o.receiver_address, o.tracking_no,
		o.shipped_at, o.completed_at, o.cancel_reason, o.cancelled_by, o.cancelled_at, o.refunded_at,
		o.created_at, o.updated_at
		FROM orders o
		INNER JOIN (
			SELECT product_id, MAX(id) AS max_id FROM orders WHERE product_id IN (` + placeholders + `) GROUP BY product_id
		) t ON o.product_id = t.product_id AND o.id = t.max_id`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("map orders by products: %w", err)
	}
	defer rows.Close()

	out := make(map[uint64]*domain.Order)
	for rows.Next() {
		o, err := scanOrderFromRows(rows)
		if err != nil {
			return nil, err
		}
		out[o.ProductID] = o
	}
	return out, rows.Err()
}

// CloseExpiredPending 将已过期的待支付订单关单
func (r *OrderRepo) CloseExpiredPending(ctx context.Context, now time.Time) (int64, error) {
	const q = `UPDATE orders SET status = ?, updated_at = NOW(3)
		WHERE status = ? AND pay_expire_at IS NOT NULL AND pay_expire_at < ?`
	res, err := r.db.ExecContext(ctx, q, string(domain.OrderStatusClosed), string(domain.OrderStatusPendingPay), now)
	if err != nil {
		return 0, fmt.Errorf("close expired orders: %w", err)
	}
	return res.RowsAffected()
}

func (r *OrderRepo) MarkPaid(ctx context.Context, orderID uint64) error {
	const q = `UPDATE orders SET status = ?, paid_at = NOW(3), updated_at = NOW(3) WHERE id = ? AND status = ?`
	res, err := r.db.ExecContext(ctx, q, string(domain.OrderStatusPaid), orderID, string(domain.OrderStatusPendingPay))
	if err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OrderRepo) UpdateShipping(ctx context.Context, orderID uint64, name, phone, address string) error {
	const q = `UPDATE orders SET receiver_name = ?, receiver_phone = ?, receiver_address = ?, updated_at = NOW(3)
		WHERE id = ? AND status = ?`
	res, err := r.db.ExecContext(ctx, q, name, phone, address, orderID, string(domain.OrderStatusPaid))
	if err != nil {
		return fmt.Errorf("update shipping: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrInvalidStateTransition
	}
	return nil
}

func (r *OrderRepo) MarkShipped(ctx context.Context, orderID uint64, trackingNo string) error {
	const q = `UPDATE orders SET status = ?, tracking_no = ?, shipped_at = NOW(3), updated_at = NOW(3)
		WHERE id = ? AND status = ?
		AND receiver_name IS NOT NULL AND receiver_name != ''
		AND receiver_phone IS NOT NULL AND receiver_phone != ''
		AND receiver_address IS NOT NULL AND receiver_address != ''`
	res, err := r.db.ExecContext(ctx, q, string(domain.OrderStatusShipped), nullString(trackingNo), orderID, string(domain.OrderStatusPaid))
	if err != nil {
		return fmt.Errorf("mark order shipped: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		o, getErr := r.GetByID(ctx, orderID)
		if getErr != nil {
			return getErr
		}
		if o.Status != domain.OrderStatusPaid {
			return domain.ErrInvalidStateTransition
		}
		return domain.ErrOrderAddressMissing
	}
	return nil
}

func (r *OrderRepo) MarkCancelled(ctx context.Context, orderID uint64, reason string, by domain.OrderActor) error {
	const q = `UPDATE orders SET status = ?, cancel_reason = ?, cancelled_by = ?, cancelled_at = NOW(3), updated_at = NOW(3)
		WHERE id = ? AND status = ?`
	res, err := r.db.ExecContext(ctx, q,
		string(domain.OrderStatusCancelled), reason, string(by), orderID, string(domain.OrderStatusPendingPay),
	)
	if err != nil {
		return fmt.Errorf("mark order cancelled: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrOrderNotCancellable
	}
	return nil
}

func (r *OrderRepo) MarkRefunded(ctx context.Context, orderID uint64, reason string, by domain.OrderActor, from domain.OrderStatus) error {
	const q = `UPDATE orders SET status = ?, cancel_reason = ?, cancelled_by = ?, refunded_at = NOW(3), updated_at = NOW(3)
		WHERE id = ? AND status = ?`
	res, err := r.db.ExecContext(ctx, q,
		string(domain.OrderStatusRefunded), reason, string(by), orderID, string(from),
	)
	if err != nil {
		return fmt.Errorf("mark order refunded: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrOrderNotRefundable
	}
	return nil
}

func (r *OrderRepo) MarkCompleted(ctx context.Context, orderID uint64) error {
	const q = `UPDATE orders SET status = ?, completed_at = NOW(3), updated_at = NOW(3) WHERE id = ? AND status = ?`
	res, err := r.db.ExecContext(ctx, q, string(domain.OrderStatusCompleted), orderID, string(domain.OrderStatusShipped))
	if err != nil {
		return fmt.Errorf("mark order completed: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrInvalidStateTransition
	}
	return nil
}

func scanOrderRow(row *sql.Row) (*domain.Order, error) {
	return scanOrder(row)
}

func scanOrderFromRows(rows *sql.Rows) (*domain.Order, error) {
	return scanOrder(rows)
}

type scanOrderRowScanner interface {
	Scan(dest ...any) error
}

func scanOrder(s scanOrderRowScanner) (*domain.Order, error) {
	var o domain.Order
	var status string
	var payExpireAt, paidAt, shippedAt, completedAt, cancelledAt, refundedAt sql.NullTime
	var receiverName, receiverPhone, receiverAddress, trackingNo, cancelReason, cancelledBy sql.NullString
	err := s.Scan(
		&o.ID, &o.OrderNo, &o.SessionID, &o.ProductID, &o.BuyerID, &o.SellerID,
		&o.Amount, &status, &payExpireAt, &paidAt,
		&receiverName, &receiverPhone, &receiverAddress, &trackingNo,
		&shippedAt, &completedAt, &cancelReason, &cancelledBy, &cancelledAt, &refundedAt,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	o.Status = domain.OrderStatus(status)
	if payExpireAt.Valid {
		t := payExpireAt.Time
		o.PayExpireAt = &t
	}
	if paidAt.Valid {
		t := paidAt.Time
		o.PaidAt = &t
	}
	if receiverName.Valid {
		o.ReceiverName = receiverName.String
	}
	if receiverPhone.Valid {
		o.ReceiverPhone = receiverPhone.String
	}
	if receiverAddress.Valid {
		o.ReceiverAddress = receiverAddress.String
	}
	if trackingNo.Valid {
		o.TrackingNo = trackingNo.String
	}
	if shippedAt.Valid {
		t := shippedAt.Time
		o.ShippedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		o.CompletedAt = &t
	}
	if cancelReason.Valid {
		o.CancelReason = cancelReason.String
	}
	if cancelledBy.Valid {
		o.CancelledBy = domain.OrderActor(cancelledBy.String)
	}
	if cancelledAt.Valid {
		t := cancelledAt.Time
		o.CancelledAt = &t
	}
	if refundedAt.Valid {
		t := refundedAt.Time
		o.RefundedAt = &t
	}
	return &o, nil
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
