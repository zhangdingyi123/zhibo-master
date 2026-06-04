package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zhibo/backend/internal/domain"
)

type BidRepo struct {
	db *sql.DB
}

func NewBidRepo(db *sql.DB) *BidRepo {
	return &BidRepo{db: db}
}

func (r *BidRepo) GetByRequestID(ctx context.Context, sessionID uint64, requestID string) (*domain.Bid, error) {
	const q = `SELECT id, session_id, user_id, amount, request_id, seq, is_winning, created_at
		FROM bids WHERE session_id = ? AND request_id = ?`
	row := r.db.QueryRowContext(ctx, q, sessionID, requestID)
	return scanBidRow(row)
}

func (r *BidRepo) GetByRequestIDTx(ctx context.Context, tx *sql.Tx, sessionID uint64, requestID string) (*domain.Bid, error) {
	const q = `SELECT id, session_id, user_id, amount, request_id, seq, is_winning, created_at
		FROM bids WHERE session_id = ? AND request_id = ?`
	row := tx.QueryRowContext(ctx, q, sessionID, requestID)
	return scanBidRow(row)
}

func (r *BidRepo) UserHasBid(ctx context.Context, tx *sql.Tx, sessionID, userID uint64) (bool, error) {
	const q = `SELECT 1 FROM bids WHERE session_id = ? AND user_id = ? LIMIT 1`
	var one int
	err := tx.QueryRowContext(ctx, q, sessionID, userID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *BidRepo) NextSeq(ctx context.Context, tx *sql.Tx, sessionID uint64) (uint32, error) {
	const q = `SELECT COALESCE(MAX(seq), 0) + 1 FROM bids WHERE session_id = ? FOR UPDATE`
	var seq uint32
	if err := tx.QueryRowContext(ctx, q, sessionID).Scan(&seq); err != nil {
		return 0, fmt.Errorf("next bid seq: %w", err)
	}
	return seq, nil
}

func (r *BidRepo) ClearWinning(ctx context.Context, tx *sql.Tx, sessionID uint64) error {
	const q = `UPDATE bids SET is_winning = 0 WHERE session_id = ? AND is_winning = 1`
	_, err := tx.ExecContext(ctx, q, sessionID)
	return err
}

func (r *BidRepo) Create(ctx context.Context, tx *sql.Tx, b *domain.Bid) error {
	if err := r.ClearWinning(ctx, tx, b.SessionID); err != nil {
		return err
	}
	const q = `INSERT INTO bids (session_id, user_id, amount, request_id, seq, is_winning)
		VALUES (?, ?, ?, ?, ?, 1)`
	res, err := tx.ExecContext(ctx, q, b.SessionID, b.UserID, b.Amount, b.RequestID, b.Seq)
	if err != nil {
		return fmt.Errorf("insert bid: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	b.ID = uint64(id)
	b.IsWinning = true
	return nil
}

// BidRankEntry 排行榜行
type BidRankEntry struct {
	UserID   uint64
	Nickname string
	Avatar   string
	Amount   int64
	Seq      uint32
	Rank     int
}

// ListTopBySession 场次出价排行榜（每用户取最高出价）
func (r *BidRepo) ListTopBySession(ctx context.Context, sessionID uint64, limit int) ([]BidRankEntry, error) {
	if limit < 1 {
		limit = 10
	}
	const q = `
		SELECT u.id, u.nickname, u.avatar, t.amount, t.seq
		FROM (
			SELECT user_id, MAX(amount) AS amount, MAX(seq) AS seq
			FROM bids
			WHERE session_id = ?
			GROUP BY user_id
		) t
		INNER JOIN users u ON u.id = t.user_id
		ORDER BY t.amount DESC, t.seq ASC
		LIMIT ?`
	rows, err := r.db.QueryContext(ctx, q, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list top bids: %w", err)
	}
	defer rows.Close()

	var out []BidRankEntry
	rank := 1
	for rows.Next() {
		var e BidRankEntry
		if err := rows.Scan(&e.UserID, &e.Nickname, &e.Avatar, &e.Amount, &e.Seq); err != nil {
			return nil, err
		}
		e.Rank = rank
		rank++
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanBidRow(row *sql.Row) (*domain.Bid, error) {
	var b domain.Bid
	var winning int
	err := row.Scan(&b.ID, &b.SessionID, &b.UserID, &b.Amount, &b.RequestID, &b.Seq, &winning, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	b.IsWinning = winning == 1
	return &b, nil
}
