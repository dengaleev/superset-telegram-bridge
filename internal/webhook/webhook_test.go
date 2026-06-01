package webhook_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dengaleev/superset-telegram-bridge/internal/telegram"
	"github.com/dengaleev/superset-telegram-bridge/internal/webhook"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// sentCall records one outbound Telegram call the handler made.
type sentCall struct {
	method  string
	caption string
	kind    string
	files   []string // filenames
}

// fakeSender records calls instead of talking to Telegram; err (if set) is
// returned by every send to exercise the failure path.
type fakeSender struct {
	calls []sentCall
	err   error
}

func (f *fakeSender) SendMessage(_ context.Context, _, text string) error {
	f.calls = append(f.calls, sentCall{method: "sendMessage", caption: text})
	return f.err
}

func (f *fakeSender) SendPhoto(_ context.Context, _, caption string, m telegram.Media) error {
	f.calls = append(f.calls, sentCall{method: "sendPhoto", caption: caption, files: []string{m.Filename}})
	return f.err
}

func (f *fakeSender) SendDocument(_ context.Context, _, caption string, m telegram.Media) error {
	f.calls = append(f.calls, sentCall{method: "sendDocument", caption: caption, files: []string{m.Filename}})
	return f.err
}

func (f *fakeSender) SendMediaGroup(_ context.Context, _, caption, kind string, files []telegram.Media) error {
	names := make([]string, len(files))
	for i, m := range files {
		names[i] = m.Filename
	}
	f.calls = append(f.calls, sentCall{method: "sendMediaGroup", caption: caption, kind: kind, files: names})
	return f.err
}

func (f *fakeSender) methods() []string {
	out := make([]string, len(f.calls))
	for i, c := range f.calls {
		out[i] = c.method
	}
	return out
}

type superFile struct{ name, mime string }

func pngFiles(n int) []superFile {
	out := make([]superFile, n)
	for i := range out {
		out[i] = superFile{fmt.Sprintf("screenshot_%d.png", i), "image/png"}
	}
	return out
}

func multipartBody(t *testing.T, files []superFile) (string, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("name", "x")
	_ = mw.WriteField("url", "https://s/1")
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

func post(f *fakeSender, ct string, body []byte) *httptest.ResponseRecorder {
	h := webhook.Handler(f, "123", discardLogger())
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
		wantMethods []string
	}{
		{name: "no files -> sendMessage", files: nil, wantMethods: []string{"sendMessage"}},
		{name: "one png -> sendPhoto", files: []superFile{{"screenshot_0.png", "image/png"}}, wantMethods: []string{"sendPhoto"}},
		{name: "two png -> sendMediaGroup", files: pngFiles(2), wantMethods: []string{"sendMediaGroup"}},
		{name: "one pdf -> sendDocument", files: []superFile{{"report.pdf", "application/pdf"}}, wantMethods: []string{"sendDocument"}},
		{name: "one csv -> sendDocument", files: []superFile{{"report.csv", "text/csv"}}, wantMethods: []string{"sendDocument"}},
		{name: "mixed png+pdf -> photo then document", files: []superFile{{"screenshot_0.png", "image/png"}, {"report.pdf", "application/pdf"}}, wantMethods: []string{"sendPhoto", "sendDocument"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeSender{}
			var rec *httptest.ResponseRecorder
			if tt.files == nil {
				rec = post(f, "application/json", []byte(`{"name":"x","url":"https://s/1"}`))
			} else {
				ct, body := multipartBody(t, tt.files)
				rec = post(f, ct, body)
			}
			assert.Equal(t, http.StatusNoContent, rec.Code)
			assert.Equal(t, tt.wantMethods, f.methods())
		})
	}
}

func TestHandlerCaptionOnlyOnFirstGroup(t *testing.T) {
	f := &fakeSender{}
	ct, body := multipartBody(t, []superFile{{"screenshot_0.png", "image/png"}, {"report.pdf", "application/pdf"}})

	post(f, ct, body)

	require.Len(t, f.calls, 2)
	assert.Equal(t, "sendPhoto", f.calls[0].method)
	assert.NotEmpty(t, f.calls[0].caption, "photo carries the caption")
	assert.Equal(t, "sendDocument", f.calls[1].method)
	assert.Empty(t, f.calls[1].caption, "document caption is cleared after photos")
}

func TestHandlerAlbumOverflowDroppedToTen(t *testing.T) {
	f := &fakeSender{}
	ct, body := multipartBody(t, pngFiles(11))

	post(f, ct, body)

	require.Equal(t, []string{"sendMediaGroup"}, f.methods())
	assert.Len(t, f.calls[0].files, telegram.MaxMediaGroup, "overflow beyond the album limit is dropped")
}

func TestHandlerUnsupportedMediaType(t *testing.T) {
	f := &fakeSender{}
	rec := post(f, "text/plain", []byte("nope"))
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Empty(t, f.calls)
}

func TestHandlerMalformedJSON(t *testing.T) {
	f := &fakeSender{}
	rec := post(f, "application/json", []byte(`{"name":`))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandlerTelegramFailure(t *testing.T) {
	f := &fakeSender{err: errors.New("boom")}
	rec := post(f, "application/json", []byte(`{"name":"x"}`))
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestHandlerBodyTooLarge(t *testing.T) {
	f := &fakeSender{}
	rec := post(f, "application/json", bytes.Repeat([]byte("x"), (50<<20)+1))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Empty(t, f.calls)
}
