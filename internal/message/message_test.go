package message

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
)

func TestRenderComposesHTML(t *testing.T) {
	r := Render(superset.Payload{
		Name:        "High 500s",
		Text:        "condition met",
		Description: "prod",
		URL:         "https://superset/alert/1",
	})

	assert.Equal(t, "<b>High 500s</b>\n\ncondition met\n\n<i>prod</i>", r.Text)
	assert.Equal(t, "https://superset/alert/1", r.ButtonURL)
}

func TestRenderEscapesHTML(t *testing.T) {
	r := Render(superset.Payload{Name: "a<b>&c"})

	assert.Contains(t, r.Text, "a&lt;b&gt;&amp;c")
	assert.NotContains(t, r.Text, "a<b>&c")
}

func TestRenderOmitsEmptyTextAndDescription(t *testing.T) {
	r := Render(superset.Payload{Name: "only name"})

	assert.Equal(t, "<b>only name</b>", r.Text)
}

func TestRenderTruncatesToMax(t *testing.T) {
	r := Render(superset.Payload{Name: strings.Repeat("x", 5000)})

	assert.Len(t, []rune(r.Text), MaxTextLen)
}
