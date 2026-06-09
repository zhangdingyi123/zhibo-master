package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zhibo/backend/internal/domain"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

type CreateSessionInput struct {
	ProductID        uint64
	AnchorID         uint64
	RoomID           string
	LiveRoomID       *uint64
	SeqInRoom        uint32
	Rules            domain.AuctionRules
	ScheduledStartAt sql.NullTime
}

const sessionSelectCols = `id, product_id, anchor_id, live_room_id, seq_in_room, room_id, status,
	starting_price, bid_increment, cap_price,
	duration_sec, extend_threshold_sec, extend_sec,
	current_price, bid_count, participant_count, winner_id, version,
	scheduled_start_at, started_at, end_at, settled_at, cancel_reason,
	created_at, updated_at`

func (r *SessionRepo) Create(ctx context.Context, in CreateSessionInput) (*domain.AuctionSession, error) {
	placeholderRoom := in.RoomID
	if placeholderRoom == "" {
		placeholderRoom = fmt.Sprintf("room_tmp_%d", time.Now().UnixNano())
	}
	seq := in.SeqInRoom
	if seq == 0 {
		seq = 1
	}
	var liveRoomID sql.NullInt64
	if in.LiveRoomID != nil {
		liveRoomID = sql.NullInt64{Int64: int64(*in.LiveRoomID), Valid: true}
	}

	const q = `INSERT INTO auction_sessions (
		product_id, anchor_id, live_room_id, seq_in_room, room_id, status,
		starting_price, bid_increment, cap_price,
		duration_sec, extend_threshold_sec, extend_sec,
		current_price, scheduled_start_at
	) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?, ?)`

	var capPrice sql.NullInt64
	if in.Rules.CapPrice != nil {
		capPrice = sql.NullInt64{Int64: *in.Rules.CapPrice, Valid: true}
	}

	res, err := r.db.ExecContext(ctx, q,
		in.ProductID, in.AnchorID, liveRoomID, seq, placeholderRoom,
		in.Rules.StartingPrice, in.Rules.BidIncrement, capPrice,
		in.Rules.DurationSec, in.Rules.ExtendThresholdSec, in.Rules.ExtendSec,
		in.Rules.StartingPrice, in.ScheduledStartAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("session last insert id: %w", err)
	}
	sessionID := uint64(id)

	if in.LiveRoomID == nil {
		roomID := domain.DefaultRoomID(sessionID)
		const uq = `UPDATE auction_sessions SET room_id = ? WHERE id = ?`
		if _, err := r.db.ExecContext(ctx, uq, roomID, sessionID); err != nil {
			return nil, fmt.Errorf("update room_id: %w", err)
		}
	}

	return r.GetByID(ctx, sessionID)
}

func (r *SessionRepo) GetByID(ctx context.Context, id uint64) (*domain.AuctionSession, error) {
	const q = `SELECT ` + sessionSelectCols + ` FROM auction_sessions WHERE id = ?`
	return r.scanOne(ctx, q, id)
}

