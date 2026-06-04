package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zhibo/backend/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

const userSelectCols = `id, open_id, phone, nickname, avatar, role, created_at, updated_at`

func scanUser(row interface {
	Scan(dest ...any) error
}) (*domain.User, error) {
	var u domain.User
	var role string
	var phone sql.NullString
	err := row.Scan(
		&u.ID, &u.OpenID, &phone, &u.Nickname, &u.Avatar, &role, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role = domain.UserRole(role)
	if phone.Valid {
		u.Phone = phone.String
	}
	return &u, nil
}

func (r *UserRepo) GetByOpenID(ctx context.Context, openID string) (*domain.User, error) {
	q := `SELECT ` + userSelectCols + ` FROM users WHERE open_id = ?`
	u, err := scanUser(r.db.QueryRowContext(ctx, q, openID))
	if err != nil {
		return nil, fmt.Errorf("get user by open_id: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uint64) (*domain.User, error) {
	q := `SELECT ` + userSelectCols + ` FROM users WHERE id = ?`
	u, err := scanUser(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	u, _, err := r.GetByPhoneWithHash(ctx, phone)
	return u, err
}

func (r *UserRepo) GetByPhoneWithHash(ctx context.Context, phone string) (*domain.User, string, error) {
	q := `SELECT ` + userSelectCols + `, password_hash FROM users WHERE phone = ?`
	var u domain.User
	var role string
	var phoneCol sql.NullString
	var hash sql.NullString
	err := r.db.QueryRowContext(ctx, q, phone).Scan(
		&u.ID, &u.OpenID, &phoneCol, &u.Nickname, &u.Avatar, &role, &u.CreatedAt, &u.UpdatedAt, &hash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", domain.ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("get user by phone: %w", err)
	}
	u.Role = domain.UserRole(role)
	if phoneCol.Valid {
		u.Phone = phoneCol.String
	}
	pw := ""
	if hash.Valid {
		pw = hash.String
	}
	return &u, pw, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User, passwordHash string) error {
	const q = `INSERT INTO users (open_id, phone, password_hash, nickname, avatar, role)
		VALUES (?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, u.OpenID, u.Phone, passwordHash, u.Nickname, u.Avatar, string(u.Role))
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	u.ID = uint64(id)
	return nil
}
