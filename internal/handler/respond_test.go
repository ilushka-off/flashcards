package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, http.StatusNotFound, CodeNotFound, "колода не найдена")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	want := `{"error":{"code":"not_found","message":"колода не найдена"}}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestWriteServiceError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "validation error",
			err:         &domain.ValidationError{Field: "name", Message: "обязательное поле"},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "name: обязательное поле",
		},
		{
			name:        "wrapped validation error",
			err:         fmt.Errorf("create deck: %w", &domain.ValidationError{Field: "name", Message: "обязательное поле"}),
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "name: обязательное поле",
		},
		{
			name:        "wrapped not found",
			err:         fmt.Errorf("get deck 42: %w", domain.ErrNotFound),
			wantStatus:  http.StatusNotFound,
			wantCode:    CodeNotFound,
			wantMessage: "колода не найдена",
		},
		{
			name:        "unknown error hides details",
			err:         errors.New("pq: connection refused to 10.0.0.1"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    CodeInternal,
			wantMessage: "внутренняя ошибка",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/decks/42", nil)

			writeServiceError(rec, req, tt.err, "колода не найдена")

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body errorBody
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Error.Code != tt.wantCode || body.Error.Message != tt.wantMessage {
				t.Errorf("error = %+v, want code %q message %q", body.Error, tt.wantCode, tt.wantMessage)
			}
		})
	}
}
