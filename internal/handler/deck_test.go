package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/userctx"
)

// fakeDeckService implements DeckService with canned results for handler tests.
// It records the ids it was called with so tests can assert that the handler
// passes the authenticated user through.
type fakeDeckService struct {
	deck      domain.Deck
	decks     []domain.Deck
	err       error
	calls     int
	gotDeckID int64
	gotUserID int64
	gotUpdate domain.DeckUpdate
}

func (f *fakeDeckService) Create(_ context.Context, _ domain.DeckCreate, userID int64) (domain.Deck, error) {
	f.calls++
	f.gotUserID = userID
	return f.deck, f.err
}

func (f *fakeDeckService) GetByID(_ context.Context, deckID, userID int64) (domain.Deck, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	return f.deck, f.err
}

func (f *fakeDeckService) GetList(_ context.Context, userID int64) ([]domain.Deck, error) {
	f.calls++
	f.gotUserID = userID
	return f.decks, f.err
}

func (f *fakeDeckService) UpdateByID(_ context.Context, deckID, userID int64, in domain.DeckUpdate) (domain.Deck, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	f.gotUpdate = in
	return f.deck, f.err
}

func (f *fakeDeckService) DeleteByID(_ context.Context, deckID, userID int64) error {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	return f.err
}

// testUserID is the authenticated user that auth-middleware would put in the context.
const testUserID int64 = 42

// withUser attaches testUserID to the request context, as auth-middleware does.
func withUser(r *http.Request) *http.Request {
	return r.WithContext(userctx.WithUserID(r.Context(), testUserID))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDeckHandlerCreateCreated(t *testing.T) {
	svc := &fakeDeckService{deck: domain.Deck{ID: 1, Name: "Spanish", Language: domain.LanguageES}}
	h := NewDeckHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(`{"name":"Spanish","language":"es"}`)))

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if svc.gotUserID != testUserID {
		t.Errorf("service user id = %d, want %d", svc.gotUserID, testUserID)
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
			req := withUser(httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(tt.body)))

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

func TestDeckHandlerGetByIDOK(t *testing.T) {
	svc := &fakeDeckService{deck: domain.Deck{ID: 7, Name: "Spanish", Language: domain.LanguageES}}
	h := NewDeckHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil))
	req.SetPathValue("id", "7")

	h.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.gotDeckID != 7 || svc.gotUserID != testUserID {
		t.Errorf("service got (deck %d, user %d), want (7, %d)", svc.gotDeckID, svc.gotUserID, testUserID)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got domain.Deck
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != 7 || got.Name != "Spanish" || got.Language != domain.LanguageES {
		t.Errorf("deck = %+v, want ID 7 Spanish es", got)
	}
}

func TestDeckHandlerGetByIDErrors(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		svc         *fakeDeckService
		wantStatus  int
		wantCode    string
		wantMessage string
		// mustNotContain guards against leaking internal error text to the client.
		mustNotContain string
	}{
		{
			name:        "not found",
			id:          "1",
			svc:         &fakeDeckService{err: domain.ErrNotFound},
			wantStatus:  http.StatusNotFound,
			wantCode:    CodeNotFound,
			wantMessage: "колода не найдена",
		},
		{
			name:        "non numeric id",
			id:          "abc",
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "id below one",
			id:          "0",
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "negative id",
			id:          "-1",
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:           "service internal error hides details",
			id:             "1",
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
			req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/decks/"+tt.id, nil))
			req.SetPathValue("id", tt.id)

			h.GetByID(rec, req)

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

func TestDeckHandlerGetListOK(t *testing.T) {
	svc := &fakeDeckService{decks: []domain.Deck{
		{ID: 1, Name: "Spanish", Language: domain.LanguageES},
		{ID: 2, Name: "English", Language: domain.LanguageEN},
	}}
	h := NewDeckHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil))

	h.GetList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.gotUserID != testUserID {
		t.Errorf("service user id = %d, want %d", svc.gotUserID, testUserID)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got Items[domain.Deck]
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(got.Items))
	}
	if got.Items[0].ID != 1 || got.Items[1].ID != 2 {
		t.Errorf("items = %+v, want IDs 1 and 2 in order", got.Items)
	}
}

