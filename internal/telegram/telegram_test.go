package telegram

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSendMessageBuildsRequest(t *testing.T) {
	var gotMethod, gotPath, gotCT string
	var gotReq sendMessageRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer ts.Close()

	c := New("test-token", discardLogger())
	c.BaseURL = ts.URL

	err := c.SendMessage(t.Context(), "12345", "<b>hi</b>", "https://superset/alert/1")
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/bottest-token/sendMessage", gotPath)
	assert.Equal(t, "application/json", gotCT)
	assert.Equal(t, "12345", gotReq.ChatID)
	assert.Equal(t, "<b>hi</b>", gotReq.Text)
	assert.Equal(t, "HTML", gotReq.ParseMode)
	assert.True(t, gotReq.LinkPreviewOptions.IsDisabled)

	require.NotNil(t, gotReq.ReplyMarkup)
	require.NotEmpty(t, gotReq.ReplyMarkup.InlineKeyboard)
	require.NotEmpty(t, gotReq.ReplyMarkup.InlineKeyboard[0])
	btn := gotReq.ReplyMarkup.InlineKeyboard[0][0]
	assert.Equal(t, "Open in Superset", btn.Text)
	assert.Equal(t, "https://superset/alert/1", btn.URL)
}

func TestSendMessageNoButtonWhenURLEmpty(t *testing.T) {
	var gotReq sendMessageRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer ts.Close()

	c := New("t", discardLogger())
	c.BaseURL = ts.URL

	require.NoError(t, c.SendMessage(t.Context(), "1", "hi", ""))
	assert.Nil(t, gotReq.ReplyMarkup)
}

func TestSendMessageRetriesOnTransportError(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			// Abort the first request at the transport level so the client
			// sees a transport error and retries.
			panic(http.ErrAbortHandler)
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer ts.Close()

	c := New("t", discardLogger())
	c.BaseURL = ts.URL
	c.RetryBackoff = 0 // keep the test fast

	require.NoError(t, c.SendMessage(t.Context(), "1", "hi", ""))
	assert.Equal(t, int32(2), calls.Load())
}

func TestSendMessageFailsAfterRetries(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := ts.URL
	ts.Close() // closed server: every attempt fails at the transport level.

	c := New("t", discardLogger())
	c.BaseURL = url
	c.RetryBackoff = 0

	require.Error(t, c.SendMessage(t.Context(), "1", "hi", ""))
}

func TestSendMessageStopsRetryWhenContextCancelled(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		panic(http.ErrAbortHandler) // every attempt fails at the transport level
	}))
	defer ts.Close()

	c := New("t", discardLogger())
	c.BaseURL = ts.URL
	c.RetryBackoff = 10 * time.Second // long; cancellation must cut the backoff short

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := c.SendMessage(ctx, "1", "hi", "")
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, 5*time.Second, "backoff should abort when context is cancelled")
	assert.Equal(t, int32(1), calls.Load(), "must not retry once the context is cancelled")
}

func TestSendMessageRedactsTokenOnTransportError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := ts.URL
	ts.Close() // force a transport error

	const token = "super-secret-token"
	c := New(token, discardLogger())
	c.BaseURL = baseURL
	c.RetryBackoff = 0

	err := c.SendMessage(t.Context(), "1", "hi", "")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), token, "bot token must not leak into errors")
	assert.Contains(t, err.Error(), "<redacted>")
}

func TestSendMessageNon2xxReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"ok":false,"description":"bad chat_id"}`)
	}))
	defer ts.Close()

	c := New("t", discardLogger())
	c.BaseURL = ts.URL

	require.Error(t, c.SendMessage(t.Context(), "1", "hi", ""))
}
