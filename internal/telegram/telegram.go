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
	"strings"
	"time"
)

const (
	defaultBaseURL      = "https://api.telegram.org"
	defaultRetryBackoff = 500 * time.Millisecond
	maxAttempts         = 2
)

// Client sends messages to the Telegram Bot API.
type Client struct {
	// BaseURL defaults to the public API; override in tests.
	BaseURL string
	// RetryBackoff is the delay before retrying a failed transport attempt.
	RetryBackoff time.Duration

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
		BaseURL:      defaultBaseURL,
		RetryBackoff: defaultRetryBackoff,
		token:        token,
		http:         &http.Client{Timeout: 10 * time.Second},
		logger:       logger,
	}
}

type linkPreviewOptions struct {
	IsDisabled bool `json:"is_disabled"`
}

type sendMessageRequest struct {
	ChatID             string             `json:"chat_id"`
	Text               string             `json:"text"`
	ParseMode          string             `json:"parse_mode"`
	LinkPreviewOptions linkPreviewOptions `json:"link_preview_options"`
}

// SendMessage posts a plain text message (the "Open in Superset" link lives in
// the text). Transport failures are retried; a non-2xx is returned immediately.
func (c *Client) SendMessage(ctx context.Context, chatID, text string) error {
	payload, err := json.Marshal(sendMessageRequest{
		ChatID:             chatID,
		Text:               text,
		ParseMode:          "HTML",
		LinkPreviewOptions: linkPreviewOptions{IsDisabled: true},
	})
	if err != nil {
		return fmt.Errorf("marshal sendMessage request: %w", err)
	}
	return c.send(ctx, "sendMessage", "application/json", payload)
}

// send POSTs body to the given Bot API method, retrying transport failures up to
// maxAttempts with a fixed RetryBackoff. A non-2xx is returned immediately.
func (c *Client) send(ctx context.Context, method, contentType string, body []byte) error {
	endpoint := fmt.Sprintf("%s/bot%s/%s", c.BaseURL, c.token, method)
	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 && !sleep(ctx, c.RetryBackoff) {
			break // context cancelled during backoff
		}
		status, respBody, err := c.postOnce(ctx, endpoint, contentType, body)
		if err != nil {
			lastErr = err
			c.logger.WarnContext(ctx, "telegram request failed", "method", method, "attempt", attempt, "error", err)
			continue
		}
		if status/100 != 2 {
			return fmt.Errorf("telegram %s returned %d: %s", method, status, respBody)
		}
		return nil
	}
	return fmt.Errorf("telegram %s failed after %d attempt(s): %w", method, maxAttempts, lastErr)
}

// sleep blocks for d or until ctx is done. It returns true if the delay
// elapsed, false if ctx was cancelled first.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (c *Client) postOnce(ctx context.Context, endpoint, contentType string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, c.redactToken(err)
	}
	req.Header.Set("Content-Type", contentType)

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

// redactToken removes the bot token from transport errors before they are logged
// or surfaced (net/http's *url.Error embeds the full request URL).
func (c *Client) redactToken(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return fmt.Errorf("%s %s: %w", ue.Op, strings.ReplaceAll(ue.URL, c.token, "<redacted>"), ue.Err)
	}
	return err
}
