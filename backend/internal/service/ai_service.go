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

	"github.com/zhibo/backend/internal/config"
	"github.com/zhibo/backend/internal/domain"
)

type AIService struct {
	cfg    config.Config
	client *http.Client
}

func NewAIService(cfg config.Config) *AIService {
	return &AIService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

type GenerateProductIntroInput struct {
	Name     string
	Keywords string
}

type GenerateProductIntroResult struct {
	Description string `json:"description"`
	Source      string `json:"source"` // "llm" | "template"
}

func (s *AIService) GenerateProductIntro(ctx context.Context, in GenerateProductIntroInput) (*GenerateProductIntroResult, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrInvalidProductName
	}
	keywords := strings.TrimSpace(in.Keywords)

	if s.cfg.AIAPIKey == "" {
		return &GenerateProductIntroResult{
			Description: templateProductIntro(name, keywords),
			Source:      "template",
		}, nil
	}

	text, err := s.callChat(ctx, buildIntroPrompt(name, keywords))
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrAIGenerationFailed
	}
	return &GenerateProductIntroResult{
		Description: text,
		Source:      "llm",
	}, nil
}

func buildIntroPrompt(name, keywords string) string {
	kw := keywords
	if kw == "" {
		kw = "品质优良、直播间专属"
	}
	return fmt.Sprintf(`你是抖音电商直播带货文案助手。根据商品信息生成直播口播介绍。

商品名称：%s
补充关键词：%s

要求：
1. 150–220 字，口语化、有直播感
2. 自然融入 2–3 个卖点
3. 结尾有一句催拍 / 限时竞拍话术
4. 只输出正文，不要标题、编号或 Markdown`, name, kw)
}

func templateProductIntro(name, keywords string) string {
	kw := keywords
	if kw == "" {
		kw = "品质过硬、性价比高"
	}
	return fmt.Sprintf(
		"家人们看过来！【%s】直播间专属好物来了！%s，主播亲自把关，细节经得起对比。现在限时竞拍，价格由你说了算，库存不多，喜欢的赶紧出手，手慢无！",
		name, kw,
	)
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type apiErrorBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *AIService) callChat(ctx context.Context, prompt string) (string, error) {
	base := strings.TrimRight(s.cfg.AIAPIBase, "/")
	reqBody, err := json.Marshal(chatRequest{
		Model: s.cfg.AIModel,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.75,
	})
	if err != nil {
		return "", domain.ErrAIGenerationFailed
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", domain.ErrAIGenerationFailed
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.AIAPIKey)

	res, err := s.client.Do(req)
	if err != nil {
		return "", domain.ErrAIServiceUnavailable
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", domain.ErrAIGenerationFailed
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var apiErr apiErrorBody
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return "", fmt.Errorf("%w: %s", domain.ErrAIGenerationFailed, apiErr.Error.Message)
		}
		return "", domain.ErrAIGenerationFailed
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", domain.ErrAIGenerationFailed
	}
	if len(parsed.Choices) == 0 {
		return "", domain.ErrAIGenerationFailed
	}
	return parsed.Choices[0].Message.Content, nil
}
