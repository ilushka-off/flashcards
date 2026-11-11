package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestRouterDeleteCard(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		auth       string
		err        error
		wantStatus int
		wantCode   string
		wantCalls  int
	}{
		{name: "authorized deletion", id: "42", auth: "valid", wantStatus: http.StatusNoContent, wantCalls: 1},
		{name: "missing token", id: "42", wantStatus: http.StatusUnauthorized, wantCode: CodeUnauthorized},
		{name: "invalid token", id: "42", auth: "invalid", wantStatus: http.StatusUnauthorized, wantCode: CodeUnauthorized},
		{name: "invalid card id", id: "abc", auth: "valid", wantStatus: http.StatusBadRequest, wantCode: CodeValidation},
		{name: "missing or foreign card", id: "42", auth: "valid", err: domain.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: CodeNotFound, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards := &fakeCardService{err: tt.err}
			router, tm := newTestRouterWithCards(&fakeDeckService{}, cards)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/cards/"+tt.id, nil)
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

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if cards.calls != tt.wantCalls {
				t.Errorf("service called %d times, want %d", cards.calls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && (cards.gotCardID != 42 || cards.gotUserID != testUserID) {
				t.Errorf("service got (card %d, user %d), want (42, %d)", cards.gotCardID, cards.gotUserID, testUserID)
			}
			if tt.wantStatus == http.StatusNoContent {
				if rec.Body.Len() != 0 {
					t.Errorf("body = %q, want empty", rec.Body.String())
				}
			} else {
				assertErrorBody(t, rec, tt.wantCode)
			}
		})
	}
}
