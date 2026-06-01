package telegram

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
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
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/bottest-token/sendMessage" {
		t.Errorf("path = %q, want %q", gotPath, "/bottest-token/sendMessage")
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q, want application/json", gotCT)
	}
	if gotReq.ChatID != "12345" {
		t.Errorf("chat_id = %q, want 12345", gotReq.ChatID)
	}
	if gotReq.Text != "<b>hi</b>" {
		t.Errorf("text = %q, want %q", gotReq.Text, "<b>hi</b>")
	}
	if gotReq.ParseMode != "HTML" {
		t.Errorf("parse_mode = %q, want HTML", gotReq.ParseMode)
	}
	if !gotReq.LinkPreviewOptions.IsDisabled {
		t.Error("link_preview_options.is_disabled = false, want true")
	}
	if gotReq.ReplyMarkup == nil {
		t.Fatal("reply_markup = nil, want inline keyboard")
	}
	btn := gotReq.ReplyMarkup.InlineKeyboard[0][0]
	if btn.URL != "https://superset/alert/1" {
		t.Errorf("button url = %q, want %q", btn.URL, "https://superset/alert/1")
	}
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

	if err := c.SendMessage(t.Context(), "1", "hi", ""); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if gotReq.ReplyMarkup != nil {
		t.Error("reply_markup set, want nil when button URL empty")
	}
}

func TestSendMessageNon2xxReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"ok":false,"description":"bad chat_id"}`)
	}))
	defer ts.Close()

	c := New("t", discardLogger())
	c.BaseURL = ts.URL

	err := c.SendMessage(t.Context(), "1", "hi", "")
	if err == nil {
		t.Fatal("SendMessage() error = nil, want error on non-2xx")
	}
}
