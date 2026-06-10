package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

// RoomViewerCounter 房间实时进房人数（WS 连接数）
type RoomViewerCounter interface {
	ClientCount(roomID string) int
}

type LiveRoomService struct {
	liveRooms *repository.LiveRoomRepo
	sessions  *repository.SessionRepo
	products  *repository.ProductRepo
	orders    *repository.OrderRepo
	users     *repository.UserRepo
	social    *repository.SocialRepo
	auction   *AuctionService
	cache     RoomCache
	notify    RoomNotifier
	viewers   RoomViewerCounter
}

func NewLiveRoomService(
	liveRooms *repository.LiveRoomRepo,
	sessions *repository.SessionRepo,
	products *repository.ProductRepo,
	orders *repository.OrderRepo,
	auction *AuctionService,
) *LiveRoomService {
	return &LiveRoomService{
		liveRooms: liveRooms,
		sessions:  sessions,
		products:  products,
		orders:    orders,
		auction:   auction,
		notify:    NoopRoomNotifier{},
	}
}

func (s *LiveRoomService) SetRoomCache(c RoomCache)                 { s.cache = c }
func (s *LiveRoomService) SetRoomNotifier(n RoomNotifier)           { s.notify = n }
func (s *LiveRoomService) SetRoomViewerCounter(v RoomViewerCounter) { s.viewers = v }
func (s *LiveRoomService) SetUserRepo(u *repository.UserRepo)       { s.users = u }
func (s *LiveRoomService) SetSocialRepo(social *repository.SocialRepo) { s.social = social }

type CreateLiveRoomInput struct {
	Title string
}

type AddSessionToLiveRoomInput struct {
	ProductID        uint64
	Rules            domain.AuctionRules
	ScheduledStartAt *time.Time
}

type LiveRoomSessionItem struct {
	Session domain.AuctionSession `json:"session"`
	Product ProductBrief          `json:"product"`
}

// LiveRoomFunnel 直播转化漏斗：进房 → 出价 → 成交 → 支付
type LiveRoomFunnel struct {
	ViewerCount  int    `json:"viewerCount"`
	BidderCount  int    `json:"bidderCount"`
	SettledCount int    `json:"settledCount"`
	PaidCount    int    `json:"paidCount"`
	Hint         string `json:"hint,omitempty"`
}

type LiveRoomDetail struct {
	LiveRoom       domain.LiveRoom       `json:"liveRoom"`
	CurrentSession *LiveRoomSessionItem  `json:"currentSession,omitempty"`
	Queue          []LiveRoomSessionItem `json:"queue"`
	History        []LiveRoomSessionItem `json:"history"`
	Funnel         LiveRoomFunnel        `json:"funnel"`
}

type UserLiveRoomDetail struct {
	RoomID         string                `json:"roomId"`
	LiveRoom       domain.LiveRoom       `json:"liveRoom"`
	Anchor         *AnchorBrief          `json:"anchor,omitempty"`
	Current        *UserAuctionDetail    `json:"current,omitempty"`
	History        []SessionSummary      `json:"history"`
}

type SessionSummary struct {
	SessionID   uint64               `json:"sessionId"`
	ProductID   uint64               `json:"productId"`
	ProductName string               `json:"productName"`
	CoverURL    string               `json:"coverUrl"`
	Status      domain.SessionStatus `json:"status"`
	FinalPrice  int64                `json:"finalPrice"`
	WinnerID    *uint64              `json:"winnerId,omitempty"`
	SeqInRoom   uint32               `json:"seqInRoom"`
}

func (s *LiveRoomService) Create(ctx context.Context, anchorID uint64, in CreateLiveRoomInput) (*domain.LiveRoom, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, domain.ErrLiveRoomTitleRequired
	}
	return s.liveRooms.Create(ctx, anchorID, title)
}

func (s *LiveRoomService) List(ctx context.Context, anchorID uint64) ([]domain.LiveRoom, error) {
	return s.liveRooms.ListByAnchor(ctx, anchorID, 50)
}

