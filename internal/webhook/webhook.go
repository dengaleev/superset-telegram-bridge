// Package webhook is the HTTP boundary: it turns an inbound Superset webhook
// request into a Telegram message.
package webhook

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/dengaleev/superset-telegram-bridge/internal/message"
	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
	"github.com/dengaleev/superset-telegram-bridge/internal/telegram"
)

// maxBodyBytes bounds request bodies so a hostile caller can't exhaust memory.
const maxBodyBytes = 1 << 20 // 1 MiB

// Handler returns the POST /webhook handler. The raw body is read before
// parsing so HMAC verification can hook in during Phase 3 without restructuring.
func Handler(tg *telegram.Client, chatID string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				logger.WarnContext(ctx, "request body too large", "limit", maxBodyBytes)
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			logger.WarnContext(ctx, "read request body", "error", err)
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}

		payload, err := superset.Parse(r.Header.Get("Content-Type"), body)
		if err != nil {
			if errors.Is(err, superset.ErrUnsupportedMediaType) {
				http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
				return
			}
			logger.WarnContext(ctx, "parse payload", "error", err)
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if err := tg.SendMessage(ctx, chatID, message.Render(payload, message.TextMaxLen)); err != nil {
			logger.ErrorContext(ctx, "send telegram message", "error", err)
			http.Error(w, "upstream error", http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
