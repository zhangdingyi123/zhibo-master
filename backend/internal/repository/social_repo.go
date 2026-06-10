package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/zhibo/backend/internal/domain"
)

type SocialRepo struct {
	db *sql.DB
}

func NewSocialRepo(db *sql.DB) *SocialRepo {
	return &SocialRepo{db: db}
}

func (r *SocialRepo) InsertComment(ctx context.Context, roomID string, userID uint64, content string) (uint64, error) {
	const q = `INSERT INTO room_comments (room_id, user_id, content) VALUES (?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, roomID, userID, content)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (r *SocialRepo) GetComment(ctx context.Context, id uint64) (*domain.RoomComment, error) {
	const q = `SELECT c.id, c.room_id, c.user_id, u.nickname, u.avatar, c.content, c.is_hidden, c.created_at
		FROM room_comments c JOIN users u ON u.id = c.user_id WHERE c.id = ?`
	return scanComment(r.db.QueryRowContext(ctx, q, id))
}

func (r *SocialRepo) ListComments(ctx context.Context, roomID string, includeHidden bool, limit int) ([]domain.RoomComment, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	where := `WHERE c.room_id = ?`
	if !includeHidden {
		where += ` AND c.is_hidden = 0`
	}
	q := `SELECT c.id, c.room_id, c.user_id, u.nickname, u.avatar, c.content, c.is_hidden, c.created_at
		FROM room_comments c JOIN users u ON u.id = c.user_id ` + where +
		` ORDER BY c.created_at DESC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, q, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.RoomComment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *c)
	}
	return items, rows.Err()
}

func (r *SocialRepo) HideComment(ctx context.Context, id uint64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE room_comments SET is_hidden = 1 WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SocialRepo) CountComments(ctx context.Context, roomID string, visibleOnly bool) (int, error) {
	q := `SELECT COUNT(*) FROM room_comments WHERE room_id = ?`
	if visibleOnly {
		q += ` AND is_hidden = 0`
	}
	var n int
	err := r.db.QueryRowContext(ctx, q, roomID).Scan(&n)
	return n, err
}

func (r *SocialRepo) InsertLike(ctx context.Context, roomID string, userID uint64) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO room_likes (room_id, user_id) VALUES (?, ?)`, roomID, userID)
	return err
}

func (r *SocialRepo) CountLikes(ctx context.Context, roomID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM room_likes WHERE room_id = ?`, roomID).Scan(&n)
	return n, err
}

func (r *SocialRepo) ToggleFavorite(ctx context.Context, userID, productID uint64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM product_favorites WHERE user_id = ? AND product_id = ?`,
		userID, productID).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists > 0 {
		_, err = r.db.ExecContext(ctx,
			`DELETE FROM product_favorites WHERE user_id = ? AND product_id = ?`, userID, productID)
		return false, err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO product_favorites (user_id, product_id) VALUES (?, ?)`, userID, productID)
	return true, err
}

func (r *SocialRepo) IsFavorite(ctx context.Context, userID, productID uint64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM product_favorites WHERE user_id = ? AND product_id = ?`,
		userID, productID).Scan(&n)
	return n > 0, err
}

func (r *SocialRepo) ToggleFollow(ctx context.Context, userID, anchorID uint64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM anchor_follows WHERE user_id = ? AND anchor_id = ?`,
		userID, anchorID).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists > 0 {
		_, err = r.db.ExecContext(ctx,
			`DELETE FROM anchor_follows WHERE user_id = ? AND anchor_id = ?`, userID, anchorID)
		return false, err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO anchor_follows (user_id, anchor_id) VALUES (?, ?)`, userID, anchorID)
	return true, err
}

func (r *SocialRepo) IsFollowing(ctx context.Context, userID, anchorID uint64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM anchor_follows WHERE user_id = ? AND anchor_id = ?`,
		userID, anchorID).Scan(&n)
	return n > 0, err
}

func (r *SocialRepo) CountFollowers(ctx context.Context, anchorID uint64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM anchor_follows WHERE anchor_id = ?`, anchorID).Scan(&n)
	return n, err
}

func scanComment(row interface {
	Scan(dest ...any) error
}) (*domain.RoomComment, error) {
	var c domain.RoomComment
	var hidden int
	err := row.Scan(&c.ID, &c.RoomID, &c.UserID, &c.Nickname, &c.Avatar, &c.Content, &hidden, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan comment: %w", err)
	}
	c.IsHidden = hidden == 1
	c.Content = strings.TrimSpace(c.Content)
	return &c, nil
}
