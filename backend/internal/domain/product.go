package domain

import "time"

// ProductStatus 商品生命周期状态
type ProductStatus string

const (
	ProductStatusDraft      ProductStatus = "draft"
	ProductStatusListed     ProductStatus = "listed"
	ProductStatusAuctioning ProductStatus = "auctioning"
	ProductStatusSold       ProductStatus = "sold"
	ProductStatusOffShelf   ProductStatus = "off_shelf"
)

// Product 商品实体
type Product struct {
	ID          uint64        `json:"id"`
	AnchorID    uint64        `json:"anchorId"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	CoverURL    string        `json:"coverUrl"`
	Images      []string      `json:"images,omitempty"`
	Status      ProductStatus `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}
