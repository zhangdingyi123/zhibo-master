package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/domain"
)

func currentUser(c *gin.Context) *domain.User {
	return middleware.CurrentUser(c)
}