func (s *LiveRoomService) GetAdminDetail(ctx context.Context, anchorID, liveRoomID uint64) (*LiveRoomDetail, error) {
	lr, err := s.liveRooms.GetByID(ctx, liveRoomID)
	if err != nil {
		return nil, err
	}
	if lr.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	return s.buildDetail(ctx, lr)
}

func (s *LiveRoomService) GetUserDetail(ctx context.Context, roomID string) (*UserLiveRoomDetail, error) {
	lr, err := s.liveRooms.GetByRoomID(ctx, roomID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return s.getUserDetailLegacy(ctx, roomID)
		}
		return nil, err
	}
	sessions, err := s.sessions.ListByLiveRoomID(ctx, lr.ID)
	if err != nil {
		return nil, err
	}

	out := &UserLiveRoomDetail{
		RoomID:   lr.RoomID,
		LiveRoom: *lr,
		History:  []SessionSummary{},
	}
	out.Anchor = s.buildAnchorBrief(ctx, lr.AnchorID)

	for _, sess := range sessions {
		p, err := s.products.GetByID(ctx, sess.ProductID)
		if err != nil {
			continue
		}
		summary := SessionSummary{
			SessionID:   sess.ID,
			ProductID:   sess.ProductID,
			ProductName: p.Name,
			CoverURL:    p.CoverURL,
			Status:      sess.Status,
			FinalPrice:  sess.CurrentPrice,
			WinnerID:    sess.WinnerID,
			SeqInRoom:   sess.SeqInRoom,
		}
		if sess.ID == ptrUint64(lr.CurrentSessionID) {
			detail, err := s.buildUserSessionDetail(ctx, &sess, p)
			if err == nil {
				out.Current = detail
			}
		}
		if isHistoryStatus(sess.Status) {
			out.History = append(out.History, summary)
		}
	}
	return out, nil
}

func (s *LiveRoomService) Start(ctx context.Context, anchorID, liveRoomID uint64) (*domain.LiveRoom, error) {
	lr, err := s.liveRooms.GetByID(ctx, liveRoomID)
	if err != nil {
		return nil, err
	}
	if lr.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	if lr.Status == domain.LiveRoomStatusEnded {
		return nil, domain.ErrLiveRoomNotLive
	}
	if err := s.liveRooms.UpdateStatus(ctx, liveRoomID, anchorID, domain.LiveRoomStatusLive); err != nil {
		return nil, err
	}
	return s.liveRooms.GetByID(ctx, liveRoomID)
}

func (s *LiveRoomService) End(ctx context.Context, anchorID, liveRoomID uint64) (*domain.LiveRoom, error) {
	lr, err := s.liveRooms.GetByID(ctx, liveRoomID)
	if err != nil {
		return nil, err
	}
	if lr.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	if lr.CurrentSessionID != nil {
		sess, err := s.sessions.GetByID(ctx, *lr.CurrentSessionID)
		if err == nil && (sess.Status == domain.SessionStatusPending || sess.Status == domain.SessionStatusRunning) {
			return nil, domain.ErrLiveRoomSessionBusy
		}
	}
	if err := s.liveRooms.UpdateStatus(ctx, liveRoomID, anchorID, domain.LiveRoomStatusEnded); err != nil {
		return nil, err
	}
	return s.liveRooms.GetByID(ctx, liveRoomID)
}

