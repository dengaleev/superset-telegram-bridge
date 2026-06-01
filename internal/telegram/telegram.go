// Package telegram is a minimal net/http client for the Telegram Bot API.
// It is a pure transport: it shapes no payloads beyond the sendMessage envelope.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.telegram.org"

// Client sends messages to the Telegram Bot API.
type Client struct {
	// BaseURL defaults to the public API; override in tests.
	BaseURL string

	token  string
	http   *http.Client
	logger *slog.Logger
}

// New returns a Client for the given bot token. If logger is nil, the default
// slog logger is used.
func New(token string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		BaseURL: defaultBaseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
		logger:  logger,
	}
}

type linkPreviewOptions struct {
	IsDisabled bool `json:"is_disabled"`
}

type inlineButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type replyMarkup struct {
	InlineKeyboard [][]inlineButton `json:"inline_keyboard"`
}

type sendMessageRequest struct {
	ChatID             string             `json:"chat_id"`
	Text               string             `json:"text"`
	ParseMode          string             `json:"parse_mode"`
	LinkPreviewOptions linkPreviewOptions `json:"link_preview_options"`
	ReplyMarkup        *replyMarkup       `json:"reply_markup,omitempty"`
}

// SendMessage posts a sendMessage call. If buttonURL is non-empty it adds an
// inline "Open in Superset" button. Transport errors are retried once; an
// HTTP non-2xx response is returned immediately (not retried).
func (c *Client) SendMessage(ctx context.Context, chatID, text, buttonURL string) error {
	body := sendMessageRequest{
		ChatID:             chatID,
		Text:               text,
		ParseMode:          "HTML",
		LinkPreviewOptions: linkPreviewOptions{IsDisabled: true},
	}
	if buttonURL != "" {
		body.ReplyMarkup = &replyMarkup{
			InlineKeyboard: [][]inlineButton{{{Text: "Open in Superset", URL: buttonURL}}},
		}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal sendMessage request: %w", err)
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", c.BaseURL, c.token)

	var lastErr error
	for attempt := range 2 {
		status, respBody, err := c.postOnce(ctx, endpoint, payload)
		if err != nil {
			lastErr = err
			c.logger.Warn("telegram request failed", "attempt", attempt, "error", err)
			// No point retrying if the caller's context is already done.
			if ctx.Err() != nil {
				break
			}
			continue
		}
		if status/100 != 2 {
			return fmt.Errorf("telegram sendMessage returned %d: %s", status, respBody)
		}
		return nil
	}
	return fmt.Errorf("telegram sendMessage failed after %d attempt(s): %w", 2, lastErr)
}

func (c *Client) postOnce(ctx context.Context, endpoint string, payload []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, c.redactToken(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, c.redactToken(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("read response: %w", err)
	}
	return resp.StatusCode, respBody, nil
}

// redactToken strips the bot token from transport errors. net/http returns a
// *url.Error whose string embeds the full request URL (including the token),
// which must never reach logs. The underlying cause is preserved via %w.
func (c *Client) redactToken(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return fmt.Errorf("%s %s/bot<redacted>/sendMessage: %w", ue.Op, c.BaseURL, ue.Err)
	}
	return err
}
