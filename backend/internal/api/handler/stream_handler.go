package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/config"
)

type StreamHandler struct {
	cfg config.Config
}

func NewStreamHandler(cfg config.Config) *StreamHandler {
	return &StreamHandler{cfg: cfg}
}

func (h *StreamHandler) GetByRoom(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomId required"})
		return
	}

	streamKey := roomID
	pushURL := "rtmp://" + h.cfg.StreamRTMPHost + "/live/" + streamKey
	hlsBase := strings.TrimRight(h.cfg.StreamHLSBase, "/")

	c.JSON(http.StatusOK, gin.H{
		"roomId":  roomID,
		"pushUrl": pushURL,
		"hlsUrl":  hlsBase + "/" + streamKey + ".m3u8",
		"flvUrl":  hlsBase + "/" + streamKey + ".flv",
	})
}
