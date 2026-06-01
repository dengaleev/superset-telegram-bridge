package telegram_test

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dengaleev/superset-telegram-bridge/internal/telegram"
)

// capture parses the inbound multipart request a Telegram send produced and
// hands the test the path, form fields, and uploaded files.
func capture(t *testing.T, sink func(path string, fields map[string]string, files map[string][]byte)) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		require.NoError(t, err)
		require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		mr := multipart.NewReader(r.Body, params["boundary"])
		fields := map[string]string{}
		files := map[string][]byte{}
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			b, _ := io.ReadAll(p)
			if p.FileName() != "" {
				files[p.FileName()] = b
			} else {
				fields[p.FormName()] = string(b)
			}
		}
		sink(r.URL.Path, fields, files)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	return ts
}

func TestSendPhoto(t *testing.T) {
	var path string
	var fields map[string]string
	var files map[string][]byte
	ts := capture(t, func(p string, fl map[string]string, f map[string][]byte) {
		path, fields, files = p, fl, f
	})
	defer ts.Close()

	c := telegram.New("tok", discardLogger())
	c.BaseURL = ts.URL

	require.NoError(t, c.SendPhoto(t.Context(), "42", "hi", telegram.Media{
		Filename: "screenshot_0.png", MIME: "image/png", Data: []byte("PNGBYTES"),
	}))

	assert.Equal(t, "/bottok/sendPhoto", path)
	assert.Equal(t, "42", fields["chat_id"])
	assert.Equal(t, "hi", fields["caption"])
	assert.Equal(t, "HTML", fields["parse_mode"])
	assert.Equal(t, []byte("PNGBYTES"), files["screenshot_0.png"])
}

func TestSendDocument(t *testing.T) {
	var path string
	var files map[string][]byte
	ts := capture(t, func(p string, fl map[string]string, f map[string][]byte) {
		path, files = p, f
	})
	defer ts.Close()

	c := telegram.New("tok", discardLogger())
	c.BaseURL = ts.URL

	require.NoError(t, c.SendDocument(t.Context(), "42", "doc", telegram.Media{
		Filename: "report.csv", MIME: "text/csv", Data: []byte("a,b\n1,2"),
	}))

	assert.Equal(t, "/bottok/sendDocument", path)
	assert.Equal(t, []byte("a,b\n1,2"), files["report.csv"])
}

func TestSendMediaGroupCaptionOnFirstOnly(t *testing.T) {
	var path, mediaJSON string
	var files map[string][]byte
	ts := capture(t, func(p string, fl map[string]string, f map[string][]byte) {
		path, mediaJSON, files = p, fl["media"], f
	})
	defer ts.Close()

	c := telegram.New("tok", discardLogger())
	c.BaseURL = ts.URL

	require.NoError(t, c.SendMediaGroup(t.Context(), "42", "album caption", "photo", []telegram.Media{
		{Filename: "screenshot_0.png", MIME: "image/png", Data: []byte("A")},
		{Filename: "screenshot_1.png", MIME: "image/png", Data: []byte("B")},
	}))

	assert.Equal(t, "/bottok/sendMediaGroup", path)
	assert.Len(t, files, 2)
	assert.Contains(t, mediaJSON, "attach://screenshot_0.png")
	assert.Contains(t, mediaJSON, "attach://screenshot_1.png")
	assert.Contains(t, mediaJSON, "album caption")
	// the caption field appears exactly once (first item only); match the JSON
	// key so the test caption value ("album caption") doesn't self-collide.
	assert.Equal(t, 1, strings.Count(mediaJSON, `"caption"`))
}
