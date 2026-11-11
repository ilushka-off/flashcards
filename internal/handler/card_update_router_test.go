package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/service"
)

// The router uses the real card service so validation must prevent repository calls.
type fakeCardUpdateRepository struct {
	t         *testing.T
	card      domain.Card
	calls     int
	gotCardID int64
	gotUserID int64
}

func (f *fakeCardUpdateRepository) UpdateCardByID(_ context.Context, cardID, userID int64, _ domain.CardUpdate, _ time.Time) (domain.Card, error) {
	f.calls++
	f.gotCardID, f.gotUserID = cardID, userID
	return f.card, nil
}

func (f *fakeCardUpdateRepository) CreateCard(context.Context, int64, int64, domain.CardCreate) (domain.Card, error) {
	f.t.Fatal("unexpected CreateCard call")
	return domain.Card{}, nil
}

func (f *fakeCardUpdateRepository) GetCardList(context.Context, int64, int64, int, int, string) ([]domain.Card, int, error) {
	f.t.Fatal("unexpected GetCardList call")
	return nil, 0, nil
}

func (f *fakeCardUpdateRepository) DeleteByID(context.Context, int64, int64) error {
	f.t.Fatal("unexpected DeleteByID call")
	return nil
}

func TestRouterUpdateCard(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		auth        string
		wantStatus  int
		wantCode    string
		wantMessage string
		wantCalls   int
	}{
		{
			name: "authorized request reaches handler", body: `{"front":"hola"}`, auth: "valid",
			wantStatus: http.StatusOK, wantCalls: 1,
		},
		{
			name: "empty object", body: `{}`, auth: "valid",
			wantStatus: http.StatusBadRequest, wantCode: CodeValidation, wantMessage: "body: нужно передать хотя бы одно поле: front, back, example",
		},
		{
			name: "missing token before invalid body", body: `{`,
			wantStatus: http.StatusUnauthorized, wantCode: CodeUnauthorized, wantMessage: "неверный токен авторизации",
		},
		{
			name: "invalid token", body: `{"front":"hola"}`, auth: "invalid",
			wantStatus: http.StatusUnauthorized, wantCode: CodeUnauthorized, wantMessage: "неверный токен авторизации",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantCard := domain.Card{ID: 7, DeckID: 3, Front: "hola", Back: "привет"}
			repo := &fakeCardUpdateRepository{t: t, card: wantCard}
			router, tm := newTestRouterWithCards(&fakeDeckService{}, service.NewCardService(repo))
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/cards/7", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.auth == "valid" {
				token, err := tm.Issue(testUserID)
				if err != nil {
					t.Fatalf("issue token: %v", err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
			} else if tt.auth == "invalid" {
				req.Header.Set("Authorization", "Bearer invalid-token")
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if repo.calls != tt.wantCalls {
				t.Errorf("repository called %d times, want %d", repo.calls, tt.wantCalls)
			}
			if tt.wantCalls > 0 {
				if repo.gotCardID != 7 || repo.gotUserID != testUserID {
					t.Errorf("repository got (card %d, user %d), want (7, %d)", repo.gotCardID, repo.gotUserID, testUserID)
				}
			}
			if tt.wantStatus != http.StatusOK {
				assertCardUpdateError(t, rec, tt.wantStatus, tt.wantCode, tt.wantMessage)
				return
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var got domain.Card
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode card: %v", err)
			}
			if got != wantCard {
				t.Errorf("card = %+v, want %+v", got, wantCard)
			}
		})
	}
}
