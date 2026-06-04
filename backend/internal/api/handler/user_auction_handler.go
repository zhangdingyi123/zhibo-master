package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/infra/metrics"
	"github.com/zhibo/backend/internal/service"
)

type UserAuctionHandler struct {
	auctions *service.UserAuctionService
	bids     *service.BidService
}

func NewUserAuctionHandler(auctions *service.UserAuctionService, bids *service.BidService) *UserAuctionHandler {
	return &UserAuctionHandler{auctions: auctions, bids: bids}
}

func (h *UserAuctionHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var status *domain.SessionStatus
	if s := c.Query("status"); s != "" {
		st := domain.SessionStatus(s)
		status = &st
	}

	result, err := h.auctions.List(c.Request.Context(), service.ListUserAuctionsInput{
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

func (h *UserAuctionHandler) Get(c *gin.Context) {
	sessionID, err := parseID(c.Param("sessionId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	detail, err := h.auctions.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *UserAuctionHandler) Snapshot(c *gin.Context) {
	sessionID, err := parseID(c.Param("sessionId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	snap, err := h.auctions.Snapshot(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, snap)
}

func (h *UserAuctionHandler) SnapshotByRoom(c *gin.Context) {
	roomID := c.Param("roomId")
	snap, err := h.auctions.SnapshotByRoom(c.Request.Context(), roomID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, snap)
}

type placeBidBody struct {
	Amount    int64  `json:"amount" binding:"required"`
	RequestID string `json:"requestId" binding:"required"`
}

func (h *UserAuctionHandler) PlaceBid(c *gin.Context) {
	sessionID, err := parseID(c.Param("sessionId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}

	var body placeBidBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrRequestIDRequired)
		return
	}

	user := currentUser(c)
	metrics.RecordBidAttempt()
	result, err := h.bids.PlaceBid(c.Request.Context(), user.ID, sessionID, service.PlaceBidInput{
		Amount:    body.Amount,
		RequestID: body.RequestID,
	})
	if err != nil {
		metrics.RecordBidFailure()
		response.Fail(c, err)
		return
	}
	metrics.RecordBidSuccess()
	response.OK(c, result)
}
