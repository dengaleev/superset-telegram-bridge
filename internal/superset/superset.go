// Package superset defines the inbound Superset webhook contract and parsing.
package superset

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
)

// ErrUnsupportedMediaType is returned by Parse when the Content-Type is neither
// application/json nor multipart/form-data.
var ErrUnsupportedMediaType = errors.New("unsupported media type")

// Attachment is one file from a multipart webhook (screenshot, CSV, or PDF).
type Attachment struct {
	Filename string
	MIME     string
	Data     []byte
}

// Payload is the Superset webhook body. Header is kept as raw JSON; Files is
// populated only for multipart (attachment) notifications.
type Payload struct {
	Name        string          `json:"name"`
	Text        string          `json:"text"`
	Description string          `json:"description"`
	URL         string          `json:"url"`
	Header      json.RawMessage `json:"header,omitempty"`
	Files       []Attachment    `json:"-"`
}

// Parse decodes a Superset webhook body. application/json yields a text payload;
// multipart/form-data yields fields plus Files. Any other media type returns
// ErrUnsupportedMediaType. The raw body is passed in (already read by the caller)
// so signature verification can hook in later without re-reading the request.
func Parse(contentType string, body []byte) (Payload, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return Payload{}, ErrUnsupportedMediaType
	}
	switch mediaType {
	case "application/json":
		var p Payload
		if err := json.Unmarshal(body, &p); err != nil {
			return Payload{}, fmt.Errorf("parse json payload: %w", err)
		}
		return p, nil
	case "multipart/form-data":
		return parseMultipart(params["boundary"], body)
	default:
		return Payload{}, ErrUnsupportedMediaType
	}
}

func parseMultipart(boundary string, body []byte) (Payload, error) {
	if boundary == "" {
		return Payload{}, errors.New("parse multipart: missing boundary")
	}
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	var p Payload
	for {
		part, err := r.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Payload{}, fmt.Errorf("parse multipart: %w", err)
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return Payload{}, fmt.Errorf("read multipart part: %w", err)
		}
		if part.FileName() != "" {
			p.Files = append(p.Files, Attachment{
				Filename: part.FileName(),
				MIME:     part.Header.Get("Content-Type"),
				Data:     data,
			})
			continue
		}
		// Unknown fields are ignored so new Superset fields don't break parsing.
		switch part.FormName() {
		case "name":
			p.Name = string(data)
		case "text":
			p.Text = string(data)
		case "description":
			p.Description = string(data)
		case "url":
			p.URL = string(data)
		case "header":
			p.Header = json.RawMessage(data)
		}
	}
	return p, nil
}
