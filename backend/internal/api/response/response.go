package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/domain"
)

type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, APIResponse{Code: 0, Message: "ok", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, APIResponse{Code: 0, Message: "ok", Data: data})
}

func Fail(c *gin.Context, err error) {
	status, code, msg := mapError(err)
	c.JSON(status, APIResponse{Code: code, Message: msg})
}

func mapError(err error) (httpStatus, code int, message string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, 40400, err.Error()
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, 40300, err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, 40100, err.Error()
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, 40101, err.Error()
	case errors.Is(err, domain.ErrPhoneAlreadyExists):
		return http.StatusConflict, 40902, err.Error()
	case errors.Is(err, domain.ErrInvalidPhone),
		errors.Is(err, domain.ErrWeakPassword),
		errors.Is(err, domain.ErrInvalidNickname),
		errors.Is(err, domain.ErrRoleNotAllowed):
		return http.StatusBadRequest, 40000, err.Error()
	case errors.Is(err, domain.ErrInvalidProductName),
		errors.Is(err, domain.ErrInvalidStartingPrice),
		errors.Is(err, domain.ErrInvalidBidIncrement),
		errors.Is(err, domain.ErrCapBelowStarting),
		errors.Is(err, domain.ErrInvalidDuration),
		errors.Is(err, domain.ErrInvalidExtendThreshold),
		errors.Is(err, domain.ErrInvalidExtendSec):
		return http.StatusBadRequest, 40000, err.Error()
	case errors.Is(err, domain.ErrProductNotEditable),
		errors.Is(err, domain.ErrProductNotDeletable),
		errors.Is(err, domain.ErrProductNotPublishable),
		errors.Is(err, domain.ErrActiveSessionExists),
		errors.Is(err, domain.ErrRulesNotEditable),
		errors.Is(err, domain.ErrSessionHasBids),
		errors.Is(err, domain.ErrSessionNotCancellable),
		errors.Is(err, domain.ErrInvalidStateTransition),
		errors.Is(err, domain.ErrOrderAlreadyExists):
		return http.StatusConflict, 40900, err.Error()
	case errors.Is(err, domain.ErrCancelReasonRequired),
		errors.Is(err, domain.ErrSettlementNoWinner),
		errors.Is(err, domain.ErrRequestIDRequired):
		return http.StatusBadRequest, 40000, err.Error()
	case errors.Is(err, domain.ErrBidTooLow),
		errors.Is(err, domain.ErrBidExceedsCap),
		errors.Is(err, domain.ErrAuctionEnded):
		return http.StatusBadRequest, 40001, err.Error()
	case errors.Is(err, domain.ErrSessionNotBiddable),
		errors.Is(err, domain.ErrAuctionNotVisible):
		return http.StatusConflict, 40900, err.Error()
	case errors.Is(err, domain.ErrVersionConflict),
		errors.Is(err, domain.ErrSessionLockBusy):
		return http.StatusConflict, 40901, err.Error()
	default:
		return http.StatusInternalServerError, 50000, "服务器内部错误"
	}
}
