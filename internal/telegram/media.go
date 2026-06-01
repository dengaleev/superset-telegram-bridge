package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
)

// MaxMediaGroup is Telegram's maximum number of items per sendMediaGroup album.
const MaxMediaGroup = 10

// Media is a file to upload to Telegram.
type Media struct {
	Filename string
	MIME     string
	Data     []byte
}

// SendPhoto uploads a single photo with an optional caption.
func (c *Client) SendPhoto(ctx context.Context, chatID, caption string, m Media) error {
	return c.sendSingle(ctx, "sendPhoto", "photo", chatID, caption, m)
}

// SendDocument uploads a single document with an optional caption.
func (c *Client) SendDocument(ctx context.Context, chatID, caption string, m Media) error {
	return c.sendSingle(ctx, "sendDocument", "document", chatID, caption, m)
}

func (c *Client) sendSingle(ctx context.Context, method, field, chatID, caption string, m Media) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("chat_id", chatID)
	_ = mw.WriteField("parse_mode", "HTML")
	if caption != "" {
		_ = mw.WriteField("caption", caption)
	}
	if err := writeFilePart(mw, field, m); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("close multipart: %w", err)
	}
	return c.send(ctx, method, mw.FormDataContentType(), buf.Bytes())
}

// SendMediaGroup uploads 2..MaxMediaGroup files as an album. kind is "photo" or
// "document"; the caption (if any) is attached to the first item only.
func (c *Client) SendMediaGroup(ctx context.Context, chatID, caption, kind string, files []Media) error {
	type item struct {
		Type      string `json:"type"`
		Media     string `json:"media"`
		Caption   string `json:"caption,omitempty"`
		ParseMode string `json:"parse_mode,omitempty"`
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("chat_id", chatID)

	items := make([]item, 0, len(files))
	for i, f := range files {
		it := item{Type: kind, Media: "attach://" + f.Filename}
		if i == 0 && caption != "" {
			it.Caption = caption
			it.ParseMode = "HTML"
		}
		items = append(items, it)
		if err := writeFilePart(mw, f.Filename, f); err != nil {
			return err
		}
	}

	mediaJSON, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal media group: %w", err)
	}
	_ = mw.WriteField("media", string(mediaJSON))
	if err := mw.Close(); err != nil {
		return fmt.Errorf("close multipart: %w", err)
	}
	return c.send(ctx, "sendMediaGroup", mw.FormDataContentType(), buf.Bytes())
}

// writeFilePart writes one file as a multipart part named `field`, carrying its
// real Content-Type so Telegram interprets it correctly. For media groups,
// `field` is the filename so it matches the attach://<filename> reference.
func writeFilePart(mw *multipart.Writer, field string, m Media) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, m.Filename))
	if m.MIME != "" {
		h.Set("Content-Type", m.MIME)
	}
	part, err := mw.CreatePart(h)
	if err != nil {
		return fmt.Errorf("create media part: %w", err)
	}
	if _, err := part.Write(m.Data); err != nil {
		return fmt.Errorf("write media bytes: %w", err)
	}
	return nil
}
