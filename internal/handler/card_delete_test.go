package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestCardHandlerDeleteByIDNoContent(t *testing.T) {
	svc := &fakeCardService{}
	h := NewCardHandler(svc, discardLogger())
	rec := httptest.NewRecorder()
	req := withUser(httptest.NewRequest(http.MethodDelete, "/api/v1/cards/42", nil))
	req.SetPathValue("id", "42")

	h.DeleteByID(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
	if svc.calls != 1 || svc.gotCardID != 42 || svc.gotUserID != testUserID {
		t.Errorf("service got (calls %d, card %d, user %d), want (1, 42, %d)", svc.calls, svc.gotCardID, svc.gotUserID, testUserID)
	}
	if svc.gotCtx != req.Context() {
		t.Error("service did not receive the request context")
	}
}

func TestCardHandlerDeleteByIDErrors(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
		wantCalls   int
	}{
		{name: "empty id", id: ""},
		{name: "non numeric id", id: "abc"},
		{name: "zero id", id: "0"},
		{name: "negative id", id: "-1"},
		{name: "fractional id", id: "1.5"},
		{name: "overflowing id", id: "9223372036854775808"},
		{
			name: "missing or foreign card", id: "42",
			err:        fmt.Errorf("delete card: %w", domain.ErrNotFound),
			wantStatus: http.StatusNotFound, wantCode: CodeNotFound,
			wantMessage: "карточка не найдена", wantCalls: 1,
		},
		{
			name: "internal error hides database details", id: "42",
			err:        errors.New("database connection failed: private database details"),
			wantStatus: http.StatusInternalServerError, wantCode: CodeInternal,
			wantMessage: msgInternal, wantCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantStatus == 0 {
				tt.wantStatus = http.StatusBadRequest
				tt.wantCode = CodeValidation
				tt.wantMessage = "id: должен быть положительным целым числом"
			}
			svc := &fakeCardService{err: tt.err}
			h := NewCardHandler(svc, discardLogger())
			rec := httptest.NewRecorder()
			req := withUser(httptest.NewRequest(http.MethodDelete, "/api/v1/cards/"+tt.id, nil))
			req.SetPathValue("id", tt.id)

			h.DeleteByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if svc.calls != tt.wantCalls {
				t.Errorf("service called %d times, want %d", svc.calls, tt.wantCalls)
			}
			assertErrorBody(t, rec, tt.wantCode)
			var body errorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Error.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", body.Error.Message, tt.wantMessage)
			}
			if strings.Contains(rec.Body.String(), "private database details") {
				t.Error("response leaks database details")
			}
		})
	}
}

func TestCardHandlerDeleteByIDWithoutUser(t *testing.T) {
	svc := &fakeCardService{}
	h := NewCardHandler(svc, discardLogger())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cards/42", nil)
	req.SetPathValue("id", "42")

	h.DeleteByID(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	assertErrorBody(t, rec, CodeUnauthorized)
	if svc.calls != 0 {
		t.Errorf("service called %d times, want 0", svc.calls)
	}
}
