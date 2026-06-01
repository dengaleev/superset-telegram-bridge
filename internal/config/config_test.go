package config_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dengaleev/superset-telegram-bridge/internal/config"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    config.Config
		wantErr string
	}{
		{
			name: "defaults",
			env:  map[string]string{"TELEGRAM_TOKEN": "tok", "TELEGRAM_CHAT_ID": "123"},
			want: config.Config{TelegramToken: "tok", TelegramChatID: "123", ListenAddr: ":8080", LogLevel: slog.LevelInfo},
		},
		{
			name: "overrides",
			env:  map[string]string{"TELEGRAM_TOKEN": "tok", "TELEGRAM_CHAT_ID": "123", "LISTEN_ADDR": ":9000", "LOG_LEVEL": "debug"},
			want: config.Config{TelegramToken: "tok", TelegramChatID: "123", ListenAddr: ":9000", LogLevel: slog.LevelDebug},
		},
		{
			name:    "missing both required lists every var",
			env:     map[string]string{},
			wantErr: "missing required env vars: TELEGRAM_TOKEN, TELEGRAM_CHAT_ID",
		},
		{
			name:    "missing one required lists only that var",
			env:     map[string]string{"TELEGRAM_TOKEN": "tok"},
			wantErr: "missing required env vars: TELEGRAM_CHAT_ID",
		},
		{
			name:    "invalid log level",
			env:     map[string]string{"TELEGRAM_TOKEN": "tok", "TELEGRAM_CHAT_ID": "123", "LOG_LEVEL": "loud"},
			wantErr: `invalid LOG_LEVEL "loud"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Blank every var Load reads so unset cases don't inherit host env.
			for _, k := range []string{"TELEGRAM_TOKEN", "TELEGRAM_CHAT_ID", "LISTEN_ADDR", "LOG_LEVEL"} {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := config.Load()
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