func (r *SessionRepo) UpdateRules(ctx context.Context, sessionID, anchorID uint64, rules domain.AuctionRules, scheduled sql.NullTime) error {
	var capPrice sql.NullInt64
	if rules.CapPrice != nil {
		capPrice = sql.NullInt64{Int64: *rules.CapPrice, Valid: true}
	}
	const q = `UPDATE auction_sessions SET
		starting_price = ?, bid_increment = ?, cap_price = ?,
		duration_sec = ?, extend_threshold_sec = ?, extend_sec = ?,
		current_price = ?, scheduled_start_at = ?
		WHERE id = ? AND anchor_id = ? AND status = 'pending' AND bid_count = 0`
	res, err := r.db.ExecContext(ctx, q,
		rules.StartingPrice, rules.BidIncrement, capPrice,
		rules.DurationSec, rules.ExtendThresholdSec, rules.ExtendSec,
		rules.StartingPrice, scheduled,
		sessionID, anchorID,
	)
	if err != nil {
		return fmt.Errorf("update session rules: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		s, err := r.GetByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if s.AnchorID != anchorID {
			return domain.ErrForbidden
		}
		if s.Status != domain.SessionStatusPending {
			return domain.ErrRulesNotEditable
		}
		if s.BidCount > 0 {
			return domain.ErrSessionHasBids
		}
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepo) Cancel(ctx context.Context, sessionID, anchorID uint64, reason string) error {
	const q = `UPDATE auction_sessions SET status = 'cancelled', cancel_reason = ?
		WHERE id = ? AND anchor_id = ? AND status IN ('pending', 'running')`
	res, err := r.db.ExecContext(ctx, q, reason, sessionID, anchorID)
	if err != nil {
		return fmt.Errorf("cancel session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		s, err := r.GetByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if s.AnchorID != anchorID {
			return domain.ErrForbidden
		}
		return domain.ErrSessionNotCancellable
	}
	return nil
}

// ListExpiredRunning 倒计时已结束但仍为 running 的场次（到时落锤扫描）
func (r *SessionRepo) ListExpiredRunning(ctx context.Context, now time.Time, limit int) ([]uint64, error) {
	if limit < 1 {
		limit = 50
	}
	const q = `SELECT id FROM auction_sessions
		WHERE status = 'running' AND end_at IS NOT NULL AND end_at <= ?
		ORDER BY end_at ASC
		LIMIT ?`
	rows, err := r.db.QueryContext(ctx, q, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list expired running sessions: %w", err)
	}
	defer rows.Close()
	var ids []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *SessionRepo) MarkFailed(ctx context.Context, sessionID uint64, reason string) error {
	const q = `UPDATE auction_sessions SET status = 'failed', cancel_reason = ?
		WHERE id = ? AND status = 'running'`
	res, err := r.db.ExecContext(ctx, q, reason, sessionID)
	if err != nil {
		return fmt.Errorf("mark session failed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrInvalidStateTransition
	}
	return nil
}

func (r *SessionRepo) MarkSettled(ctx context.Context, sessionID uint64, winnerID uint64, finalPrice int64) error {
	const q = `UPDATE auction_sessions SET
		status = 'settled', winner_id = ?, current_price = ?, settled_at = NOW(3)
		WHERE id = ? AND status = 'running'`
	res, err := r.db.ExecContext(ctx, q, winnerID, finalPrice, sessionID)
	if err != nil {
		return fmt.Errorf("settle session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrInvalidStateTransition
	}
	return nil
}

func (r *SessionRepo) GetActiveByProductID(ctx context.Context, productID uint64) (*domain.AuctionSession, error) {
	const q = `SELECT ` + sessionSelectCols + `
		FROM auction_sessions WHERE product_id = ? AND status IN ('pending', 'running')
		ORDER BY id DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, productID)
	s, err := scanSessionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active session: %w", err)
	}
	return s, nil
}

func (r *SessionRepo) MapActiveByProductIDs(ctx context.Context, productIDs []uint64) (map[uint64]*domain.AuctionSession, error) {
	if len(productIDs) == 0 {
		return map[uint64]*domain.AuctionSession{}, nil
	}
	ph := placeholders(len(productIDs))
	args := uint64sToAny(productIDs)

	q := `SELECT s.id, s.product_id, s.anchor_id, s.live_room_id, s.seq_in_room, s.room_id, s.status,
		s.starting_price, s.bid_increment, s.cap_price,
		s.duration_sec, s.extend_threshold_sec, s.extend_sec,
		s.current_price, s.bid_count, s.participant_count, s.winner_id, s.version,
		s.scheduled_start_at, s.started_at, s.end_at, s.settled_at, s.cancel_reason,
		s.created_at, s.updated_at
		FROM auction_sessions s
		INNER JOIN (
			SELECT product_id, MAX(id) AS max_id FROM auction_sessions
			WHERE product_id IN (` + ph + `) AND status IN ('pending', 'running')
			GROUP BY product_id
		) t ON s.product_id = t.product_id AND s.id = t.max_id`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("map active sessions: %w", err)
	}
	defer rows.Close()

	out := make(map[uint64]*domain.AuctionSession)
	for rows.Next() {
		s, err := scanSessionFromRows(rows)
		if err != nil {
			return nil, err
		}
		out[s.ProductID] = s
	}
	return out, rows.Err()
}

func (r *SessionRepo) MapLatestByProductIDs(ctx context.Context, productIDs []uint64) (map[uint64]*domain.AuctionSession, error) {
	if len(productIDs) == 0 {
		return map[uint64]*domain.AuctionSession{}, nil
	}
	ph := placeholders(len(productIDs))
	args := uint64sToAny(productIDs)

	q := `SELECT s.id, s.product_id, s.anchor_id, s.live_room_id, s.seq_in_room, s.room_id, s.status,
		s.starting_price, s.bid_increment, s.cap_price,
		s.duration_sec, s.extend_threshold_sec, s.extend_sec,
		s.current_price, s.bid_count, s.participant_count, s.winner_id, s.version,
		s.scheduled_start_at, s.started_at, s.end_at, s.settled_at, s.cancel_reason,
		s.created_at, s.updated_at
		FROM auction_sessions s
		INNER JOIN (
			SELECT product_id, MAX(id) AS max_id FROM auction_sessions WHERE product_id IN (` + ph + `) GROUP BY product_id
		) t ON s.product_id = t.product_id AND s.id = t.max_id`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("map latest sessions: %w", err)
	}
	defer rows.Close()

	out := make(map[uint64]*domain.AuctionSession)
	for rows.Next() {
		s, err := scanSessionFromRows(rows)
		if err != nil {
			return nil, err
		}
		out[s.ProductID] = s
	}
	return out, rows.Err()
}

// PublicSessionRow 用户端场次 + 商品摘要
type PublicSessionRow struct {
	Session            domain.AuctionSession
	ProductName        string
	ProductCoverURL    string
	ProductDescription string
}

type PublicSessionFilter struct {
	Statuses []domain.SessionStatus
	Page     int
	PageSize int
}

func (r *SessionRepo) GetByRoomID(ctx context.Context, roomID string) (*domain.AuctionSession, error) {
	const qLive = `SELECT current_session_id FROM live_rooms WHERE room_id = ? AND current_session_id IS NOT NULL LIMIT 1`
	var currentID uint64
	err := r.db.QueryRowContext(ctx, qLive, roomID).Scan(&currentID)
	if err == nil && currentID > 0 {
		return r.GetByID(ctx, currentID)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get live room current session: %w", err)
	}

	const qActive = `SELECT ` + sessionSelectCols + `
		FROM auction_sessions WHERE room_id = ? AND status IN ('pending', 'running')
		ORDER BY id DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, qActive, roomID)
	s, err := scanSessionRow(row)
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get active session by room: %w", err)
	}

	const qAny = `SELECT ` + sessionSelectCols + `
		FROM auction_sessions WHERE room_id = ? ORDER BY id DESC LIMIT 1`
	row = r.db.QueryRowContext(ctx, qAny, roomID)
	s, err = scanSessionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session by room: %w", err)
	}
	return s, nil
}

