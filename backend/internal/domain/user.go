package domain

import "time"

// UserRole 用户角色
type UserRole string

const (
	UserRoleBuyer UserRole = "buyer"
	UserRoleAnchor UserRole = "anchor"
	UserRoleAdmin UserRole = "admin"
)

// User 用户实体
type User struct {
	ID        uint64    `json:"id"`
	OpenID    string    `json:"openId"`
	Phone     string    `json:"phone,omitempty"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
