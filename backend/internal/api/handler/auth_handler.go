package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type registerBody struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
	Role     string `json:"role"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var body registerBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidPhone)
		return
	}
	result, err := h.svc.Register(c.Request.Context(), service.RegisterInput{
		Phone:    body.Phone,
		Password: body.Password,
		Nickname: body.Nickname,
		Role:     domain.UserRole(body.Role),
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, result)
}

type loginBody struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var body loginBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidCredentials)
		return
	}
	result, err := h.svc.Login(c.Request.Context(), body.Phone, body.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	u := middleware.CurrentUser(c)
	if u == nil {
		response.Fail(c, domain.ErrUnauthorized)
		return
	}
	fresh, err := h.svc.Me(c.Request.Context(), u.ID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, fresh)
}
