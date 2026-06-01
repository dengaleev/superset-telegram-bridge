package config

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "123")
	// LISTEN_ADDR and LOG_LEVEL unset.

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "tok", cfg.TelegramToken)
	assert.Equal(t, "123", cfg.TelegramChatID)
	assert.Equal(t, ":8080", cfg.ListenAddr)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "123")
	t.Setenv("LISTEN_ADDR", ":9000")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, ":9000", cfg.ListenAddr)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")

	_, err := Load()
	require.Error(t, err)
	// The error must list every missing variable, not just the first.
	assert.Contains(t, err.Error(), "TELEGRAM_TOKEN")
	assert.Contains(t, err.Error(), "TELEGRAM_CHAT_ID")
}

func TestLoadMissingOneRequired(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TELEGRAM_CHAT_ID")
	assert.NotContains(t, err.Error(), "TELEGRAM_TOKEN")
}

func TestLoadInvalidLogLevel(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "123")
	t.Setenv("LOG_LEVEL", "loud")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loud")
}
