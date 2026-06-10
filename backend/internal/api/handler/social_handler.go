package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type SocialHandler struct {
	social *service.SocialService
}

func NewSocialHandler(social *service.SocialService) *SocialHandler {
	return &SocialHandler{social: social}
}

func (h *SocialHandler) GetRoomStats(c *gin.Context) {
	roomID := c.Param("roomId")
	var viewerID *uint64
	if u, ok := middleware.TryCurrentUser(c); ok {
		viewerID = &u.ID
	}
	var productID *uint64
	if pid := c.Query("productId"); pid != "" {
		if id, err := parseID(pid); err == nil {
			productID = &id
		}
	}
	stats, err := h.social.GetRoomStats(c.Request.Context(), roomID, viewerID, productID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, stats)
}

func (h *SocialHandler) ListComments(c *gin.Context) {
	roomID := c.Param("roomId")
	items, err := h.social.ListComments(c.Request.Context(), roomID, false)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *SocialHandler) PostComment(c *gin.Context) {
	roomID := c.Param("roomId")
	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrCommentEmpty)
		return
	}
	user := middleware.CurrentUser(c)
	comment, err := h.social.PostComment(c.Request.Context(), user.ID, roomID, body.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, comment)
}

func (h *SocialHandler) Like(c *gin.Context) {
	roomID := c.Param("roomId")
	user := middleware.CurrentUser(c)
	count, err := h.social.Like(c.Request.Context(), user.ID, roomID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"likeCount": count})
}

func (h *SocialHandler) ToggleFavorite(c *gin.Context) {
	productID, err := parseID(c.Param("productId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	user := middleware.CurrentUser(c)
	favorited, err := h.social.ToggleFavorite(c.Request.Context(), user.ID, productID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"favorited": favorited})
}

func (h *SocialHandler) ToggleFollow(c *gin.Context) {
	anchorID, err := parseID(c.Param("anchorId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	user := middleware.CurrentUser(c)
	following, err := h.social.ToggleFollow(c.Request.Context(), user.ID, anchorID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"following": following})
}

func (h *SocialHandler) AdminListComments(c *gin.Context) {
	roomID := c.Param("roomId")
	items, err := h.social.ListComments(c.Request.Context(), roomID, true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *SocialHandler) HideComment(c *gin.Context) {
	commentID, err := parseID(c.Param("commentId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	user := middleware.CurrentUser(c)
	if err := h.social.HideComment(c.Request.Context(), user.ID, commentID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
