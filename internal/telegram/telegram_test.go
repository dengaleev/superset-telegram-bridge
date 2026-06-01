package telegram

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
