// Package superset defines the inbound Superset webhook contract and parsing.
package superset

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
)

// ErrUnsupportedMediaType is returned by Parse when the Content-Type is not
// application/json. This is the seam where multipart support lands in Phase 3.
var ErrUnsupportedMediaType = errors.New("unsupported media type")

// Payload is the JSON body Superset POSTs when a notification has no attachments.
// Header is kept as raw JSON: it is parsed but not rendered in Phase 1.
type Payload struct {
	Name        string          `json:"name"`
	Text        string          `json:"text"`
	Description string          `json:"description"`
	URL         string          `json:"url"`
	Header      json.RawMessage `json:"header,omitempty"`
}

// Parse decodes a Superset webhook body. It accepts application/json only;
// any other media type returns ErrUnsupportedMediaType. The raw body is passed
// in (already read by the caller) so signature verification can hook in later
// without re-reading the request.
func Parse(contentType string, body []byte) (Payload, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return Payload{}, ErrUnsupportedMediaType
	}

	var p Payload
	if err := json.Unmarshal(body, &p); err != nil {
		return Payload{}, fmt.Errorf("parse json payload: %w", err)
	}
	return p, nil
}