func TestDeckHandlerGetListInternalError(t *testing.T) {
	svc := &fakeDeckService{err: errors.New("pq: password=secret host=10.0.0.1")}
	h := NewDeckHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil))

	h.GetList(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	raw := rec.Body.String()
	if strings.Contains(raw, "secret") {
		t.Errorf("response body leaks internal error: %s", raw)
	}

	var body errorBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("decode body %q: %v", raw, err)
	}
	if body.Error.Code != CodeInternal {
		t.Errorf("code = %q, want %q", body.Error.Code, CodeInternal)
	}
	if body.Error.Message != msgInternal {
		t.Errorf("message = %q, want %q", body.Error.Message, msgInternal)
	}
}

// Without an authenticated user in the context every deck endpoint answers 401
// (who is asking is unknown, unlike 404 "this user has no such deck") and must
// not reach the service with a zero user id.
func TestDeckHandlerWithoutUser(t *testing.T) {
	tests := []struct {
		name string
		call func(h *DeckHandler, w http.ResponseWriter, r *http.Request)
		req  func() *http.Request
	}{
		{
			name: "create",
			call: (*DeckHandler).Create,
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(`{"name":"Spanish","language":"es"}`))
			},
		},
		{
			name: "get by id",
			call: (*DeckHandler).GetByID,
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil)
				r.SetPathValue("id", "7")
				return r
			},
		},
		{
			name: "update by id",
			call: (*DeckHandler).UpdateByID,
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/7", strings.NewReader(`{"name":"Spanish"}`))
				r.SetPathValue("id", "7")
				return r
			},
		},
		{
			name: "delete by id",
			call: (*DeckHandler).DeleteByID,
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodDelete, "/api/v1/decks/7", nil)
				r.SetPathValue("id", "7")
				return r
			},
		},
		{
			name: "get list",
			call: (*DeckHandler).GetList,
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeDeckService{deck: domain.Deck{ID: 7}, decks: []domain.Deck{{ID: 7}}}
			h := NewDeckHandler(svc, discardLogger())

			rec := httptest.NewRecorder()
			tt.call(h, rec, tt.req())

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if svc.calls != 0 {
				t.Errorf("service called %d times, want 0 without a user", svc.calls)
			}

			var body errorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body %q: %v", rec.Body.String(), err)
			}
			if body.Error.Code != CodeUnauthorized {
				t.Errorf("code = %q, want %q", body.Error.Code, CodeUnauthorized)
			}
			if body.Error.Message != "требуется авторизация" {
				t.Errorf("message = %q, want %q", body.Error.Message, "требуется авторизация")
			}
		})
	}
}

// fmtPtr renders an optional field for error messages: nil and "" must be told apart.
func fmtPtr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return strconv.Quote(*p)
}

func strPtr(s string) *string { return &s }

// The handler only decodes the body; it must keep "field absent" (nil) apart
// from "field set to empty" (pointer to "") so the service can tell a no-op
// from clearing the description.
func TestDeckHandlerUpdateByIDOK(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		wantName        *string
		wantDescription *string
	}{
		{"empty object reaches service with both nil", `{}`, nil, nil},
		{"null is the same as absent", `{"name":null,"description":null}`, nil, nil},
		{"only name", `{"name":"Spanish"}`, strPtr("Spanish"), nil},
		{"empty description is kept as empty, not nil", `{"description":""}`, nil, strPtr("")},
		{"both fields", `{"name":"Spanish","description":"colours"}`, strPtr("Spanish"), strPtr("colours")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeDeckService{deck: domain.Deck{ID: 7, Name: "Spanish", Language: domain.LanguageES}}
			h := NewDeckHandler(svc, discardLogger())

			rec := httptest.NewRecorder()
			req := withUser(httptest.NewRequest(http.MethodPatch, "/api/v1/decks/7", strings.NewReader(tt.body)))
			req.SetPathValue("id", "7")

			h.UpdateByID(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if svc.gotDeckID != 7 || svc.gotUserID != testUserID {
				t.Errorf("service got (deck %d, user %d), want (7, %d)", svc.gotDeckID, svc.gotUserID, testUserID)
			}
			if fmtPtr(svc.gotUpdate.Name) != fmtPtr(tt.wantName) {
				t.Errorf("service name = %s, want %s", fmtPtr(svc.gotUpdate.Name), fmtPtr(tt.wantName))
			}
			if fmtPtr(svc.gotUpdate.Description) != fmtPtr(tt.wantDescription) {
				t.Errorf("service description = %s, want %s", fmtPtr(svc.gotUpdate.Description), fmtPtr(tt.wantDescription))
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var got domain.Deck
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.ID != 7 || got.Name != "Spanish" {
				t.Errorf("deck = %+v, want ID 7 Spanish", got)
			}
		})
	}
}