func (r *SessionRepo) NextSeqInLiveRoom(ctx context.Context, liveRoomID uint64) (uint32, error) {
	const q = `SELECT COALESCE(MAX(seq_in_room), 0) + 1 FROM auction_sessions WHERE live_room_id = ?`
	var seq uint32
	if err := r.db.QueryRowContext(ctx, q, liveRoomID).Scan(&seq); err != nil {
		return 0, fmt.Errorf("next seq in live room: %w", err)
	}
	return seq, nil
}

func (r *SessionRepo) ListByLiveRoomID(ctx context.Context, liveRoomID uint64) ([]domain.AuctionSession, error) {
	const q = `SELECT ` + sessionSelectCols + `
		FROM auction_sessions WHERE live_room_id = ? ORDER BY seq_in_room ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, liveRoomID)
	if err != nil {
		return nil, fmt.Errorf("list sessions by live room: %w", err)
	}
	defer rows.Close()
	var items []domain.AuctionSession
	for rows.Next() {
		s, err := scanSessionFromRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *s)
	}
	if items == nil {
		items = []domain.AuctionSession{}
	}
	return items, rows.Err()
}

func (r *SessionRepo) GetNextPendingInLiveRoom(ctx context.Context, liveRoomID uint64, afterSeq uint32) (*domain.AuctionSession, error) {
	const q = `SELECT ` + sessionSelectCols + `
		FROM auction_sessions
		WHERE live_room_id = ? AND status = 'pending' AND seq_in_room > ?
		ORDER BY seq_in_room ASC LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, liveRoomID, afterSeq)
	s, err := scanSessionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get next pending session: %w", err)
	}
	return s, nil
}

