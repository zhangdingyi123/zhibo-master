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

// Prometheus 输出 Prometheus 文本格式（Grafana 数据源）
func (h *MetricsHandler) Prometheus(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(metrics.PrometheusText(h.ws)))
}
