package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestCardHandlerUpdateByIDPartial(t *testing.T) {
	tests := []struct {
		name string
		body string
		want domain.CardUpdate
	}{
		{name: "front only", body: `{"front":"hola"}`, want: domain.CardUpdate{Front: strPtr("hola")}},
		{name: "back only", body: `{"back":"привет"}`, want: domain.CardUpdate{Back: strPtr("привет")}},
		{name: "example only", body: `{"example":"Hola, mundo!"}`, want: domain.CardUpdate{Example: strPtr("Hola, mundo!")}},
		{name: "clear example", body: `{"example":""}`, want: domain.CardUpdate{Example: strPtr("")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A complete service result must be returned, including the existing schedule.
			want := domain.Card{
				ID: 7, DeckID: 3, Front: "uno", Back: "один", Example: "uno, dos",
				EaseFactor: 2.15, IntervalDays: 14, Repetitions: 4,
				DueAt:     time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC),
				CreatedAt: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC),
			}
			if tt.want.Front != nil {
				want.Front = *tt.want.Front
			}
			if tt.want.Back != nil {
				want.Back = *tt.want.Back
			}
			if tt.want.Example != nil {
				want.Example = *tt.want.Example
			}
			svc := &fakeCardService{card: want}
			h := NewCardHandler(svc, discardLogger())
			req := withUser(httptest.NewRequest(http.MethodPatch, "/api/v1/cards/7", strings.NewReader(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", "7")
			rec := httptest.NewRecorder()

			h.UpdateByID(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			if svc.calls != 1 || svc.gotCardID != 7 || svc.gotUserID != testUserID {
				t.Errorf("service got (calls %d, card %d, user %d), want (1, 7, %d)", svc.calls, svc.gotCardID, svc.gotUserID, testUserID)
			}
			if !reflect.DeepEqual(svc.gotUpdate, tt.want) {
				t.Errorf("service input = %+v, want %+v", svc.gotUpdate, tt.want)
			}
			if svc.gotCtx != req.Context() {
				t.Error("service did not receive the request context")
			}
			var got domain.Card
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode card: %v", err)
			}
			if got != want {
				t.Errorf("card = %+v, want %+v", got, want)
			}
		})
	}
}

func TestCardHandlerUpdateByIDErrors(t *testing.T) {
	type testCase struct {
		name        string
		id          string
		body        string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
		wantCalls   int
	}
	tests := []testCase{
		{name: "empty id", id: "", body: `{"front":"hola"}`, wantMessage: "id: должен быть положительным целым числом"},
		{name: "invalid id checked before body", id: "abc", body: `{`, wantMessage: "id: должен быть положительным целым числом"},
		{name: "zero id", id: "0", body: `{"front":"hola"}`, wantMessage: "id: должен быть положительным целым числом"},
		{name: "negative id", id: "-1", body: `{"front":"hola"}`, wantMessage: "id: должен быть положительным целым числом"},
		{name: "overflowing id", id: "9223372036854775808", body: `{"front":"hola"}`, wantMessage: "id: должен быть положительным целым числом"},
		{name: "empty body", id: "7", body: "", wantMessage: "body: невалидный JSON"},
		{
			name: "card not found", id: "7", body: `{"front":"hola"}`, err: fmt.Errorf("update card: %w", domain.ErrNotFound),
			wantStatus: http.StatusNotFound, wantCode: CodeNotFound, wantMessage: "карточка не найдена", wantCalls: 1,
		},
	}
	for _, field := range []struct{ name, value string }{
		{name: "deck_id", value: "3"},
		{name: "ease_factor", value: "2.5"},
		{name: "interval_days", value: "14"},
		{name: "repetitions", value: "4"},
		{name: "due_at", value: `"2026-11-25T12:00:00Z"`},
	} {
		tests = append(tests, testCase{
			name: "reject " + field.name, id: "7",
			body:        fmt.Sprintf(`{"front":"hola",%q:%s}`, field.name, field.value),
			wantMessage: field.name + ": неизвестное поле",
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantStatus == 0 {
				tt.wantStatus = http.StatusBadRequest
				tt.wantCode = CodeValidation
			}
			svc := &fakeCardService{err: tt.err}
			h := NewCardHandler(svc, discardLogger())
			req := withUser(httptest.NewRequest(http.MethodPatch, "/api/v1/cards/"+tt.id, strings.NewReader(tt.body)))
			req.SetPathValue("id", tt.id)
			rec := httptest.NewRecorder()

			h.UpdateByID(rec, req)

			assertCardUpdateError(t, rec, tt.wantStatus, tt.wantCode, tt.wantMessage)
			if svc.calls != tt.wantCalls {
				t.Errorf("service called %d times, want %d", svc.calls, tt.wantCalls)
			}
		})
	}
}

func TestCardHandlerUpdateByIDWithoutUser(t *testing.T) {
	svc := &fakeCardService{}
	h := NewCardHandler(svc, discardLogger())
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cards/7", strings.NewReader(`{"front":"hola"}`))
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()

	h.UpdateByID(rec, req)

	assertCardUpdateError(t, rec, http.StatusUnauthorized, CodeUnauthorized, "требуется авторизация")
	if svc.calls != 0 {
		t.Errorf("service called %d times, want 0", svc.calls)
	}
}

func assertCardUpdateError(t *testing.T, rec *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("status = %d, want %d", rec.Code, status)
	}
	assertErrorBody(t, rec, code)
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error.Message != message {
		t.Errorf("message = %q, want %q", body.Error.Message, message)
	}
}
