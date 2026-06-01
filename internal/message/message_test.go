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
		want    message.Rendered
	}{
		{
			name:    "composes all fields",
			payload: superset.Payload{Name: "High 500s", Text: "condition met", Description: "prod", URL: "https://superset/alert/1"},
			want:    message.Rendered{Text: "<b>High 500s</b>\n\ncondition met\n\n<i>prod</i>", ButtonURL: "https://superset/alert/1"},
		},
		{
			name:    "escapes html in user content",
			payload: superset.Payload{Name: "a<b>&c"},
			want:    message.Rendered{Text: "<b>a&lt;b&gt;&amp;c</b>"},
		},
		{
			name:    "omits empty text and description",
			payload: superset.Payload{Name: "only name"},
			want:    message.Rendered{Text: "<b>only name</b>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, message.Render(tt.payload))
		})
	}
}

func TestRenderTruncatesToMax(t *testing.T) {
	r := message.Render(superset.Payload{Name: strings.Repeat("x", 5000)})

	assert.Len(t, []rune(r.Text), message.MaxTextLen)
}
