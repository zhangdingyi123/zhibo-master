package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zhibo/backend/internal/config"
	"github.com/zhibo/backend/internal/domain"
)

type TTSService struct {
	cfg    config.Config
	client *http.Client
}

func NewTTSService(cfg config.Config) *TTSService {
	return &TTSService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type ttsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

func (s *TTSService) Synthesize(ctx context.Context, text string) ([]byte, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrTTSInputRequired
	}
	if s.cfg.AIAPIKey == "" {
		return nil, domain.ErrTTSServiceUnavailable
	}
	if utf8.RuneCountInString(text) > 200 {
		text = string([]rune(text)[:200])
	}

	base := strings.TrimRight(s.cfg.AIAPIBase, "/")
	reqBody, err := json.Marshal(ttsRequest{
		Model: s.cfg.AITTSModel,
		Input: text,
		Voice: s.cfg.AITTSVoice,
	})
	if err != nil {
		return nil, domain.ErrTTSSynthesisFailed
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/audio/speech", bytes.NewReader(reqBody))
	if err != nil {
		return nil, domain.ErrTTSSynthesisFailed
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.AIAPIKey)

	res, err := s.client.Do(req)
	if err != nil {
		return nil, domain.ErrTTSServiceUnavailable
	}
	defer res.Body.Close()

	audio, err := io.ReadAll(io.LimitReader(res.Body, 5<<20))
	if err != nil {
		return nil, domain.ErrTTSSynthesisFailed
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", domain.ErrTTSSynthesisFailed, res.StatusCode)
	}
	if len(audio) == 0 {
		return nil, domain.ErrTTSSynthesisFailed
	}
	return audio, nil
}
