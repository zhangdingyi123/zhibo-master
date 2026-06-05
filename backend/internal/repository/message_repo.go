package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zhibo/backend/internal/domain"
)

type MessageRepo struct {
	db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

type InsertMessageInput struct {
	UserID    uint64
	EventType domain.MessageEventType
	Category  domain.MessageCategory
	Title     string
	Body      string
	Payload   map[string]any
	DedupeKey string
}

func (r *MessageRepo) Insert(ctx context.Context, in InsertMessageInput) error {
	var payload []byte
	if in.Payload != nil {
		var err error
		payload, err = json.Marshal(in.Payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
	}
	const q = `INSERT INTO user_messages (user_id, event_type, category, title, body, payload, dedupe_key)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = id`
	_, err := r.db.ExecContext(ctx, q,
		in.UserID, in.EventType, in.Category, in.Title, in.Body, nullJSON(payload), nullString(in.DedupeKey))
	return err
}

func (r *MessageRepo) ListByUser(ctx context.Context, userID uint64, unreadOnly bool, page, pageSize int) ([]domain.UserMessage, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	where := `WHERE user_id = ?`
	args := []any{userID}
	if unreadOnly {
		where += ` AND is_read = 0`
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_messages `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `SELECT id, user_id, event_type, category, title, body, payload, is_read, created_at
		FROM user_messages ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []domain.UserMessage
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	return items, total, rows.Err()
}

func (r *MessageRepo) CountUnread(ctx context.Context, userID uint64) (int, error) {
	const q = `SELECT COUNT(*) FROM user_messages WHERE user_id = ? AND is_read = 0`
	var n int
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&n)
	return n, err
}

func (r *MessageRepo) MarkRead(ctx context.Context, userID, messageID uint64) error {
	const q = `UPDATE user_messages SET is_read = 1 WHERE id = ? AND user_id = ? AND is_read = 0`
	res, err := r.db.ExecContext(ctx, q, messageID, userID)
	if err != nil {
		return err
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

func (r *MessageRepo) MarkAllRead(ctx context.Context, userID uint64) (int64, error) {
	const q = `UPDATE user_messages SET is_read = 1 WHERE user_id = ? AND is_read = 0`
	res, err := r.db.ExecContext(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *MessageRepo) GetByID(ctx context.Context, userID, messageID uint64) (*domain.UserMessage, error) {
	const q = `SELECT id, user_id, event_type, category, title, body, payload, is_read, created_at
		FROM user_messages WHERE id = ? AND user_id = ?`
	row := r.db.QueryRowContext(ctx, q, messageID, userID)
	m, err := scanMessage(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type messageScanner interface {
	Scan(dest ...any) error
}

func scanMessage(row messageScanner) (domain.UserMessage, error) {
	var m domain.UserMessage
	var payload sql.NullString
	var isRead int
	err := row.Scan(
		&m.ID, &m.UserID, &m.EventType, &m.Category,
		&m.Title, &m.Body, &payload, &isRead, &m.CreatedAt,
	)
	if err != nil {
		return m, err
	}
	m.IsRead = isRead == 1
	if payload.Valid && payload.String != "" {
		_ = json.Unmarshal([]byte(payload.String), &m.Payload)
	}
	return m, nil
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
