package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/infra/metrics"
)

type MetricsHandler struct {
	ws metrics.WSCollector
}

func NewMetricsHandler(ws metrics.WSCollector) *MetricsHandler {
	return &MetricsHandler{ws: ws}
}

func (h *MetricsHandler) Get(c *gin.Context) {
	snap := metrics.Collect(h.ws)
	c.JSON(http.StatusOK, snap)
}
