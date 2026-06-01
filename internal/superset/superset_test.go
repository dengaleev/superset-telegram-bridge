package superset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidJSON(t *testing.T) {
	body := []byte(`{"name":"High 500s","text":"condition met","description":"prod","url":"https://superset/alert/1","header":{"k":"v"}}`)

	p, err := Parse("application/json", body)
	require.NoError(t, err)
	assert.Equal(t, "High 500s", p.Name)
	assert.Equal(t, "condition met", p.Text)
	assert.Equal(t, "prod", p.Description)
	assert.Equal(t, "https://superset/alert/1", p.URL)
}

func TestParseJSONWithCharsetParam(t *testing.T) {
	_, err := Parse("application/json; charset=utf-8", []byte(`{"name":"x"}`))
	require.NoError(t, err)
}

func TestParseUnsupportedMediaType(t *testing.T) {
	_, err := Parse("multipart/form-data; boundary=abc", []byte("name=x&text=y"))
	require.ErrorIs(t, err, ErrUnsupportedMediaType)
}

func TestParseEmptyContentType(t *testing.T) {
	_, err := Parse("", []byte(`{"name":"x"}`))
	require.ErrorIs(t, err, ErrUnsupportedMediaType)
}

func TestParseMalformedJSON(t *testing.T) {
	_, err := Parse("application/json", []byte(`{"name": `))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrUnsupportedMediaType)
}
