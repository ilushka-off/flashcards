package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// newTestRouter wires NewRouter with in-memory fakes so routing can be tested
// without a database. The DeckHandler is built on fakeDeckService instead of the
// real service+repository, so no Postgres connection is needed.
func newTestRouter(svc DeckService) http.Handler {
	decks := NewDeckHandler(svc, discardLogger())
	health := NewHealthHandler(fakePinger{}, discardLogger())
	return NewRouter(decks, health, newTestUserHandler(&fakeUserRegistrar{}), discardLogger())
}

func TestRouterGetByIDReachesHandler(t *testing.T) {
	svc := &fakeDeckService{deck: domain.Deck{ID: 7, Name: "Spanish", Language: domain.LanguageES}}
	router := newTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got domain.Deck
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	// A deck in the body proves the {id} route reached GetByID instead of 404.
	if got.ID != 7 {
		t.Errorf("deck ID = %d, want 7", got.ID)
	}
}

func TestRouterUnknownPathReturns404(t *testing.T) {
	router := newTestRouter(&fakeDeckService{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	// Контракт openapi.yaml: любая ошибка, включая 404, — в формате Error.
	assertErrorBody(t, rec, CodeNotFound)
}

// assertErrorBody checks the response uses the unified FC-6 error envelope
// {"error":{"code","message"}} with the expected code and a non-empty message.
func assertErrorBody(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.Error.Code != wantCode {
		t.Errorf("code = %q, want %q", body.Error.Code, wantCode)
	}
	if body.Error.Message == "" {
		t.Error("message is empty, want human-readable text")
	}
}
