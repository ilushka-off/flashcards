package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/domain"
)

const testJWTSecret = "test-secret-at-least-32-bytes-long"

// newTestRouter wires NewRouter with in-memory fakes so routing can be tested
// without a database. The DeckHandler is built on fakeDeckService instead of the
// real service+repository, so no Postgres connection is needed.
func newTestRouter(svc DeckService) (http.Handler, *auth.TokenManager) {
	return newTestRouterWithCards(svc, &fakeCardService{})
}

// newTestRouterWithCards is newTestRouter with a custom CardService, for
// routes under /decks/{id}/cards.
func newTestRouterWithCards(decksSvc DeckService, cardsSvc CardService) (http.Handler, *auth.TokenManager) {
	tm := auth.NewTokenManager([]byte(testJWTSecret), time.Hour)
	decks := NewDeckHandler(decksSvc, discardLogger())
	cards := NewCardHandler(cardsSvc, discardLogger())
	health := NewHealthHandler(fakePinger{}, discardLogger())
	router := NewRouter(decks, cards, health, newTestUserHandler(&fakeUserRegistrar{}), tm, discardLogger())
	return router, tm
}

func TestRouterGetByIDReachesHandler(t *testing.T) {
	svc := &fakeDeckService{deck: domain.Deck{ID: 7, Name: "Spanish", Language: domain.LanguageES}}
	router, tm := newTestRouter(svc)
	token, err := tm.Issue(1)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
	router, _ := newTestRouter(&fakeDeckService{})

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

func TestRouterCreateCardReachesHandler(t *testing.T) {
	cards := &fakeCardService{card: domain.Card{ID: 3, DeckID: 7, Front: "uno", Back: "one"}}
	router, tm := newTestRouterWithCards(&fakeDeckService{}, cards)
	token, err := tm.Issue(1)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks/7/cards", strings.NewReader(`{"front":"uno","back":"one"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	// The {id} from the path and the user from the token must reach the service.
	if cards.gotDeckID != 7 || cards.gotUserID != 1 {
		t.Errorf("service got (deck %d, user %d), want (7, 1)", cards.gotDeckID, cards.gotUserID)
	}
}

func TestRouterCreateCardRequiresAuth(t *testing.T) {
	cards := &fakeCardService{}
	router, _ := newTestRouterWithCards(&fakeDeckService{}, cards)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks/7/cards", strings.NewReader(`{"front":"uno","back":"one"}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	assertErrorBody(t, rec, CodeUnauthorized)
	if cards.calls != 0 {
		t.Errorf("service called %d times, want 0 without token", cards.calls)
	}
}
