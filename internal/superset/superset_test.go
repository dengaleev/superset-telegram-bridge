package superset_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
)

func TestParseDecodesPayload(t *testing.T) {
	body := []byte(`{"name":"High 500s","text":"condition met","description":"prod","url":"https://superset/alert/1","header":{"k":"v"}}`)

	p, err := superset.Parse("application/json", body)
	require.NoError(t, err)
	assert.Equal(t, superset.Payload{
		Name:        "High 500s",
		Text:        "condition met",
		Description: "prod",
		URL:         "https://superset/alert/1",
		Header:      json.RawMessage(`{"k":"v"}`),
	}, p)
}

func TestParse(t *testing.T) {
	tests := []struct {
		name            string
		contentType     string
		body            string
		wantErr         bool
		wantUnsupported bool // only consulted when wantErr
	}{
		{name: "json accepted", contentType: "application/json", body: `{"name":"x"}`},
		{name: "json with charset param accepted", contentType: "application/json; charset=utf-8", body: `{"name":"x"}`},
		{name: "multipart missing boundary is a parse error, not unsupported", contentType: "multipart/form-data", body: "name=x", wantErr: true},
		{name: "empty content type rejected", contentType: "", body: `{"name":"x"}`, wantErr: true, wantUnsupported: true},
		{name: "malformed json is a parse error, not unsupported", contentType: "application/json", body: `{"name": `, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := superset.Parse(tt.contentType, []byte(tt.body))
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tt.wantUnsupported {
				require.ErrorIs(t, err, superset.ErrUnsupportedMediaType)
			} else {
				require.NotErrorIs(t, err, superset.ErrUnsupportedMediaType)
			}
		})
	}
}

func TestParseMultipart(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("name", "High 500s")
	_ = mw.WriteField("text", "met")
	_ = mw.WriteField("description", "prod")
	_ = mw.WriteField("url", "https://superset/1")
	_ = mw.WriteField("header", `{"notification_type":"Alert"}`)
	part, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="files"; filename="screenshot_0.png"`},
		"Content-Type":        []string{"image/png"},
	})
	_, _ = part.Write([]byte("PNGBYTES"))
	_ = mw.Close()

	p, err := superset.Parse(mw.FormDataContentType(), buf.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "High 500s", p.Name)
	assert.Equal(t, "met", p.Text)
	assert.Equal(t, "prod", p.Description)
	assert.Equal(t, "https://superset/1", p.URL)
	assert.JSONEq(t, `{"notification_type":"Alert"}`, string(p.Header))
	require.Len(t, p.Files, 1)
	assert.Equal(t, "screenshot_0.png", p.Files[0].Filename)
	assert.Equal(t, "image/png", p.Files[0].MIME)
	assert.Equal(t, []byte("PNGBYTES"), p.Files[0].Data)
}

func TestParseMultipartMissingBoundary(t *testing.T) {
	_, err := superset.Parse("multipart/form-data", []byte("whatever"))
	require.Error(t, err)
	require.NotErrorIs(t, err, superset.ErrUnsupportedMediaType)
}
