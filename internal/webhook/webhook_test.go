package webhook_test

import (
	"bytes"
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
	"github.com/stretchr/testify/mock"

	"github.com/dengaleev/superset-telegram-bridge/internal/telegram"
	"github.com/dengaleev/superset-telegram-bridge/internal/webhook"
	"github.com/dengaleev/superset-telegram-bridge/internal/webhook/mocks"
)

const chatID = "123"

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type superFile struct{ name, mime string }

func png(name string) superFile        { return superFile{name, "image/png"} }
func file(name, mime string) superFile { return superFile{name, mime} }

func pngFiles(n int) []superFile {
	out := make([]superFile, n)
	for i := range out {
		out[i] = png(fmt.Sprintf("screenshot_%d.png", i))
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

func post(s webhook.Sender, ct string, body []byte) *httptest.ResponseRecorder {
	h := webhook.Handler(s, chatID, discardLogger())
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// postFiles sends a JSON body when files is nil, otherwise a multipart one.
func postFiles(t *testing.T, s webhook.Sender, files []superFile) *httptest.ResponseRecorder {
	t.Helper()
	if files == nil {
		return post(s, "application/json", []byte(`{"name":"x","url":"https://s/1"}`))
	}
	ct, body := multipartBody(t, files)
	return post(s, ct, body)
}

func TestHandlerRouting(t *testing.T) {
	tests := []struct {
		name   string
		files  []superFile
		expect func(e *mocks.MockSender_Expecter)
	}{
		{
			name:  "no files goes to SendMessage",
			files: nil,
			expect: func(e *mocks.MockSender_Expecter) {
				e.SendMessage(mock.Anything, chatID, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:  "one png goes to SendPhoto",
			files: []superFile{png("shot.png")},
			expect: func(e *mocks.MockSender_Expecter) {
				e.SendPhoto(mock.Anything, chatID, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:  "several png form an album",
			files: pngFiles(2),
			expect: func(e *mocks.MockSender_Expecter) {
				e.SendMediaGroup(mock.Anything, chatID, mock.Anything, "photo", mock.Anything).Return(nil).Once()
			},
		},
		{
			name:  "one pdf goes to SendDocument",
			files: []superFile{file("report.pdf", "application/pdf")},
			expect: func(e *mocks.MockSender_Expecter) {
				e.SendDocument(mock.Anything, chatID, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:  "one csv goes to SendDocument",
			files: []superFile{file("report.csv", "text/csv")},
			expect: func(e *mocks.MockSender_Expecter) {
				e.SendDocument(mock.Anything, chatID, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:  "mixed types go to photo and document",
			files: []superFile{png("shot.png"), file("report.pdf", "application/pdf")},
			expect: func(e *mocks.MockSender_Expecter) {
				e.SendPhoto(mock.Anything, chatID, mock.Anything, mock.Anything).Return(nil).Once()
				e.SendDocument(mock.Anything, chatID, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewMockSender(t) // its cleanup asserts the expected calls happened
			tt.expect(m.EXPECT())

			rec := postFiles(t, m, tt.files)

			assert.Equal(t, http.StatusNoContent, rec.Code)
		})
	}
}

func TestHandlerCaptionOnlyOnFirstGroup(t *testing.T) {
	m := mocks.NewMockSender(t)
	nonEmpty := mock.MatchedBy(func(c string) bool { return c != "" })
	m.EXPECT().SendPhoto(mock.Anything, chatID, nonEmpty, mock.Anything).Return(nil).Once()
	m.EXPECT().SendDocument(mock.Anything, chatID, "", mock.Anything).Return(nil).Once()

	postFiles(t, m, []superFile{png("shot.png"), file("report.pdf", "application/pdf")})
}

func TestHandlerAlbumOverflowDroppedToTen(t *testing.T) {
	m := mocks.NewMockSender(t)
	tenFiles := mock.MatchedBy(func(files []telegram.Media) bool { return len(files) == telegram.MaxMediaGroup })
	m.EXPECT().SendMediaGroup(mock.Anything, chatID, mock.Anything, "photo", tenFiles).Return(nil).Once()

	postFiles(t, m, pngFiles(11))
}

func TestHandlerTelegramFailure(t *testing.T) {
	m := mocks.NewMockSender(t)
	m.EXPECT().SendMessage(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("boom"))

	rec := post(m, "application/json", []byte(`{"name":"x"}`))
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestHandlerUnsupportedMediaType(t *testing.T) {
	m := mocks.NewMockSender(t) // no expectations: a send here would fail the test
	rec := post(m, "text/plain", []byte("nope"))
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestHandlerMalformedJSON(t *testing.T) {
	m := mocks.NewMockSender(t)
	rec := post(m, "application/json", []byte(`{"name":`))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandlerBodyTooLarge(t *testing.T) {
	m := mocks.NewMockSender(t)
	rec := post(m, "application/json", bytes.Repeat([]byte("x"), (50<<20)+1))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}
