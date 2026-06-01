package webhook_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dengaleev/superset-telegram-bridge/internal/telegram"
	"github.com/dengaleev/superset-telegram-bridge/internal/webhook"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler(t *testing.T) {
	const maxBody = 1 << 20

	tests := []struct {
		name           string
		contentType    string
		body           string
		telegramStatus int
		wantCode       int
		wantForwarded  bool
	}{
		{
			name:           "happy path forwards and returns 204",
			contentType:    "application/json",
			body:           `{"name":"High 500s","text":"met","url":"https://superset/1"}`,
			telegramStatus: http.StatusOK,
			wantCode:       http.StatusNoContent,
			wantForwarded:  true,
		},
		{
			name:        "unsupported media type returns 415",
			contentType: "text/plain",
			body:        "name=x",
			wantCode:    http.StatusUnsupportedMediaType,
		},
		{
			name:        "malformed json returns 400",
			contentType: "application/json",
			body:        `{"name":`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "oversized body returns 413",
			contentType: "application/json",
			body:        `{"name":"` + strings.Repeat("x", maxBody+1) + `"}`,
			wantCode:    http.StatusRequestEntityTooLarge,
		},
		{
			name:           "telegram failure returns 502",
			contentType:    "application/json",
			body:           `{"name":"x"}`,
			telegramStatus: http.StatusInternalServerError,
			wantCode:       http.StatusBadGateway,
			wantForwarded:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var forwarded atomic.Bool
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				forwarded.Store(true)
				w.WriteHeader(tt.telegramStatus)
				_, _ = io.WriteString(w, `{"ok":true}`)
			}))
			defer ts.Close()

			tg := telegram.New("tok", discardLogger())
			tg.BaseURL = ts.URL
			h := webhook.Handler(tg, "123", discardLogger())

			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Equal(t, tt.wantForwarded, forwarded.Load())
		})
	}
}
