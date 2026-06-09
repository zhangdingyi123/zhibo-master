package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zhibo/backend/internal/domain"
)

type LiveRoomRepo struct {
	db *sql.DB
}

func NewLiveRoomRepo(db *sql.DB) *LiveRoomRepo {
	return &LiveRoomRepo{db: db}
}

func (r *LiveRoomRepo) Create(ctx context.Context, anchorID uint64, title string) (*domain.LiveRoom, error) {
	placeholder := fmt.Sprintf("room_tmp_live_%d", anchorID)
	const q = `INSERT INTO live_rooms (anchor_id, title, room_id, status) VALUES (?, ?, ?, 'idle')`
	res, err := r.db.ExecContext(ctx, q, anchorID, title, placeholder)
	if err != nil {
		return nil, fmt.Errorf("insert live room: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("live room last insert id: %w", err)
	}
	liveRoomID := uint64(id)
	roomID := domain.DefaultLiveRoomID(liveRoomID)
	const uq = `UPDATE live_rooms SET room_id = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, uq, roomID, liveRoomID); err != nil {
		return nil, fmt.Errorf("update live room_id: %w", err)
	}
	return r.GetByID(ctx, liveRoomID)
}

func (r *LiveRoomRepo) GetByID(ctx context.Context, id uint64) (*domain.LiveRoom, error) {
	const q = `SELECT id, anchor_id, title, room_id, status, current_session_id, created_at, updated_at
		FROM live_rooms WHERE id = ?`
	return r.scanOne(ctx, q, id)
}

func (r *LiveRoomRepo) GetByRoomID(ctx context.Context, roomID string) (*domain.LiveRoom, error) {
	const q = `SELECT id, anchor_id, title, room_id, status, current_session_id, created_at, updated_at
		FROM live_rooms WHERE room_id = ?`
	return r.scanOne(ctx, q, roomID)
}

func (r *LiveRoomRepo) ListByAnchor(ctx context.Context, anchorID uint64, limit int) ([]domain.LiveRoom, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	const q = `SELECT id, anchor_id, title, room_id, status, current_session_id, created_at, updated_at
		FROM live_rooms WHERE anchor_id = ? ORDER BY id DESC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, q, anchorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list live rooms: %w", err)
	}
	defer rows.Close()
	var items []domain.LiveRoom
	for rows.Next() {
		lr, err := scanLiveRoomRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *lr)
	}
	if items == nil {
		items = []domain.LiveRoom{}
	}
	return items, rows.Err()
}

func (r *LiveRoomRepo) UpdateStatus(ctx context.Context, id, anchorID uint64, status domain.LiveRoomStatus) error {
	const q = `UPDATE live_rooms SET status = ? WHERE id = ? AND anchor_id = ?`
	res, err := r.db.ExecContext(ctx, q, string(status), id, anchorID)
	if err != nil {
		return fmt.Errorf("update live room status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *LiveRoomRepo) SetCurrentSession(ctx context.Context, id uint64, sessionID *uint64) error {
	var sid sql.NullInt64
	if sessionID != nil {
		sid = sql.NullInt64{Int64: int64(*sessionID), Valid: true}
	}
	const q = `UPDATE live_rooms SET current_session_id = ? WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, sid, id)
	if err != nil {
		return fmt.Errorf("set current session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *LiveRoomRepo) scanOne(ctx context.Context, q string, id any) (*domain.LiveRoom, error) {
	row := r.db.QueryRowContext(ctx, q, id)
	lr, err := scanLiveRoom(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan live room: %w", err)
	}
	return lr, nil
}

type liveRoomScanner interface {
	Scan(dest ...any) error
}

func scanLiveRoom(row liveRoomScanner) (*domain.LiveRoom, error) {
	var lr domain.LiveRoom
	var status string
	var currentSession sql.NullInt64
	err := row.Scan(
		&lr.ID, &lr.AnchorID, &lr.Title, &lr.RoomID, &status, &currentSession,
		&lr.CreatedAt, &lr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	lr.Status = domain.LiveRoomStatus(status)
	if currentSession.Valid {
		sid := uint64(currentSession.Int64)
		lr.CurrentSessionID = &sid
	}
	return &lr, nil
}

func scanLiveRoomRow(rows *sql.Rows) (*domain.LiveRoom, error) {
	return scanLiveRoom(rows)
}
