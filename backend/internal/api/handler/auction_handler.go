package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type AuctionHandler struct {
	svc *service.AuctionService
}

func NewAuctionHandler(svc *service.AuctionService) *AuctionHandler {
	return &AuctionHandler{svc: svc}
}

type publishAuctionBody struct {
	StartingPrice      int64  `json:"startingPrice"`
	BidIncrement       int64  `json:"bidIncrement" binding:"required"`
	CapPrice           *int64 `json:"capPrice"`
	DurationSec        uint32 `json:"durationSec" binding:"required"`
	ExtendThresholdSec uint32 `json:"extendThresholdSec"`
	ExtendSec          uint32 `json:"extendSec"`
	ScheduledStartAt   string `json:"scheduledStartAt"`
}

func (h *AuctionHandler) Publish(c *gin.Context) {
	productID, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}

	var body publishAuctionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidBidIncrement)
		return
	}

	rules, scheduled, err := parseAuctionRulesBody(body)
	if err != nil {
		response.Fail(c, err)
		return
	}

	user := middleware.CurrentUser(c)
	session, err := h.svc.Publish(c.Request.Context(), user.ID, productID, service.PublishAuctionInput{
		Rules:            rules,
		ScheduledStartAt: scheduled,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, session)
}

func (h *AuctionHandler) Get(c *gin.Context) {
	user := middleware.CurrentUser(c)
	sessionID, err := parseID(c.Param("sessionId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	session, err := h.svc.Get(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, session)
}

func (h *AuctionHandler) UpdateRules(c *gin.Context) {
	sessionID, err := parseID(c.Param("sessionId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}

	var body publishAuctionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidBidIncrement)
		return
	}

	rules, scheduled, err := parseAuctionRulesBody(body)
	if err != nil {
		response.Fail(c, err)
		return
	}

	user := middleware.CurrentUser(c)
	session, err := h.svc.UpdateRules(c.Request.Context(), user.ID, sessionID, service.UpdateRulesInput{
		Rules:            rules,
		ScheduledStartAt: scheduled,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, session)
}

type cancelAuctionBody struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *AuctionHandler) Cancel(c *gin.Context) {
	sessionID, err := parseID(c.Param("sessionId"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}

	var body cancelAuctionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrCancelReasonRequired)
		return
	}

	user := middleware.CurrentUser(c)
	session, err := h.svc.Cancel(c.Request.Context(), user.ID, sessionID, body.Reason)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, session)
}

func parseAuctionRulesBody(body publishAuctionBody) (domain.AuctionRules, *time.Time, error) {
	rules := domain.AuctionRules{
		StartingPrice:      body.StartingPrice,
		BidIncrement:       body.BidIncrement,
		CapPrice:           body.CapPrice,
		DurationSec:        body.DurationSec,
		ExtendThresholdSec: body.ExtendThresholdSec,
		ExtendSec:          body.ExtendSec,
	}
	if rules.ExtendThresholdSec == 0 {
		rules.ExtendThresholdSec = 10
	}
	if rules.ExtendSec == 0 {
		rules.ExtendSec = 30
	}

	var scheduled *time.Time
	if body.ScheduledStartAt != "" {
		t, err := time.Parse(time.RFC3339, body.ScheduledStartAt)
		if err != nil {
			return rules, nil, domain.ErrInvalidDuration
		}
		scheduled = &t
	}
	return rules, scheduled, nil
}
