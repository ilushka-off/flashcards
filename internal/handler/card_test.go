package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// fakeCardService implements CardService with canned results for handler tests.
type fakeCardService struct {
	card      domain.Card
	err       error
	calls     int
	gotDeckID int64
	gotUserID int64
	gotInput  domain.CardCreate
}

func (f *fakeCardService) Create(_ context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	f.gotInput = in
	return f.card, f.err
}

func TestCardHandlerCreateCreated(t *testing.T) {
	svc := &fakeCardService{card: domain.Card{ID: 1, DeckID: 7, Front: "uno", Back: "one", EaseFactor: 2.5}}
	h := NewCardHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/v1/decks/7/cards", strings.NewReader(`{"front":"uno","back":"one"}`)))
	req.SetPathValue("id", "7")

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if svc.gotDeckID != 7 || svc.gotUserID != testUserID {
		t.Errorf("service got (deck %d, user %d), want (7, %d)", svc.gotDeckID, svc.gotUserID, testUserID)
	}
	if svc.gotInput.Front != "uno" || svc.gotInput.Back != "one" {
		t.Errorf("service got input %+v, want front uno, back one", svc.gotInput)
	}

	var got domain.Card
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != 1 || got.DeckID != 7 || got.EaseFactor != 2.5 {
		t.Errorf("card = %+v, want ID 1 in deck 7 with ease 2.5", got)
	}
}

func TestCardHandlerCreateErrors(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		body        string
		svc         *fakeCardService
		wantStatus  int
		wantCode    string
		wantMessage string
		wantCalls   int
		// mustNotContain guards against leaking internal error text to the client.
		mustNotContain string
	}{
		{
			name:        "non numeric id",
			id:          "abc",
			body:        `{"front":"uno","back":"one"}`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "id below one",
			id:          "0",
			body:        `{"front":"uno","back":"one"}`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "negative id",
			id:          "-1",
			body:        `{"front":"uno","back":"one"}`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "malformed json",
			id:          "7",
			body:        `{"front":`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "body: невалидный JSON",
		},
		{
			name:        "trailing data after object",
			id:          "7",
			body:        `{"front":"uno","back":"one"}{}`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "body: невалидный JSON",
		},
		{
			// The schedule is set by the server only; a client cannot pass it in.
			name:        "schedule field in body is rejected",
			id:          "7",
			body:        `{"front":"uno","back":"one","ease_factor":5}`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "ease_factor: неизвестное поле",
		},
		{
			name:        "validation error from service",
			id:          "7",
			body:        `{"front":"","back":"one"}`,
			svc:         &fakeCardService{err: &domain.ValidationError{Field: "front", Message: "обязательное поле"}},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "front: обязательное поле",
			wantCalls:   1,
		},
		{
			name:        "deck not found",
			id:          "7",
			body:        `{"front":"uno","back":"one"}`,
			svc:         &fakeCardService{err: domain.ErrNotFound},
			wantStatus:  http.StatusNotFound,
			wantCode:    CodeNotFound,
			wantMessage: "колода не найдена",
			wantCalls:   1,
		},
		{
			name:           "service internal error hides details",
			id:             "7",
			body:           `{"front":"uno","back":"one"}`,
			svc:            &fakeCardService{err: errors.New("pq: password=secret host=10.0.0.1")},
			wantStatus:     http.StatusInternalServerError,
			wantCode:       CodeInternal,
			wantMessage:    msgInternal,
			wantCalls:      1,
			mustNotContain: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewCardHandler(tt.svc, discardLogger())

			rec := httptest.NewRecorder()
			req := withUser(httptest.NewRequest(http.MethodPost, "/api/v1/decks/"+tt.id+"/cards", strings.NewReader(tt.body)))
			req.SetPathValue("id", tt.id)

			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			if tt.svc.calls != tt.wantCalls {
				t.Errorf("service called %d times, want %d", tt.svc.calls, tt.wantCalls)
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

func TestCardHandlerCreateWithoutUser(t *testing.T) {
	svc := &fakeCardService{}
	h := NewCardHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks/7/cards", strings.NewReader(`{"front":"uno","back":"one"}`))
	req.SetPathValue("id", "7")

	h.Create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.calls != 0 {
		t.Errorf("service called %d times, want 0 without user", svc.calls)
	}
}
