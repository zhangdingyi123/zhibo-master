package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/zhibo/backend/internal/domain"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

type ProductFilter struct {
	AnchorID uint64
	Status   *domain.ProductStatus
	Page     int
	PageSize int
}

func (r *ProductRepo) Create(ctx context.Context, p *domain.Product) error {
	imagesJSON, err := json.Marshal(p.Images)
	if err != nil {
		return fmt.Errorf("marshal images: %w", err)
	}
	const q = `INSERT INTO products (anchor_id, name, description, cover_url, images, status)
		VALUES (?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q,
		p.AnchorID, p.Name, p.Description, p.CoverURL, imagesJSON, string(p.Status),
	)
	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	p.ID = uint64(id)
	return nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id uint64) (*domain.Product, error) {
	const q = `SELECT id, anchor_id, name, description, cover_url, images, status, created_at, updated_at
		FROM products WHERE id = ?`
	return r.scanOne(ctx, q, id)
}

func (r *ProductRepo) Update(ctx context.Context, p *domain.Product) error {
	imagesJSON, err := json.Marshal(p.Images)
	if err != nil {
		return fmt.Errorf("marshal images: %w", err)
	}
	const q = `UPDATE products SET name = ?, description = ?, cover_url = ?, images = ?, status = ?
		WHERE id = ? AND anchor_id = ?`
	res, err := r.db.ExecContext(ctx, q,
		p.Name, p.Description, p.CoverURL, imagesJSON, string(p.Status), p.ID, p.AnchorID,
	)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) UpdateStatusByID(ctx context.Context, id uint64, status domain.ProductStatus) error {
	const q = `UPDATE products SET status = ? WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, string(status), id)
	if err != nil {
		return fmt.Errorf("update product status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id uint64, status domain.ProductStatus) error {
	const q = `UPDATE products SET status = ? WHERE id = ?`
	_, err := tx.ExecContext(ctx, q, string(status), id)
	return err
}

func (r *ProductRepo) UpdateStatus(ctx context.Context, id, anchorID uint64, status domain.ProductStatus) error {
	const q = `UPDATE products SET status = ? WHERE id = ? AND anchor_id = ?`
	res, err := r.db.ExecContext(ctx, q, string(status), id, anchorID)
	if err != nil {
		return fmt.Errorf("update product status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) Delete(ctx context.Context, id, anchorID uint64) error {
	const q = `DELETE FROM products WHERE id = ? AND anchor_id = ?`
	res, err := r.db.ExecContext(ctx, q, id, anchorID)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) List(ctx context.Context, f ProductFilter) ([]domain.Product, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	var conds []string
	var args []any
	conds = append(conds, "anchor_id = ?")
	args = append(args, f.AnchorID)
	if f.Status != nil {
		conds = append(conds, "status = ?")
		args = append(args, string(*f.Status))
	}
	where := strings.Join(conds, " AND ")

	countQ := `SELECT COUNT(*) FROM products WHERE ` + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	listQ := `SELECT id, anchor_id, name, description, cover_url, images, status, created_at, updated_at
		FROM products WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), f.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var items []domain.Product
	for rows.Next() {
		p, err := scanProductRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *p)
	}
	return items, total, rows.Err()
}

func (r *ProductRepo) scanOne(ctx context.Context, q string, id uint64) (*domain.Product, error) {
	row := r.db.QueryRowContext(ctx, q, id)
	p, err := scanProductRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan product: %w", err)
	}
	return p, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanProductRow(row scannable) (*domain.Product, error) {
	var p domain.Product
	var status string
	var imagesRaw []byte
	err := row.Scan(
		&p.ID, &p.AnchorID, &p.Name, &p.Description, &p.CoverURL,
		&imagesRaw, &status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Status = domain.ProductStatus(status)
	if len(imagesRaw) > 0 {
		_ = json.Unmarshal(imagesRaw, &p.Images)
	}
	return &p, nil
}
