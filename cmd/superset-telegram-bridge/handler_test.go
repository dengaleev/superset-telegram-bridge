package main

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
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestHandler returns the webhook handler with a telegram client pointed at
// a stub server returning telegramStatus, plus a flag reporting whether the
// stub was called.
func newTestHandler(t *testing.T, telegramStatus int) (http.Handler, *atomic.Bool) {
	t.Helper()
	var hit atomic.Bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		w.WriteHeader(telegramStatus)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(ts.Close)

	tg := telegram.New("tok", discardLogger())
	tg.BaseURL = ts.URL
	return webhookHandler(tg, "123", discardLogger()), &hit
}

func TestWebhookHappyPath(t *testing.T) {
	h, hit := newTestHandler(t, http.StatusOK)

	req := httptest.NewRequest(http.MethodPost, "/webhook",
		strings.NewReader(`{"name":"High 500s","text":"met","url":"https://superset/1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.True(t, hit.Load(), "telegram stub should have been called")
}

func TestWebhookUnsupportedMediaType(t *testing.T) {
	h, _ := newTestHandler(t, http.StatusOK)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestWebhookMalformedJSON(t *testing.T) {
	h, _ := newTestHandler(t, http.StatusOK)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"name":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookTelegramFailure(t *testing.T) {
	h, _ := newTestHandler(t, http.StatusInternalServerError)

	req := httptest.NewRequest(http.MethodPost, "/webhook",
		strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}
