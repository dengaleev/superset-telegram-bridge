// Package message composes a Superset payload into the HTML text Telegram sends.
// It performs no I/O so it can be unit-tested in isolation.
package message

import (
	"strings"

	"github.com/dengaleev/superset-telegram-bridge/internal/superset"
)

// Telegram length limits, in characters.
const (
	TextMaxLen    = 4096 // sendMessage text
	CaptionMaxLen = 1024 // media caption
)

// htmlEscaper escapes the chars Telegram HTML mode needs; the quote keeps URLs
// safe inside the href attribute.
var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;")

// Render composes the HTML body for a payload and appends an "Open in Superset"
// link (when a URL is present), escaping all user content. The body is truncated
// first so the link always survives within maxLen characters.
func Render(p superset.Payload, maxLen int) string {
	var link string
	if p.URL != "" {
		link = "\n\n<a href=\"" + htmlEscaper.Replace(p.URL) + "\">Open in Superset</a>"
	}

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

	body := truncate(b.String(), maxLen-len([]rune(link)))
	return body + link
}

func truncate(s string, limit int) string {
	limit = max(limit, 0)
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit])
}
