package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

type decodeTarget struct {
	Name string `json:"name"`
}

func TestDecodeJSONInvalid(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{name: "empty body", body: "", wantField: "body"},
		{name: "malformed json", body: "{", wantField: "body"},
		{name: "unknown field", body: `{"name":"x","bogus":1}`, wantField: "bogus"},
		{name: "second object after the first", body: `{"name":"x"}{"name":"y"}`, wantField: "body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))

			var dst decodeTarget
			err := decodeJSON(rec, req, &dst)

			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("decodeJSON() error = %v, want *domain.ValidationError", err)
			}
			if verr.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", verr.Field, tt.wantField)
			}
		})
	}
}

func TestDecodeJSONBodyTooLarge(t *testing.T) {
	// A JSON value whose body exceeds maxBodyBytes (1 MiB) must be rejected,
	// not read into memory.
	big := `{"name":"` + strings.Repeat("a", (1<<20)+1) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(big))

	var dst decodeTarget
	err := decodeJSON(rec, req, &dst)

	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("decodeJSON() error = %v, want *domain.ValidationError", err)
	}
	if verr.Field != "body" {
		t.Errorf("Field = %q, want body", verr.Field)
	}
}

func TestDecodeJSONValid(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Spanish"}`))

	var dst decodeTarget
	if err := decodeJSON(rec, req, &dst); err != nil {
		t.Fatalf("decodeJSON() unexpected error = %v", err)
	}
	if dst.Name != "Spanish" {
		t.Errorf("Name = %q, want Spanish", dst.Name)
	}
}
