package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/dengaleev/superset-telegram-bridge/internal/message"
	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
	"github.com/dengaleev/superset-telegram-bridge/internal/telegram"
)

// maxBodyBytes caps the inbound webhook body size.
const maxBodyBytes = 1 << 20 // 1 MiB

// webhookHandler returns the POST /webhook handler. It reads the raw body
// (kept raw so HMAC verification can hook in during Phase 3), parses the
// Superset payload, renders it, and forwards it to Telegram.
func webhookHandler(tg *telegram.Client, chatID string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				logger.Warn("request body too large", "limit", maxBodyBytes)
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			logger.Warn("read request body", "error", err)
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}

		payload, err := superset.Parse(r.Header.Get("Content-Type"), body)
		if err != nil {
			if errors.Is(err, superset.ErrUnsupportedMediaType) {
				http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
				return
			}
			logger.Warn("parse payload", "error", err)
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		rendered := message.Render(payload)
		if err := tg.SendMessage(r.Context(), chatID, rendered.Text, rendered.ButtonURL); err != nil {
			logger.Error("send telegram message", "error", err)
			http.Error(w, "upstream error", http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
