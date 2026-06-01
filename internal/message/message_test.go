package message

import (
	"strings"
	"testing"

	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
)

func TestRenderComposesHTML(t *testing.T) {
	r := Render(superset.Payload{
		Name:        "High 500s",
		Text:        "condition met",
		Description: "prod",
		URL:         "https://superset/alert/1",
	})

	want := "<b>High 500s</b>\n\ncondition met\n\n<i>prod</i>"
	if r.Text != want {
		t.Errorf("Text = %q, want %q", r.Text, want)
	}
	if r.ButtonURL != "https://superset/alert/1" {
		t.Errorf("ButtonURL = %q, want %q", r.ButtonURL, "https://superset/alert/1")
	}
}

func TestRenderEscapesHTML(t *testing.T) {
	r := Render(superset.Payload{Name: "a<b>&c"})

	if strings.Contains(r.Text, "<b>a<b>&c</b>") {
		t.Fatalf("unescaped content present: %q", r.Text)
	}
	if !strings.Contains(r.Text, "a&lt;b&gt;&amp;c") {
		t.Errorf("Text = %q, want escaped %q", r.Text, "a&lt;b&gt;&amp;c")
	}
}

func TestRenderOmitsEmptyTextAndDescription(t *testing.T) {
	r := Render(superset.Payload{Name: "only name"})

	if r.Text != "<b>only name</b>" {
		t.Errorf("Text = %q, want %q", r.Text, "<b>only name</b>")
	}
}

func TestRenderTruncatesToMax(t *testing.T) {
	r := Render(superset.Payload{Name: strings.Repeat("x", 5000)})

	got := len([]rune(r.Text))
	if got != MaxTextLen {
		t.Errorf("rune length = %d, want %d", got, MaxTextLen)
	}
}
