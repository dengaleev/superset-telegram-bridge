package superset

import (
	"errors"
	"testing"
)

func TestParseValidJSON(t *testing.T) {
	body := []byte(`{"name":"High 500s","text":"condition met","description":"prod","url":"https://superset/alert/1","header":{"k":"v"}}`)

	p, err := Parse("application/json", body)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if p.Name != "High 500s" {
		t.Errorf("Name = %q, want %q", p.Name, "High 500s")
	}
	if p.Text != "condition met" {
		t.Errorf("Text = %q, want %q", p.Text, "condition met")
	}
	if p.URL != "https://superset/alert/1" {
		t.Errorf("URL = %q, want %q", p.URL, "https://superset/alert/1")
	}
}

func TestParseJSONWithCharsetParam(t *testing.T) {
	body := []byte(`{"name":"x"}`)

	if _, err := Parse("application/json; charset=utf-8", body); err != nil {
		t.Fatalf("Parse() error = %v, want nil for json with charset param", err)
	}
}

func TestParseUnsupportedMediaType(t *testing.T) {
	body := []byte("name=x&text=y")

	_, err := Parse("multipart/form-data; boundary=abc", body)
	if !errors.Is(err, ErrUnsupportedMediaType) {
		t.Fatalf("Parse() error = %v, want ErrUnsupportedMediaType", err)
	}
}

func TestParseEmptyContentType(t *testing.T) {
	_, err := Parse("", []byte(`{"name":"x"}`))
	if !errors.Is(err, ErrUnsupportedMediaType) {
		t.Fatalf("Parse() error = %v, want ErrUnsupportedMediaType", err)
	}
}

func TestParseMalformedJSON(t *testing.T) {
	body := []byte(`{"name": `)

	_, err := Parse("application/json", body)
	if err == nil {
		t.Fatal("Parse() error = nil, want parse error")
	}
	if errors.Is(err, ErrUnsupportedMediaType) {
		t.Fatal("Parse() returned ErrUnsupportedMediaType, want a JSON parse error")
	}
}
