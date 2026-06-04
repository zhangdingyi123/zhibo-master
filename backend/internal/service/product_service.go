package service

import (
	"context"
	"strings"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

type ProductService struct {
	products *repository.ProductRepo
	sessions *repository.SessionRepo
	orders   *repository.OrderRepo
}

func NewProductService(products *repository.ProductRepo, sessions *repository.SessionRepo, orders *repository.OrderRepo) *ProductService {
	return &ProductService{products: products, sessions: sessions, orders: orders}
}

type CreateProductInput struct {
	Name        string
	Description string
	CoverURL    string
	Images      []string
}

type UpdateProductInput struct {
	Name        string
	Description string
	CoverURL    string
	Images      []string
}

type ListProductsInput struct {
	Status   *domain.ProductStatus
	Page     int
	PageSize int
}

type ListProductsResult struct {
	Items    []ProductView `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

func (s *ProductService) Create(ctx context.Context, anchorID uint64, in CreateProductInput) (*ProductView, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrInvalidProductName
	}
	p := &domain.Product{
		AnchorID:    anchorID,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		CoverURL:    strings.TrimSpace(in.CoverURL),
		Images:      in.Images,
		Status:      domain.ProductStatusDraft,
	}
	if p.Images == nil {
		p.Images = []string{}
	}
	if err := s.products.Create(ctx, p); err != nil {
		return nil, err
	}
	created, err := s.products.GetByID(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	return &ProductView{Product: *created}, nil
}

func (s *ProductService) Get(ctx context.Context, anchorID, productID uint64) (*ProductView, error) {
	p, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	views, err := s.buildViews(ctx, []domain.Product{*p})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *ProductService) List(ctx context.Context, anchorID uint64, in ListProductsInput) (*ListProductsResult, error) {
	items, total, err := s.products.List(ctx, repository.ProductFilter{
		AnchorID: anchorID,
		Status:   in.Status,
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
	if items == nil {
		items = []domain.Product{}
	}
	views, err := s.buildViews(ctx, items)
	if err != nil {
		return nil, err
	}
	return &ListProductsResult{
		Items:    views,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *ProductService) buildViews(ctx context.Context, products []domain.Product) ([]ProductView, error) {
	if len(products) == 0 {
		return []ProductView{}, nil
	}
	ids := make([]uint64, len(products))
	for i, p := range products {
		ids[i] = p.ID
	}

	activeMap, err := s.sessions.MapActiveByProductIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	latestMap, err := s.sessions.MapLatestByProductIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	orderMap, err := s.orders.MapLatestByProductIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	views := make([]ProductView, len(products))
	for i, p := range products {
		active := activeMap[p.ID]
		latest := latestMap[p.ID]
		sess := pickSessionForView(active, latest)
		var order *domain.Order
		if sess != nil && sess.Status == domain.SessionStatusSettled {
			order = orderMap[p.ID]
		}
		views[i] = ProductView{
			Product: p,
			Auction: sessionToProgress(sess, order),
		}
	}
	return views, nil
}

func (s *ProductService) Update(ctx context.Context, anchorID, productID uint64, in UpdateProductInput) (*ProductView, error) {
	p, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.AnchorID != anchorID {
		return nil, domain.ErrForbidden
	}
	if p.Status == domain.ProductStatusAuctioning || p.Status == domain.ProductStatusSold {
		return nil, domain.ErrProductNotEditable
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrInvalidProductName
	}
	p.Name = name
	p.Description = strings.TrimSpace(in.Description)
	p.CoverURL = strings.TrimSpace(in.CoverURL)
	if in.Images != nil {
		p.Images = in.Images
	}
	if err := s.products.Update(ctx, p); err != nil {
		return nil, err
	}
	return s.Get(ctx, anchorID, productID)
}

func (s *ProductService) Delete(ctx context.Context, anchorID, productID uint64) error {
	p, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if p.AnchorID != anchorID {
		return domain.ErrForbidden
	}
	switch p.Status {
	case domain.ProductStatusAuctioning, domain.ProductStatusSold:
		return domain.ErrProductNotDeletable
	case domain.ProductStatusDraft:
		return s.products.Delete(ctx, productID, anchorID)
	default:
		active, err := s.sessions.HasActiveByProductID(ctx, productID)
		if err != nil {
			return err
		}
		if active {
			return domain.ErrProductNotDeletable
		}
		return s.products.UpdateStatus(ctx, productID, anchorID, domain.ProductStatusOffShelf)
	}
}
