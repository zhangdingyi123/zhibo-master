package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type MessageHandler struct {
	messages *service.MessageService
}

func NewMessageHandler(messages *service.MessageService) *MessageHandler {
	return &MessageHandler{messages: messages}
}

func (h *MessageHandler) List(c *gin.Context) {
	user := currentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	unreadOnly := c.Query("unread") == "1"

	result, err := h.messages.List(c.Request.Context(), user.ID, service.ListMessagesInput{
		UnreadOnly: unreadOnly,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *MessageHandler) UnreadCount(c *gin.Context) {
	user := currentUser(c)
	n, err := h.messages.UnreadCount(c.Request.Context(), user.ID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"count": n})
}

func (h *MessageHandler) MarkRead(c *gin.Context) {
	user := currentUser(c)
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	if err := h.messages.MarkRead(c.Request.Context(), user.ID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *MessageHandler) MarkAllRead(c *gin.Context) {
	user := currentUser(c)
	n, err := h.messages.MarkAllRead(c.Request.Context(), user.ID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"updated": n})
}
