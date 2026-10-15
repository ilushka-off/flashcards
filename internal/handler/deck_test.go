package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// fakeDeckService implements DeckService with canned results for handler tests.
type fakeDeckService struct {
	deck domain.Deck
	err  error
}

func (f *fakeDeckService) Create(_ context.Context, _ domain.DeckCreate) (domain.Deck, error) {
	return f.deck, f.err
}

func (f *fakeDeckService) GetByID(_ context.Context, _ int64) (domain.Deck, error) {
	return f.deck, f.err
}

func (f *fakeDeckService) GetList(_ context.Context) ([]domain.Deck, error) {
	return nil, f.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDeckHandlerCreateCreated(t *testing.T) {
	svc := &fakeDeckService{deck: domain.Deck{ID: 1, Name: "Spanish", Language: domain.LanguageES}}
	h := NewDeckHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(`{"name":"Spanish","language":"es"}`))

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got domain.Deck
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != 1 || got.Name != "Spanish" || got.Language != domain.LanguageES {
		t.Errorf("deck = %+v, want ID 1 Spanish es", got)
	}
}

func TestDeckHandlerCreateErrors(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		svc         *fakeDeckService
		wantStatus  int
		wantCode    string
		wantMessage string
		// mustNotContain guards against leaking internal error text to the client.
		mustNotContain string
	}{
		{
			name:        "malformed body is a validation error",
			body:        "{",
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "body: невалидный JSON",
		},
		{
			name:        "service validation error",
			body:        `{"name":"","language":"es"}`,
			svc:         &fakeDeckService{err: &domain.ValidationError{Field: "name", Message: "обязательное поле"}},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "name: обязательное поле",
		},
		{
			name:           "service internal error hides details",
			body:           `{"name":"Spanish","language":"es"}`,
			svc:            &fakeDeckService{err: errors.New("pq: password=secret host=10.0.0.1")},
			wantStatus:     http.StatusInternalServerError,
			wantCode:       CodeInternal,
			wantMessage:    msgInternal,
			mustNotContain: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDeckHandler(tt.svc, discardLogger())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(tt.body))

			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			raw := rec.Body.String()
			if tt.mustNotContain != "" && strings.Contains(raw, tt.mustNotContain) {
				t.Errorf("response body leaks %q: %s", tt.mustNotContain, raw)
			}

			var body errorBody
			if err := json.Unmarshal([]byte(raw), &body); err != nil {
				t.Fatalf("decode body %q: %v", raw, err)
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
			if body.Error.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", body.Error.Message, tt.wantMessage)
			}
		})
	}
}
