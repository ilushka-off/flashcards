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

	// GetList: canned page and what the handler passed in.
	cards     []domain.Card
	total     int
	gotLimit  int
	gotOffset int
	gotQ      string
}

func (f *fakeCardService) Create(_ context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	f.gotInput = in
	return f.card, f.err
}

func (f *fakeCardService) GetList(_ context.Context, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	f.gotLimit = limit
	f.gotOffset = offset
	f.gotQ = q
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.cards, f.total, nil
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
			wantMessage: "id: должен быть положительным целым числом",
		},
		{
			name:        "id below one",
			id:          "0",
			body:        `{"front":"uno","back":"one"}`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "id: должен быть положительным целым числом",
		},
		{
			name:        "negative id",
			id:          "-1",
			body:        `{"front":"uno","back":"one"}`,
			svc:         &fakeCardService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "id: должен быть положительным целым числом",
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

func TestCardHandlerGetListOK(t *testing.T) {
	svc := &fakeCardService{
		cards: []domain.Card{{ID: 7, DeckID: 3, Front: "deadline", Back: "крайний срок"}},
		total: 41,
	}
	h := NewCardHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/decks/3/cards?limit=5&offset=10&q=dead", nil))
	req.SetPathValue("id", "3")

	h.GetList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.gotDeckID != 3 || svc.gotUserID != testUserID {
		t.Errorf("service got (deck %d, user %d), want (3, %d)", svc.gotDeckID, svc.gotUserID, testUserID)
	}
	if svc.gotLimit != 5 || svc.gotOffset != 10 || svc.gotQ != "dead" {
		t.Errorf("service got (limit %d, offset %d, q %q), want (5, 10, %q)", svc.gotLimit, svc.gotOffset, svc.gotQ, "dead")
	}

	var got struct {
		Items []domain.Card `json:"items"`
		Total *int          `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != 7 {
		t.Errorf("items = %+v, want one card with ID 7", got.Items)
	}
	if got.Total == nil || *got.Total != 41 {
		t.Errorf("total = %v, want 41", got.Total)
	}
}

func TestCardHandlerGetListDefaults(t *testing.T) {
	svc := &fakeCardService{cards: []domain.Card{}}
	h := NewCardHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/decks/3/cards", nil))
	req.SetPathValue("id", "3")

	h.GetList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.gotLimit != 20 || svc.gotOffset != 0 || svc.gotQ != "" {
		t.Errorf("service got (limit %d, offset %d, q %q), want (20, 0, \"\")", svc.gotLimit, svc.gotOffset, svc.gotQ)
	}
}

// An empty result must be [] with total 0, not null or a missing total:
// the client tells "nothing found" from "deck not found" by this 200.
func TestCardHandlerGetListEmpty(t *testing.T) {
	svc := &fakeCardService{cards: []domain.Card{}, total: 0}
	h := NewCardHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/decks/3/cards?q=zzz", nil))
	req.SetPathValue("id", "3")

	h.GetList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"items":[],"total":0}` {
		t.Errorf("body = %s, want {\"items\":[],\"total\":0}", body)
	}
}

func TestCardHandlerGetListErrors(t *testing.T) {
	const (
		msgID     = "id: должен быть положительным целым числом"
		msgLimit  = "limit: целое число от 1 до 100"
		msgOffset = "offset: целое число не меньше 0"
	)
	tests := []struct {
		name        string
		id          string
		query       string
		svc         *fakeCardService
		wantStatus  int
		wantCode    string
		wantMessage string
		wantCalls   int
		// mustNotContain guards against leaking internal error text to the client.
		mustNotContain string
	}{
		{name: "non numeric id", id: "abc", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgID},
		{name: "id below one", id: "0", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgID},
		{name: "non numeric limit", id: "3", query: "limit=abc", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgLimit},
		{name: "zero limit", id: "3", query: "limit=0", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgLimit},
		{name: "negative limit", id: "3", query: "limit=-1", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgLimit},
		{name: "limit above max", id: "3", query: "limit=101", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgLimit},
		{name: "non numeric offset", id: "3", query: "offset=abc", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgOffset},
		{name: "negative offset", id: "3", query: "offset=-1", svc: &fakeCardService{},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: msgOffset},
		{name: "validation error from service", id: "3", query: "q=long",
			svc:        &fakeCardService{err: &domain.ValidationError{Field: "q", Message: "не длиннее 100 символов"}},
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: "q: не длиннее 100 символов", wantCalls: 1},
		{name: "deck not found", id: "3", svc: &fakeCardService{err: domain.ErrNotFound},
			wantStatus: http.StatusNotFound, wantCode: CodeNotFound, wantMessage: "колода не найдена", wantCalls: 1},
		{name: "service internal error hides details", id: "3",
			svc:        &fakeCardService{err: errors.New("pq: password=secret host=10.0.0.1")},
			wantStatus: http.StatusInternalServerError, wantCode: CodeInternal, wantMessage: msgInternal, wantCalls: 1,
			mustNotContain: "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewCardHandler(tt.svc, discardLogger())

			url := "/api/v1/decks/" + tt.id + "/cards"
			if tt.query != "" {
				url += "?" + tt.query
			}
			rec := httptest.NewRecorder()
			req := withUser(httptest.NewRequest(http.MethodGet, url, nil))
			req.SetPathValue("id", tt.id)

			h.GetList(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
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

func TestCardHandlerGetListWithoutUser(t *testing.T) {
	svc := &fakeCardService{}
	h := NewCardHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/3/cards", nil)
	req.SetPathValue("id", "3")

	h.GetList(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.calls != 0 {
		t.Errorf("service called %d times, want 0 without user", svc.calls)
	}
}
