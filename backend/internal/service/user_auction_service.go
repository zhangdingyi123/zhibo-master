package service

import (
	"context"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

type UserAuctionService struct {
	sessions *repository.SessionRepo
	products *repository.ProductRepo
	cache    RoomCache
}

func NewUserAuctionService(sessions *repository.SessionRepo, products *repository.ProductRepo) *UserAuctionService {
	return &UserAuctionService{sessions: sessions, products: products}
}

// SetRoomCache 注入 Redis 热数据缓存（5.1）
func (s *UserAuctionService) SetRoomCache(c RoomCache) {
	s.cache = c
}

// ProductBrief 用户端商品摘要
type ProductBrief struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CoverURL    string `json:"coverUrl"`
}

// UserAuctionListItem 竞拍列表项
type UserAuctionListItem struct {
	Session domain.AuctionSession `json:"session"`
	Product ProductBrief          `json:"product"`
}

type ListUserAuctionsInput struct {
	Status   *domain.SessionStatus
	Page     int
	PageSize int
}

type ListUserAuctionsResult struct {
	Items    []UserAuctionListItem `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

// UserAuctionDetail 竞拍详情
type UserAuctionDetail struct {
	Session  domain.AuctionSession `json:"session"`
	Product  *domain.Product       `json:"product"`
	Snapshot *SessionSnapshot      `json:"snapshot"`
}

func (s *UserAuctionService) List(ctx context.Context, in ListUserAuctionsInput) (*ListUserAuctionsResult, error) {
	var statuses []domain.SessionStatus
	if in.Status != nil {
		statuses = []domain.SessionStatus{*in.Status}
	}
	rows, total, err := s.sessions.ListPublic(ctx, repository.PublicSessionFilter{
		Statuses: statuses,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
	if err != nil {
		return nil, err
	}
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	items := make([]UserAuctionListItem, len(rows))
	for i, row := range rows {
		items[i] = UserAuctionListItem{
			Session: row.Session,
			Product: ProductBrief{
				ID:          row.Session.ProductID,
				Name:        row.ProductName,
				Description: row.ProductDescription,
				CoverURL:    row.ProductCoverURL,
			},
		}
	}
	return &ListUserAuctionsResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *UserAuctionService) GetByID(ctx context.Context, sessionID uint64) (*UserAuctionDetail, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, session)
}

func (s *UserAuctionService) GetByRoomID(ctx context.Context, roomID string) (*UserAuctionDetail, error) {
	session, err := s.sessions.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, session)
}

func (s *UserAuctionService) Snapshot(ctx context.Context, sessionID uint64) (*SessionSnapshot, error) {
	if s.cache != nil {
		snap, err := s.cache.GetSnapshotBySession(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if snap != nil {
			if !isPublicSession(snap.Status) {
				return nil, domain.ErrAuctionNotVisible
			}
			return snap, nil
		}
	}
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !isPublicSession(session.Status) {
		return nil, domain.ErrAuctionNotVisible
	}
	return s.snapshotFromSession(ctx, session)
}

func (s *UserAuctionService) SnapshotByRoom(ctx context.Context, roomID string) (*SessionSnapshot, error) {
	if s.cache != nil {
		snap, err := s.cache.GetSnapshotByRoom(ctx, roomID)
		if err != nil {
			return nil, err
		}
		if snap != nil {
			if !isPublicSession(snap.Status) {
				return nil, domain.ErrAuctionNotVisible
			}
			return snap, nil
		}
	}
	session, err := s.sessions.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if !isPublicSession(session.Status) {
		return nil, domain.ErrAuctionNotVisible
	}
	return s.snapshotFromSession(ctx, session)
}

func (s *UserAuctionService) snapshotFromSession(ctx context.Context, session *domain.AuctionSession) (*SessionSnapshot, error) {
	snap := BuildSnapshot(session, time.Now())
	if s.cache != nil {
		_ = s.cache.RefreshFromSession(ctx, session)
	}
	return snap, nil
}

func (s *UserAuctionService) buildDetail(ctx context.Context, session *domain.AuctionSession) (*UserAuctionDetail, error) {
	if !isPublicSession(session.Status) {
		return nil, domain.ErrAuctionNotVisible
	}
	p, err := s.products.GetByID(ctx, session.ProductID)
	if err != nil {
		return nil, err
	}
	switch p.Status {
	case domain.ProductStatusDraft, domain.ProductStatusOffShelf:
		return nil, domain.ErrAuctionNotVisible
	}
	return &UserAuctionDetail{
		Session:  *session,
		Product:  p,
		Snapshot: BuildSnapshot(session, time.Now()),
	}, nil
}

func isPublicSession(status domain.SessionStatus) bool {
	switch status {
	case domain.SessionStatusPending, domain.SessionStatusRunning, domain.SessionStatusSettled:
		return true
	default:
		return false
	}
}
