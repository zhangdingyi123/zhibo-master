package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/zhibo/backend/internal/domain"
)

type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) CreateTx(ctx context.Context, tx *sql.Tx, o *domain.Order) error {
	const q = `INSERT INTO orders (order_no, session_id, product_id, buyer_id, seller_id, amount, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, q,
		o.OrderNo, o.SessionID, o.ProductID, o.BuyerID, o.SellerID, o.Amount, string(o.Status),
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
	const q = `INSERT INTO orders (order_no, session_id, product_id, buyer_id, seller_id, amount, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q,
		o.OrderNo, o.SessionID, o.ProductID, o.BuyerID, o.SellerID, o.Amount, string(o.Status),
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

func (r *OrderRepo) GetByID(ctx context.Context, id uint64) (*domain.Order, error) {
	const q = `SELECT id, order_no, session_id, product_id, buyer_id, seller_id, amount, status, paid_at, created_at, updated_at
		FROM orders WHERE id = ?`
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

func (r *OrderRepo) GetBySessionID(ctx context.Context, sessionID uint64) (*domain.Order, error) {
	const q = `SELECT id, order_no, session_id, product_id, buyer_id, seller_id, amount, status, paid_at, created_at, updated_at
		FROM orders WHERE session_id = ?`
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

	listQ := `SELECT id, order_no, session_id, product_id, buyer_id, seller_id, amount, status, paid_at, created_at, updated_at
		FROM orders WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
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

	q := `SELECT o.id, o.order_no, o.session_id, o.product_id, o.buyer_id, o.seller_id, o.amount, o.status, o.paid_at, o.created_at, o.updated_at
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

func scanOrderRow(row *sql.Row) (*domain.Order, error) {
	var o domain.Order
	var status string
	var paidAt sql.NullTime
	err := row.Scan(
		&o.ID, &o.OrderNo, &o.SessionID, &o.ProductID, &o.BuyerID, &o.SellerID,
		&o.Amount, &status, &paidAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	o.Status = domain.OrderStatus(status)
	if paidAt.Valid {
		t := paidAt.Time
		o.PaidAt = &t
	}
	return &o, nil
}

func scanOrderFromRows(rows *sql.Rows) (*domain.Order, error) {
	var o domain.Order
	var status string
	var paidAt sql.NullTime
	err := rows.Scan(
		&o.ID, &o.OrderNo, &o.SessionID, &o.ProductID, &o.BuyerID, &o.SellerID,
		&o.Amount, &status, &paidAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	o.Status = domain.OrderStatus(status)
	if paidAt.Valid {
		t := paidAt.Time
		o.PaidAt = &t
	}
	return &o, nil
}
