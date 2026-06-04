package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) List(c *gin.Context) {
	user := middleware.CurrentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var status *domain.OrderStatus
	if s := c.Query("status"); s != "" {
		st := domain.OrderStatus(s)
		status = &st
	}

	result, err := h.svc.List(c.Request.Context(), user.ID, service.ListOrdersInput{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *OrderHandler) Get(c *gin.Context) {
	user := middleware.CurrentUser(c)
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	o, err := h.svc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}
