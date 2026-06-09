package config

import (
	"os"
	"strings"
	"strconv"
)

type Config struct {
	Port         string
	MySQLDSN     string
	RedisAddr    string
	RedisPass    string
	RedisDB      int
	FrontendURL  string
	FrontendURLs []string
	JWTSecret    string
	StreamRTMPHost   string
	StreamHLSBase    string
	PayTimeoutMinutes int
	AIAPIKey     string
	AIAPIBase    string
	AIModel      string
	AITTSModel   string
	AITTSVoice   string
}

func Load() Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	payTimeoutMin, _ := strconv.Atoi(getEnv("PAY_TIMEOUT_MINUTES", "30"))
	if payTimeoutMin < 1 {
		payTimeoutMin = 30
	}
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:5173")
	frontendURLs := parseCSV(getEnv("FRONTEND_URLS", ""))
	if len(frontendURLs) == 0 {
		frontendURLs = []string{frontendURL}
	}
	return Config{
		Port:         getEnv("PORT", "8081"),
		MySQLDSN:     getEnv("MYSQL_DSN", "zhibo:zhibo@tcp(localhost:3306)/zhibo?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:      redisDB,
		FrontendURL:  frontendURL,
		FrontendURLs: frontendURLs,
		JWTSecret:        getEnv("JWT_SECRET", "zhibo-dev-jwt-secret-change-in-prod"),
		StreamRTMPHost:    getEnv("STREAM_RTMP_HOST", "localhost:1935"),
		StreamHLSBase:     getEnv("STREAM_HLS_BASE", "/live"),
		PayTimeoutMinutes: payTimeoutMin,
		AIAPIKey:          getEnv("AI_API_KEY", ""),
		AIAPIBase:         getEnv("AI_API_BASE", "https://api.openai.com/v1"),
		AIModel:           getEnv("AI_MODEL", "gpt-4o-mini"),
		AITTSModel:        getEnv("AI_TTS_MODEL", "tts-1"),
		AITTSVoice:        getEnv("AI_TTS_VOICE", "nova"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
