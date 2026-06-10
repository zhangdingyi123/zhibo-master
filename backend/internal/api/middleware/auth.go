package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/auth"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

const (
	ctxUserKey   = "authUser"
	headerOpenID = "X-Mock-Open-Id"
	headerUserID = "X-User-Id"
)

// RequireAuth 必须登录：Bearer JWT 或开发 Mock 头
func RequireAuth(users *repository.UserRepo, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, err := resolveUser(c, users, jwtSecret, true)
		if err != nil {
			response.Fail(c, err)
			c.Abort()
			return
		}
		c.Set(ctxUserKey, u)
		c.Next()
	}
}

// OptionalAuth 可选登录（用于 WS 等场景由调用方处理匿名）
func OptionalAuth(users *repository.UserRepo, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, err := resolveUser(c, users, jwtSecret, false)
		if err != nil {
			response.Fail(c, err)
			c.Abort()
			return
		}
		if u != nil {
			c.Set(ctxUserKey, u)
		}
		c.Next()
	}
}

// MockAuth 兼容旧名
func MockAuth(users *repository.UserRepo, jwtSecret string) gin.HandlerFunc {
	return RequireAuth(users, jwtSecret)
}

func resolveUser(c *gin.Context, users *repository.UserRepo, jwtSecret string, required bool) (*domain.User, error) {
	if token := bearerToken(c.GetHeader("Authorization")); token != "" {
		claims, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			if required {
				return nil, domain.ErrUnauthorized
			}
			return nil, nil
		}
		u, err := users.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
		return u, nil
	}

	openID := c.GetHeader(headerOpenID)
	userIDHeader := c.GetHeader(headerUserID)

	switch {
	case openID != "":
		u, err := users.GetByOpenID(c.Request.Context(), openID)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
		return u, nil
	case userIDHeader != "":
		id, parseErr := strconv.ParseUint(userIDHeader, 10, 64)
		if parseErr != nil {
			return nil, domain.ErrUnauthorized
		}
		u, err := users.GetByID(c.Request.Context(), id)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
		return u, nil
	default:
		if required {
			return nil, domain.ErrUnauthorized
		}
		return nil, nil
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

// RequireAnchor 要求主播或管理员角色
func RequireAnchor() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := c.MustGet(ctxUserKey).(*domain.User)
		if !ok || u == nil {
			response.Fail(c, domain.ErrUnauthorized)
			c.Abort()
			return
		}
		if u.Role != domain.UserRoleAnchor && u.Role != domain.UserRoleAdmin {
			response.Fail(c, domain.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// CurrentUser 从上下文取当前用户
func CurrentUser(c *gin.Context) *domain.User {
	u, _ := c.MustGet(ctxUserKey).(*domain.User)
	return u
}

// TryCurrentUser 可选登录场景下尝试取当前用户
func TryCurrentUser(c *gin.Context) (*domain.User, bool) {
	u, ok := c.Get(ctxUserKey)
	if !ok {
		return nil, false
	}
	user, ok := u.(*domain.User)
	return user, ok && user != nil
}
