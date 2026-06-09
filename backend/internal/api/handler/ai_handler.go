package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type AIHandler struct {
	ai  *service.AIService
	tts *service.TTSService
}

func NewAIHandler(ai *service.AIService, tts *service.TTSService) *AIHandler {
	return &AIHandler{ai: ai, tts: tts}
}

type generateIntroBody struct {
	Name     string `json:"name" binding:"required"`
	Keywords string `json:"keywords"`
}

func (h *AIHandler) GenerateProductIntro(c *gin.Context) {
	var body generateIntroBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidProductName)
		return
	}
	result, err := h.ai.GenerateProductIntro(c.Request.Context(), service.GenerateProductIntroInput{
		Name:     body.Name,
		Keywords: body.Keywords,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

type ttsBody struct {
	Text string `json:"text" binding:"required"`
}

func (h *AIHandler) SynthesizeSpeech(c *gin.Context) {
	var body ttsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrTTSInputRequired)
		return
	}
	audio, err := h.tts.Synthesize(c.Request.Context(), body.Text)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Data(http.StatusOK, "audio/mpeg", audio)
}
