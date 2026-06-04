package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type UserOrderHandler struct {
	orders *service.OrderService
}

func NewUserOrderHandler(orders *service.OrderService) *UserOrderHandler {
	return &UserOrderHandler{orders: orders}
}

func (h *UserOrderHandler) List(c *gin.Context) {
	user := currentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var status *domain.OrderStatus
	if s := c.Query("status"); s != "" {
		st := domain.OrderStatus(s)
		status = &st
	}

	result, err := h.orders.ListForBuyer(c.Request.Context(), user.ID, service.ListOrdersInput{
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

func (h *UserOrderHandler) Get(c *gin.Context) {
	user := currentUser(c)
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	o, err := h.orders.GetForBuyer(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}

func (h *UserOrderHandler) GetBySession(c *gin.Context) {
	user := currentUser(c)
	sessionID, err := parseID(c.Param("sessionId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	o, err := h.orders.GetBySessionForBuyer(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}

func (h *UserOrderHandler) MockPay(c *gin.Context) {
	user := currentUser(c)
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	o, err := h.orders.MockPay(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, o)
}