func (r *SessionRepo) ListPublic(ctx context.Context, f PublicSessionFilter) ([]PublicSessionRow, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if len(f.Statuses) == 0 {
		f.Statuses = []domain.SessionStatus{
			domain.SessionStatusPending,
			domain.SessionStatusRunning,
		}
	}
	offset := (f.Page - 1) * f.PageSize

	ph := placeholders(len(f.Statuses))
	statusArgs := make([]any, len(f.Statuses))
	for i, st := range f.Statuses {
		statusArgs[i] = string(st)
	}

	countQ := `SELECT COUNT(*) FROM auction_sessions s
		INNER JOIN products p ON s.product_id = p.id
		WHERE s.status IN (` + ph + `) AND p.status IN ('listed', 'auctioning', 'sold')`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, statusArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count public sessions: %w", err)
	}

	listQ := `SELECT s.id, s.product_id, s.anchor_id, s.live_room_id, s.seq_in_room, s.room_id, s.status,
		s.starting_price, s.bid_increment, s.cap_price,
		s.duration_sec, s.extend_threshold_sec, s.extend_sec,
		s.current_price, s.bid_count, s.participant_count, s.winner_id, s.version,
		s.scheduled_start_at, s.started_at, s.end_at, s.settled_at, s.cancel_reason,
		s.created_at, s.updated_at,
		p.name, p.cover_url, COALESCE(p.description, '')
		FROM auction_sessions s
		INNER JOIN products p ON s.product_id = p.id
		WHERE s.status IN (` + ph + `) AND p.status IN ('listed', 'auctioning', 'sold')
		ORDER BY FIELD(s.status, 'running', 'pending', 'settled', 'cancelled', 'failed'), s.id DESC
		LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, statusArgs...), f.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list public sessions: %w", err)
	}
	defer rows.Close()

	var items []PublicSessionRow
	for rows.Next() {
		item, err := scanPublicSessionRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	if items == nil {
		items = []PublicSessionRow{}
	}
	return items, total, rows.Err()
}

func (r *SessionRepo) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id uint64) (*domain.AuctionSession, error) {
	const q = `SELECT ` + sessionSelectCols + ` FROM auction_sessions WHERE id = ? FOR UPDATE`
	row := tx.QueryRowContext(ctx, q, id)
	s, err := scanSessionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

type ApplyBidParams struct {
	SessionID          uint64
	Version            uint32
	CurrentPrice       int64
	BidCount           uint32
	ParticipantCount   uint32
	Status             domain.SessionStatus
	StartedAt          *time.Time
	EndAt              *time.Time
}

func (r *SessionRepo) ApplyBid(ctx context.Context, tx *sql.Tx, p ApplyBidParams) error {
	var startedAt, endAt sql.NullTime
	if p.StartedAt != nil {
		startedAt = sql.NullTime{Time: *p.StartedAt, Valid: true}
	}
	if p.EndAt != nil {
		endAt = sql.NullTime{Time: *p.EndAt, Valid: true}
	}
	const q = `UPDATE auction_sessions SET
		status = ?, current_price = ?, bid_count = ?, participant_count = ?,
		version = version + 1, started_at = COALESCE(started_at, ?), end_at = ?
		WHERE id = ? AND version = ?`
	res, err := tx.ExecContext(ctx, q,
		string(p.Status), p.CurrentPrice, p.BidCount, p.ParticipantCount,
		startedAt, endAt, p.SessionID, p.Version,
	)
	if err != nil {
		return fmt.Errorf("apply bid: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

func (r *SessionRepo) MarkSettledTx(ctx context.Context, tx *sql.Tx, sessionID uint64, winnerID uint64, finalPrice int64) error {
	const q = `UPDATE auction_sessions SET
		status = 'settled', winner_id = ?, current_price = ?, settled_at = NOW(3)
		WHERE id = ? AND status IN ('pending', 'running')`
	res, err := tx.ExecContext(ctx, q, winnerID, finalPrice, sessionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrInvalidStateTransition
	}
	return nil
}

func scanPublicSessionRow(rows *sql.Rows) (*PublicSessionRow, error) {
	var item PublicSessionRow
	var status string
	var capPrice sql.NullInt64
	var winnerID sql.NullInt64
	var scheduledStart, startedAt, endAt, settledAt sql.NullTime
	var cancelReason sql.NullString
	s := &item.Session

	var liveRoomID sql.NullInt64
	err := rows.Scan(
		&s.ID, &s.ProductID, &s.AnchorID, &liveRoomID, &s.SeqInRoom, &s.RoomID, &status,
		&s.Rules.StartingPrice, &s.Rules.BidIncrement, &capPrice,
		&s.Rules.DurationSec, &s.Rules.ExtendThresholdSec, &s.Rules.ExtendSec,
		&s.CurrentPrice, &s.BidCount, &s.ParticipantCount, &winnerID, &s.Version,
		&scheduledStart, &startedAt, &endAt, &settledAt, &cancelReason,
		&s.CreatedAt, &s.UpdatedAt,
		&item.ProductName, &item.ProductCoverURL, &item.ProductDescription,
	)
	if err != nil {
		return nil, err
	}
	s.Status = domain.SessionStatus(status)
	applySessionScalars(s, liveRoomID, capPrice, winnerID, scheduledStart, startedAt, endAt, settledAt, cancelReason)
	return &item, nil
}

func (r *SessionRepo) HasActiveByProductID(ctx context.Context, productID uint64) (bool, error) {
	const q = `SELECT 1 FROM auction_sessions
		WHERE product_id = ? AND status IN ('pending', 'running') LIMIT 1`
	var one int
	err := r.db.QueryRowContext(ctx, q, productID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check active session: %w", err)
	}
	return true, nil
}

func (r *SessionRepo) scanOne(ctx context.Context, q string, id uint64) (*domain.AuctionSession, error) {
	row := r.db.QueryRowContext(ctx, q, id)
	s, err := scanSessionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}
	return s, nil
}

func scanSessionRow(row *sql.Row) (*domain.AuctionSession, error) {
	return scanSession(scanRow{row})
}

func scanSessionFromRows(rows *sql.Rows) (*domain.AuctionSession, error) {
	return scanSession(rows)
}

type rowScanner interface {
	Scan(dest ...any) error
}

type scanRow struct{ *sql.Row }

func scanSession(row rowScanner) (*domain.AuctionSession, error) {
	var s domain.AuctionSession
	var status string
	var liveRoomID sql.NullInt64
	var capPrice sql.NullInt64
	var winnerID sql.NullInt64
	var scheduledStart, startedAt, endAt, settledAt sql.NullTime
	var cancelReason sql.NullString

	err := row.Scan(
		&s.ID, &s.ProductID, &s.AnchorID, &liveRoomID, &s.SeqInRoom, &s.RoomID, &status,
		&s.Rules.StartingPrice, &s.Rules.BidIncrement, &capPrice,
		&s.Rules.DurationSec, &s.Rules.ExtendThresholdSec, &s.Rules.ExtendSec,
		&s.CurrentPrice, &s.BidCount, &s.ParticipantCount, &winnerID, &s.Version,
		&scheduledStart, &startedAt, &endAt, &settledAt, &cancelReason,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Status = domain.SessionStatus(status)
	applySessionScalars(&s, liveRoomID, capPrice, winnerID, scheduledStart, startedAt, endAt, settledAt, cancelReason)
	return &s, nil
}

func applySessionScalars(
	s *domain.AuctionSession,
	liveRoomID sql.NullInt64,
	capPrice sql.NullInt64,
	winnerID sql.NullInt64,
	scheduledStart, startedAt, endAt, settledAt sql.NullTime,
	cancelReason sql.NullString,
) {
	if liveRoomID.Valid {
		lid := uint64(liveRoomID.Int64)
		s.LiveRoomID = &lid
	}
	if capPrice.Valid {
		v := capPrice.Int64
		s.Rules.CapPrice = &v
	}
	if winnerID.Valid {
		w := uint64(winnerID.Int64)
		s.WinnerID = &w
	}
	if scheduledStart.Valid {
		t := scheduledStart.Time
		s.ScheduledStartAt = &t
	}
	if startedAt.Valid {
		t := startedAt.Time
		s.StartedAt = &t
	}
	if endAt.Valid {
		t := endAt.Time
		s.EndAt = &t
	}
	if settledAt.Valid {
		t := settledAt.Time
		s.SettledAt = &t
	}
	if cancelReason.Valid {
		cr := cancelReason.String
		s.CancelReason = &cr
	}
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n*2-1)
	for i := 0; i < n; i++ {
		if i > 0 {
			b[i*2-1] = ','
		}
		b[i*2] = '?'
	}
	return string(b)
}

func uint64sToAny(ids []uint64) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}
