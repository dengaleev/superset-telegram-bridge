package config

import (
	"log/slog"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "123")
	// LISTEN_ADDR and LOG_LEVEL unset.

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TelegramToken != "tok" {
		t.Errorf("TelegramToken = %q, want %q", cfg.TelegramToken, "tok")
	}
	if cfg.TelegramChatID != "123" {
		t.Errorf("TelegramChatID = %q, want %q", cfg.TelegramChatID, "123")
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "123")
	t.Setenv("LISTEN_ADDR", ":9000")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ListenAddr != ":9000" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":9000")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing vars")
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "123")
	t.Setenv("LOG_LEVEL", "loud")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error for invalid LOG_LEVEL")
	}
}
