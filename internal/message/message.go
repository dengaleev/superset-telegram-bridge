// Package message composes a Superset payload into the HTML text Telegram sends.
// It performs no I/O so it can be unit-tested in isolation.
package message

import (
	"strings"

	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
)

// MaxTextLen is Telegram's sendMessage text limit, in characters.
const MaxTextLen = 4096

// htmlEscaper escapes only the characters Telegram's HTML parse mode requires.
var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// Rendered is the result of composing a payload: the HTML message body and the
// URL for the inline "Open in Superset" button (empty if the payload had none).
type Rendered struct {
	Text      string
	ButtonURL string
}

// Render composes the HTML body from a payload, escaping user content and
// truncating to MaxTextLen characters.
func Render(p superset.Payload) Rendered {
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(htmlEscaper.Replace(p.Name))
	b.WriteString("</b>")

	if p.Text != "" {
		b.WriteString("\n\n")
		b.WriteString(htmlEscaper.Replace(p.Text))
	}
	if p.Description != "" {
		b.WriteString("\n\n<i>")
		b.WriteString(htmlEscaper.Replace(p.Description))
		b.WriteString("</i>")
	}

	return Rendered{
		Text:      truncate(b.String(), MaxTextLen),
		ButtonURL: p.URL,
	}
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
