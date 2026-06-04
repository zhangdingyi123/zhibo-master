package ws

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zhibo/backend/internal/auth"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

// resolveUser 从 token / Query / Header 解析用户（4.2）
func resolveUser(r *http.Request, users *repository.UserRepo, jwtSecret string) (*domain.User, error) {
	ctx := r.Context()

	token := r.URL.Query().Get("token")
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	if token != "" {
		claims, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
		return users.GetByID(ctx, claims.UserID)
	}

	openID := r.URL.Query().Get("openId")
	if openID == "" {
		openID = r.Header.Get("X-Mock-Open-Id")
	}
	userIDStr := r.URL.Query().Get("userId")
	if userIDStr == "" {
		userIDStr = r.Header.Get("X-User-Id")
	}

	switch {
	case openID != "":
		return users.GetByOpenID(ctx, openID)
	case userIDStr != "":
		id, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
		return users.GetByID(ctx, id)
	default:
		return nil, nil // 匿名围观
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