func (s *LiveRoomService) AddSession(ctx context.Context, anchorID, liveRoomID uint64, in AddSessionToLiveRoomInput) (*domain.AuctionSession, error) {
	if err := in.Rules.Validate(); err != nil {
		return nil, err
	}
	lr, err := s.liveRooms.GetByID(ctx, liveRoomID)
	if err != nil {
		return nil, err
	}
	if lr.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	if lr.Status == domain.LiveRoomStatusEnded {
		return nil, domain.ErrLiveRoomNotLive
	}

	p, err := s.products.GetByID(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	if p.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	switch p.Status {
	case domain.ProductStatusOffShelf, domain.ProductStatusSold, domain.ProductStatusAuctioning:
		return nil, domain.ErrProductNotPublishable
	}
	active, err := s.sessions.HasActiveByProductID(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, domain.ErrActiveSessionExists
	}

	seq, err := s.sessions.NextSeqInLiveRoom(ctx, liveRoomID)
	if err != nil {
		return nil, err
	}
	var scheduled sql.NullTime
	if in.ScheduledStartAt != nil {
		scheduled = sql.NullTime{Time: *in.ScheduledStartAt, Valid: true}
	}
	lid := liveRoomID
	session, err := s.sessions.Create(ctx, repository.CreateSessionInput{
		ProductID:        in.ProductID,
		AnchorID:         anchorID,
		RoomID:           lr.RoomID,
		LiveRoomID:       &lid,
		SeqInRoom:        seq,
		Rules:            in.Rules,
		ScheduledStartAt: scheduled,
	})
	if err != nil {
		return nil, err
	}

	if p.Status == domain.ProductStatusDraft {
		_ = s.products.UpdateStatus(ctx, in.ProductID, anchorID, domain.ProductStatusListed)
	}

	if lr.CurrentSessionID == nil {
		_ = s.liveRooms.SetCurrentSession(ctx, liveRoomID, &session.ID)
		if lr.Status == domain.LiveRoomStatusIdle {
			_, _ = s.Start(ctx, anchorID, liveRoomID)
		}
	}
	return session, nil
}

type BatchAddSessionsInput struct {
	ProductIDs       []uint64
	Rules            domain.AuctionRules
	ScheduledStartAt *time.Time
}

// AddSessionsBatch 批量将多个商品加入直播队列（共用同一套竞拍规则）
func (s *LiveRoomService) AddSessionsBatch(ctx context.Context, anchorID, liveRoomID uint64, in BatchAddSessionsInput) ([]domain.AuctionSession, error) {
	if len(in.ProductIDs) == 0 {
		return nil, domain.ErrInvalidProductName
	}
	var out []domain.AuctionSession
	for _, pid := range in.ProductIDs {
		sess, err := s.AddSession(ctx, anchorID, liveRoomID, AddSessionToLiveRoomInput{
			ProductID:        pid,
			Rules:            in.Rules,
			ScheduledStartAt: in.ScheduledStartAt,
		})
		if err != nil {
			return out, err
		}
		out = append(out, *sess)
	}
	return out, nil
}

func (s *LiveRoomService) buildAnchorBrief(ctx context.Context, anchorID uint64) *AnchorBrief {
	if s.users == nil {
		return nil
	}
	u, err := s.users.GetByID(ctx, anchorID)
	if err != nil {
		return nil
	}
	brief := &AnchorBrief{
		ID:       u.ID,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
	}
	if s.social != nil {
		brief.FollowerCount, _ = s.social.CountFollowers(ctx, anchorID)
	}
	return brief
}

// EndCurrentAndSwitch 结束当前品并切换到队列中下一 pending 场次
func (s *LiveRoomService) EndCurrentAndSwitch(ctx context.Context, anchorID, liveRoomID uint64) (*LiveRoomDetail, error) {
	lr, err := s.liveRooms.GetByID(ctx, liveRoomID)
	if err != nil {
		return nil, err
	}
	if lr.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	if lr.Status != domain.LiveRoomStatusLive {
		return nil, domain.ErrLiveRoomNotLive
	}
	if lr.CurrentSessionID == nil {
		return nil, domain.ErrLiveRoomNoCurrent
	}

	current, err := s.sessions.GetByID(ctx, *lr.CurrentSessionID)
	if err != nil {
		return nil, err
	}

	var previousSummary *SessionSummary
	switch current.Status {
	case domain.SessionStatusRunning:
		if current.BidCount == 0 {
			_ = s.sessions.MarkFailed(ctx, current.ID, "主播结束当前品，无有效出价")
		} else {
			winnerID, err := s.auction.resolveWinner(ctx, current.ID)
			if err != nil {
				return nil, err
			}
			if winnerID == 0 {
				return nil, domain.ErrSettlementNoWinner
			}
			_, err = s.auction.CompleteSettlement(ctx, current.ID, winnerID, current.CurrentPrice)
			if err != nil {
				return nil, err
			}
		}
		current, _ = s.sessions.GetByID(ctx, current.ID)
	case domain.SessionStatusPending:
		_, err = s.auction.Cancel(ctx, anchorID, current.ID, "主播跳过当前品")
		if err != nil {
			return nil, err
		}
		current, _ = s.sessions.GetByID(ctx, current.ID)
	case domain.SessionStatusSettled, domain.SessionStatusCancelled, domain.SessionStatusFailed:
		// 已终态，直接切下一品
	default:
		return nil, domain.ErrLiveRoomSessionBusy
	}

	if p, err := s.products.GetByID(ctx, current.ProductID); err == nil {
		previousSummary = &SessionSummary{
			SessionID:   current.ID,
			ProductID:   current.ProductID,
			ProductName: p.Name,
			CoverURL:    p.CoverURL,
			Status:      current.Status,
			FinalPrice:  current.CurrentPrice,
			WinnerID:    current.WinnerID,
			SeqInRoom:   current.SeqInRoom,
		}
	}

	next, err := s.sessions.GetNextPendingInLiveRoom(ctx, liveRoomID, current.SeqInRoom)
	if err != nil {
		return nil, err
	}

	var nextDetail *UserAuctionDetail
	if next != nil {
		if err := s.liveRooms.SetCurrentSession(ctx, liveRoomID, &next.ID); err != nil {
			return nil, err
		}
		p, _ := s.products.GetByID(ctx, next.ProductID)
		if p != nil {
			nextDetail, _ = s.buildUserSessionDetail(ctx, next, p)
		}
	} else {
		if err := s.liveRooms.SetCurrentSession(ctx, liveRoomID, nil); err != nil {
			return nil, err
		}
	}

	detail, err := s.buildDetail(ctx, lr)
	if err != nil {
		return nil, err
	}

	if s.notify != nil {
		history := make([]SessionSummary, 0, len(detail.History))
		for _, h := range detail.History {
			history = append(history, SessionSummary{
				SessionID:   h.Session.ID,
				ProductID:   h.Session.ProductID,
				ProductName: h.Product.Name,
				CoverURL:    h.Product.CoverURL,
				Status:      h.Session.Status,
				FinalPrice:  h.Session.CurrentPrice,
				WinnerID:    h.Session.WinnerID,
				SeqInRoom:   h.Session.SeqInRoom,
			})
		}
		if previousSummary != nil {
			history = append(history, *previousSummary)
		}
		s.notify.OnSessionSwitch(ctx, lr, previousSummary, nextDetail, history)
	}
	return detail, nil
}

func (s *LiveRoomService) buildDetail(ctx context.Context, lr *domain.LiveRoom) (*LiveRoomDetail, error) {
	lr, err := s.liveRooms.GetByID(ctx, lr.ID)
	if err != nil {
		return nil, err
	}
	sessions, err := s.sessions.ListByLiveRoomID(ctx, lr.ID)
	if err != nil {
		return nil, err
	}

	out := &LiveRoomDetail{
		LiveRoom: *lr,
		Queue:    []LiveRoomSessionItem{},
		History:  []LiveRoomSessionItem{},
	}

	sessionIDs := make([]uint64, 0, len(sessions))
	settledCount := 0
	var currentSess *domain.AuctionSession

	for _, sess := range sessions {
		sessionIDs = append(sessionIDs, sess.ID)
		if sess.Status == domain.SessionStatusSettled {
			settledCount++
		}
		p, err := s.products.GetByID(ctx, sess.ProductID)
		if err != nil {
			continue
		}
		item := LiveRoomSessionItem{
			Session: sess,
			Product: ProductBrief{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				CoverURL:    p.CoverURL,
			},
		}
		switch {
		case sess.ID == ptrUint64(lr.CurrentSessionID):
			out.CurrentSession = &item
			cp := sess
			currentSess = &cp
		case sess.Status == domain.SessionStatusPending:
			out.Queue = append(out.Queue, item)
		case isHistoryStatus(sess.Status):
			out.History = append(out.History, item)
		}
	}
	out.Funnel = s.buildFunnel(ctx, lr, sessionIDs, settledCount, currentSess)
	return out, nil
}

func (s *LiveRoomService) buildFunnel(
	ctx context.Context,
	lr *domain.LiveRoom,
	sessionIDs []uint64,
	settledCount int,
	current *domain.AuctionSession,
) LiveRoomFunnel {
	f := LiveRoomFunnel{SettledCount: settledCount}
	if s.viewers != nil {
		f.ViewerCount = s.viewers.ClientCount(lr.RoomID)
	}
	if current != nil {
		f.BidderCount = int(current.ParticipantCount)
	}
	if s.orders != nil && len(sessionIDs) > 0 {
		if paid, err := s.orders.CountPaidBySessionIDs(ctx, sessionIDs); err == nil {
			f.PaidCount = paid
		}
	}
	f.Hint = funnelHint(f)
	return f
}

func funnelHint(f LiveRoomFunnel) string {
	if f.ViewerCount < 3 {
		return "进房偏少，优先拉流量（分享链接、列表曝光）"
	}
	if f.ViewerCount >= 3 && f.BidderCount == 0 {
		return "有人进房但无人出价，检查起拍价 / 加价幅度是否过高"
	}
	if f.BidderCount > 0 && f.SettledCount == 0 && f.ViewerCount > 0 {
		return "有出价但未成交，关注倒计时与封顶规则是否过严"
	}
	if f.SettledCount > 0 && f.PaidCount < f.SettledCount {
		return "已成交但支付未完成，提醒胜者尽快支付或缩短待支付时限"
	}
	if f.SettledCount > 0 && f.PaidCount >= f.SettledCount {
		return "转化链路健康，可继续上架下一品"
	}
	return ""
}

func (s *LiveRoomService) buildUserSessionDetail(ctx context.Context, sess *domain.AuctionSession, p *domain.Product) (*UserAuctionDetail, error) {
	if !isPublicSession(sess.Status) {
		return nil, domain.ErrAuctionNotVisible
	}
	return &UserAuctionDetail{
		Session:  *sess,
		Product:  p,
		Snapshot: BuildSnapshot(sess, time.Now()),
	}, nil
}

func isHistoryStatus(st domain.SessionStatus) bool {
	switch st {
	case domain.SessionStatusSettled, domain.SessionStatusCancelled, domain.SessionStatusFailed:
		return true
	default:
		return false
	}
}

func ptrUint64(p *uint64) uint64 {
	if p == nil {
		return 0
	}
	return *p
}

func (s *LiveRoomService) getUserDetailLegacy(ctx context.Context, roomID string) (*UserLiveRoomDetail, error) {
	sess, err := s.sessions.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	p, err := s.products.GetByID(ctx, sess.ProductID)
	if err != nil {
		return nil, err
	}
	current, err := s.buildUserSessionDetail(ctx, sess, p)
	if err != nil {
		return nil, err
	}
	out := &UserLiveRoomDetail{
		RoomID: roomID,
		LiveRoom: domain.LiveRoom{
			RoomID: roomID,
			Status: domain.LiveRoomStatusLive,
		},
		Current: current,
		History: []SessionSummary{},
	}
	out.Anchor = s.buildAnchorBrief(ctx, sess.AnchorID)
	return out, nil
}
