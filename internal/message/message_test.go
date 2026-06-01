package message_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dengaleev/superset-telegram-bridge/internal/message"
	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		payload superset.Payload
		want    string
	}{
		{
			name:    "composes all fields with link",
			payload: superset.Payload{Name: "High 500s", Text: "met", Description: "prod", URL: "https://superset/1"},
			want:    "<b>High 500s</b>\n\nmet\n\n<i>prod</i>\n\n<a href=\"https://superset/1\">Open in Superset</a>",
		},
		{
			name:    "escapes html in content and href",
			payload: superset.Payload{Name: "a<b>&c", URL: "https://x/?a=1&b=2"},
			want:    "<b>a&lt;b&gt;&amp;c</b>\n\n<a href=\"https://x/?a=1&amp;b=2\">Open in Superset</a>",
		},
		{
			name:    "no link when url empty",
			payload: superset.Payload{Name: "only name"},
			want:    "<b>only name</b>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, message.Render(tt.payload, message.TextMaxLen))
		})
	}
}

func TestRenderTruncatesBodyButKeepsLink(t *testing.T) {
	p := superset.Payload{Name: strings.Repeat("x", 5000), URL: "https://superset/1"}

	out := message.Render(p, message.CaptionMaxLen)

	assert.LessOrEqual(t, len([]rune(out)), message.CaptionMaxLen)
	assert.True(t, strings.HasSuffix(out, "<a href=\"https://superset/1\">Open in Superset</a>"),
		"link must survive truncation")
}
