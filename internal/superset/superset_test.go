package superset_test

import (
	"encoding/json"
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
		{name: "multipart rejected", contentType: "multipart/form-data; boundary=abc", body: "name=x", wantErr: true, wantUnsupported: true},
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