func TestDeckHandlerUpdateByIDErrors(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		body        string
		svc         *fakeDeckService
		wantStatus  int
		wantCode    string
		wantMessage string
		wantCalls   int
		// mustNotContain guards against leaking internal error text to the client.
		mustNotContain string
	}{
		{
			name:        "malformed body",
			id:          "7",
			body:        "{",
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "body: невалидный JSON",
		},
		{
			name:        "unknown field such as language is rejected",
			id:          "7",
			body:        `{"language":"en"}`,
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "language: неизвестное поле",
		},
		{
			name:        "non numeric id",
			id:          "abc",
			body:        `{"name":"Spanish"}`,
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "id below one",
			id:          "0",
			body:        `{"name":"Spanish"}`,
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "service validation error",
			id:          "7",
			body:        `{}`,
			svc:         &fakeDeckService{err: &domain.ValidationError{Field: "body", Message: "нужно передать хотя бы одно поле: name, description"}},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "body: нужно передать хотя бы одно поле: name, description",
			wantCalls:   1,
		},
		{
			name:        "missing or foreign deck is not found",
			id:          "7",
			body:        `{"name":"Spanish"}`,
			svc:         &fakeDeckService{err: domain.ErrNotFound},
			wantStatus:  http.StatusNotFound,
			wantCode:    CodeNotFound,
			wantMessage: "колода не найдена",
			wantCalls:   1,
		},
		{
			name:           "service internal error hides details",
			id:             "7",
			body:           `{"name":"Spanish"}`,
			svc:            &fakeDeckService{err: errors.New("pq: password=secret host=10.0.0.1")},
			wantStatus:     http.StatusInternalServerError,
			wantCode:       CodeInternal,
			wantMessage:    msgInternal,
			wantCalls:      1,
			mustNotContain: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDeckHandler(tt.svc, discardLogger())

			rec := httptest.NewRecorder()
			req := withUser(httptest.NewRequest(http.MethodPatch, "/api/v1/decks/"+tt.id, strings.NewReader(tt.body)))
			req.SetPathValue("id", tt.id)

			h.UpdateByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.svc.calls != tt.wantCalls {
				t.Errorf("service called %d times, want %d", tt.svc.calls, tt.wantCalls)
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

func TestDeckHandlerDeleteByIDNoContent(t *testing.T) {
	svc := &fakeDeckService{}
	h := NewDeckHandler(svc, discardLogger())

	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodDelete, "/api/v1/decks/7", nil))
	req.SetPathValue("id", "7")

	h.DeleteByID(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if svc.gotDeckID != 7 || svc.gotUserID != testUserID {
		t.Errorf("service got (deck %d, user %d), want (7, %d)", svc.gotDeckID, svc.gotUserID, testUserID)
	}
	// 204 must not carry a body: net/http refuses to write one and returns an error.
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204", rec.Body.String())
	}
}

func TestDeckHandlerDeleteByIDErrors(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		svc         *fakeDeckService
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
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "id below one",
			id:          "0",
			svc:         &fakeDeckService{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный id в path запросе",
		},
		{
			name:        "missing or foreign deck is not found",
			id:          "7",
			svc:         &fakeDeckService{err: domain.ErrNotFound},
			wantStatus:  http.StatusNotFound,
			wantCode:    CodeNotFound,
			wantMessage: "колода не найдена",
			wantCalls:   1,
		},
		{
			name:           "service internal error hides details",
			id:             "7",
			svc:            &fakeDeckService{err: errors.New("pq: password=secret host=10.0.0.1")},
			wantStatus:     http.StatusInternalServerError,
			wantCode:       CodeInternal,
			wantMessage:    msgInternal,
			wantCalls:      1,
			mustNotContain: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDeckHandler(tt.svc, discardLogger())

			rec := httptest.NewRecorder()
			req := withUser(httptest.NewRequest(http.MethodDelete, "/api/v1/decks/"+tt.id, nil))
			req.SetPathValue("id", tt.id)

			h.DeleteByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.svc.calls != tt.wantCalls {
				t.Errorf("service called %d times, want %d", tt.svc.calls, tt.wantCalls)
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
