package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type LiveRoomHandler struct {
	svc *service.LiveRoomService
}

func NewLiveRoomHandler(svc *service.LiveRoomService) *LiveRoomHandler {
	return &LiveRoomHandler{svc: svc}
}

type createLiveRoomBody struct {
	Title string `json:"title" binding:"required"`
}

func (h *LiveRoomHandler) Create(c *gin.Context) {
	var body createLiveRoomBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrLiveRoomTitleRequired)
		return
	}
	user := middleware.CurrentUser(c)
	lr, err := h.svc.Create(c.Request.Context(), user.ID, service.CreateLiveRoomInput{Title: body.Title})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, lr)
}

func (h *LiveRoomHandler) List(c *gin.Context) {
	user := middleware.CurrentUser(c)
	items, err := h.svc.List(c.Request.Context(), user.ID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *LiveRoomHandler) Get(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	user := middleware.CurrentUser(c)
	detail, err := h.svc.GetAdminDetail(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *LiveRoomHandler) Start(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	user := middleware.CurrentUser(c)
	lr, err := h.svc.Start(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, lr)
}

func (h *LiveRoomHandler) End(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	user := middleware.CurrentUser(c)
	lr, err := h.svc.End(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, lr)
}

func (h *LiveRoomHandler) AddSession(c *gin.Context) {
	liveRoomID, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}

	var body struct {
		ProductID          uint64 `json:"productId" binding:"required"`
		StartingPrice      int64  `json:"startingPrice"`
		BidIncrement       int64  `json:"bidIncrement" binding:"required"`
		CapPrice           *int64 `json:"capPrice"`
		DurationSec        uint32 `json:"durationSec" binding:"required"`
		ExtendThresholdSec uint32 `json:"extendThresholdSec"`
		ExtendSec          uint32 `json:"extendSec"`
		ScheduledStartAt   string `json:"scheduledStartAt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidBidIncrement)
		return
	}

	rules, scheduled, err := parseAuctionRulesBody(publishAuctionBody{
		StartingPrice:      body.StartingPrice,
		BidIncrement:       body.BidIncrement,
		CapPrice:           body.CapPrice,
		DurationSec:        body.DurationSec,
		ExtendThresholdSec: body.ExtendThresholdSec,
		ExtendSec:          body.ExtendSec,
		ScheduledStartAt:   body.ScheduledStartAt,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	user := middleware.CurrentUser(c)
	session, err := h.svc.AddSession(c.Request.Context(), user.ID, liveRoomID, service.AddSessionToLiveRoomInput{
		ProductID:        body.ProductID,
		Rules:            rules,
		ScheduledStartAt: scheduled,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, session)
}

func (h *LiveRoomHandler) EndCurrentAndSwitch(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	user := middleware.CurrentUser(c)
	detail, err := h.svc.EndCurrentAndSwitch(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *LiveRoomHandler) GetByRoom(c *gin.Context) {
	roomID := c.Param("roomId")
	if roomID == "" {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	detail, err := h.svc.GetUserDetail(c.Request.Context(), roomID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}
