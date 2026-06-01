package webhook_test

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dengaleev/superset-telegram-bridge/internal/telegram"
	"github.com/dengaleev/superset-telegram-bridge/internal/webhook"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// stub records which Bot API methods the bridge called, and can force a status.
type stub struct {
	mu      sync.Mutex
	methods []string
	status  int
}

func (s *stub) server(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.methods = append(s.methods, strings.TrimPrefix(r.URL.Path, "/bottok/"))
		s.mu.Unlock()
		if s.status != 0 {
			w.WriteHeader(s.status)
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func (s *stub) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.methods...)
}

type superFile struct{ name, mime string }

func pngFiles(n int) []superFile {
	out := make([]superFile, n)
	for i := range out {
		out[i] = superFile{fmt.Sprintf("screenshot_%d.png", i), "image/png"}
	}
	return out
}

func multipartBody(t *testing.T, fields map[string]string, files []superFile) (string, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	for _, f := range files {
		h := textproto.MIMEHeader{
			"Content-Disposition": []string{`form-data; name="files"; filename="` + f.name + `"`},
			"Content-Type":        []string{f.mime},
		}
		part, _ := mw.CreatePart(h)
		_, _ = part.Write([]byte("BYTES"))
	}
	_ = mw.Close()
	return mw.FormDataContentType(), buf.Bytes()
}

func newHandler(t *testing.T, s *stub) http.Handler {
	t.Helper()
	tg := telegram.New("tok", discardLogger())
	tg.BaseURL = s.server(t).URL
	return webhook.Handler(tg, "123", discardLogger())
}

func post(h http.Handler, ct string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandlerRouting(t *testing.T) {
	tests := []struct {
		name        string
		files       []superFile
		wantCode    int
		wantMethods []string
	}{
		{name: "no files -> sendMessage", files: nil, wantCode: 204, wantMethods: []string{"sendMessage"}},
		{name: "one png -> sendPhoto", files: []superFile{{"screenshot_0.png", "image/png"}}, wantCode: 204, wantMethods: []string{"sendPhoto"}},
		{name: "two png -> sendMediaGroup", files: []superFile{{"screenshot_0.png", "image/png"}, {"screenshot_1.png", "image/png"}}, wantCode: 204, wantMethods: []string{"sendMediaGroup"}},
		{name: "one pdf -> sendDocument", files: []superFile{{"report.pdf", "application/pdf"}}, wantCode: 204, wantMethods: []string{"sendDocument"}},
		{name: "one csv -> sendDocument", files: []superFile{{"report.csv", "text/csv"}}, wantCode: 204, wantMethods: []string{"sendDocument"}},
		{name: "mixed png+pdf -> photo then document", files: []superFile{{"screenshot_0.png", "image/png"}, {"report.pdf", "application/pdf"}}, wantCode: 204, wantMethods: []string{"sendPhoto", "sendDocument"}},
		{name: "eleven png -> single album, overflow dropped", files: pngFiles(11), wantCode: 204, wantMethods: []string{"sendMediaGroup"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stub{}
			h := newHandler(t, s)
			var rec *httptest.ResponseRecorder
			if tt.files == nil {
				rec = post(h, "application/json", []byte(`{"name":"x","url":"https://s/1"}`))
			} else {
				ct, body := multipartBody(t, map[string]string{"name": "x", "url": "https://s/1"}, tt.files)
				rec = post(h, ct, body)
			}
			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Equal(t, tt.wantMethods, s.calls())
		})
	}
}

func TestHandlerUnsupportedMediaType(t *testing.T) {
	s := &stub{}
	h := newHandler(t, s)
	rec := post(h, "text/plain", []byte("nope"))
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Empty(t, s.calls())
}

func TestHandlerMalformedJSON(t *testing.T) {
	s := &stub{}
	h := newHandler(t, s)
	rec := post(h, "application/json", []byte(`{"name":`))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandlerTelegramFailure(t *testing.T) {
	s := &stub{status: http.StatusInternalServerError}
	h := newHandler(t, s)
	rec := post(h, "application/json", []byte(`{"name":"x"}`))
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestHandlerBodyTooLarge(t *testing.T) {
	s := &stub{}
	h := newHandler(t, s)
	rec := post(h, "application/json", bytes.Repeat([]byte("x"), (50<<20)+1))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Empty(t, s.calls())
}
