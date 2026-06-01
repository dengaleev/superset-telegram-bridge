// Package config loads and validates runtime configuration from the environment.
package config

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config holds validated runtime configuration.
type Config struct {
	TelegramToken  string
	TelegramChatID string
	ListenAddr     string
	LogLevel       slog.Level
}

// Load reads configuration from the environment, applying defaults and
// validating required values. It returns an error listing any missing
// required variables or an invalid LOG_LEVEL.
func Load() (Config, error) {
	cfg := Config{
		TelegramToken:  os.Getenv("TELEGRAM_TOKEN"),
		TelegramChatID: os.Getenv("TELEGRAM_CHAT_ID"),
		ListenAddr:     cmp.Or(os.Getenv("LISTEN_ADDR"), ":8080"),
	}

	var missing []string
	if cfg.TelegramToken == "" {
		missing = append(missing, "TELEGRAM_TOKEN")
	}
	if cfg.TelegramChatID == "" {
		missing = append(missing, "TELEGRAM_CHAT_ID")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	level, err := parseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	return cfg, nil
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q", s)
	}
}
