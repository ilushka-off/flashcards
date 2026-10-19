package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/service"
)

// fakeUserRegistrar implements UserRegistrar with canned results for handler tests.
type fakeUserRegistrar struct {
	user     domain.User
	token    string
	err      error
	pair     domain.TokenPair
	loginErr error
}

func (f *fakeUserRegistrar) Register(_ context.Context, _ domain.UserCreate) (domain.User, string, error) {
	return f.user, f.token, f.err
}

func (f *fakeUserRegistrar) Login(_ context.Context, _, _ string) (domain.TokenPair, error) {
	return f.pair, f.loginErr
}
func (f *fakeUserRegistrar) GetUserByID(_ context.Context, _ int64) (domain.User, error) {
	return f.user, f.err
}

func newTestUserHandler(reg UserRegistrar) *UserHandler {
	return NewUserHandler(reg, 15*time.Minute, false, discardLogger())
}

const validRegisterBody = `{"email":"ilya@mail.ru","password":"password123"}`

func TestUserHandlerRegisterCreated(t *testing.T) {
	reg := &fakeUserRegistrar{
		user:  domain.User{ID: 1, Email: "ilya@mail.ru"},
		token: "signed-token",
	}
	h := newTestUserHandler(reg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(validRegisterBody))

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	// Authorization — заголовок запроса; сервер его в ответе возвращать не должен.
	if got := rec.Header().Get("Authorization"); got != "" {
		t.Errorf("response Authorization = %q, want it absent", got)
	}

	var got domain.User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != 1 || got.Email != "ilya@mail.ru" {
		t.Errorf("user = %+v, want ID 1 ilya@mail.ru", got)
	}

	cookies := rec.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			tokenCookie = c
		}
	}
	if tokenCookie == nil {
		t.Fatalf("no %q cookie set, got %v", auth.CookieName, cookies)
	}
	if tokenCookie.Value != "signed-token" {
		t.Errorf("cookie value = %q, want %q", tokenCookie.Value, "signed-token")
	}
	if !tokenCookie.HttpOnly {
		t.Error("cookie is not HttpOnly")
	}
	if tokenCookie.MaxAge != int((15 * time.Minute).Seconds()) {
		t.Errorf("cookie MaxAge = %d, want %d", tokenCookie.MaxAge, int((15 * time.Minute).Seconds()))
	}
}

func TestUserHandlerRegisterErrors(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		reg         *fakeUserRegistrar
		wantStatus  int
		wantCode    string
		wantMessage string
		// mustNotContain guards against leaking internal error text to the client.
		mustNotContain string
	}{
		{
			name:        "malformed body is a validation error",
			body:        "{",
			reg:         &fakeUserRegistrar{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "body: невалидный JSON",
		},
		{
			name:        "unknown field is rejected",
			body:        `{"email":"ilya@mail.ru","password":"password123","role":"admin"}`,
			reg:         &fakeUserRegistrar{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "role: неизвестное поле",
		},
		{
			name:        "service invalid input",
			body:        validRegisterBody,
			reg:         &fakeUserRegistrar{err: service.ErrInvalidInput},
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantMessage: "некорректный email или пароль",
		},
		{
			name:        "email already taken is a conflict",
			body:        validRegisterBody,
			reg:         &fakeUserRegistrar{err: service.ErrEmailTaken},
			wantStatus:  http.StatusConflict,
			wantCode:    CodeConflict,
			wantMessage: "email: уже зарегистрирован",
		},
		{
			name:           "internal error hides details",
			body:           validRegisterBody,
			reg:            &fakeUserRegistrar{err: errors.New("pq: password=secret host=10.0.0.1")},
			wantStatus:     http.StatusInternalServerError,
			wantCode:       CodeInternal,
			wantMessage:    msgInternal,
			mustNotContain: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestUserHandler(tt.reg)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tt.body))

			h.Register(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			raw := rec.Body.String()
			if tt.mustNotContain != "" && strings.Contains(raw, tt.mustNotContain) {
				t.Errorf("response leaks %q: %s", tt.mustNotContain, raw)
			}

			var body errorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body %q: %v", raw, err)
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
			if body.Error.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", body.Error.Message, tt.wantMessage)
			}
			// No auth cookie should be set on any error path.
			if cookies := rec.Result().Cookies(); len(cookies) != 0 {
				t.Errorf("error response set cookies %v, want none", cookies)
			}
		})
	}
}

func TestUserHandlerLoginOK(t *testing.T) {
	reg := &fakeUserRegistrar{pair: domain.TokenPair{AccessToken: "access-jwt", RefreshToken: "opaque-refresh"}}
	h := newTestUserHandler(reg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(validRegisterBody))

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got domain.TokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.AccessToken != "access-jwt" || got.RefreshToken != "opaque-refresh" {
		t.Errorf("tokens = %+v, want access-jwt / opaque-refresh", got)
	}
	if got.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", got.TokenType)
	}
	if got.ExpiresIn != int((15 * time.Minute).Seconds()) {
		t.Errorf("expires_in = %d, want %d", got.ExpiresIn, int((15 * time.Minute).Seconds()))
	}
}

func TestUserHandlerLoginErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		reg        *fakeUserRegistrar
		wantStatus int
		wantCode   string
	}{
		{
			name:       "malformed body is a validation error",
			body:       "{",
			reg:        &fakeUserRegistrar{},
			wantStatus: http.StatusBadRequest,
			wantCode:   CodeValidation,
		},
		{
			name:       "invalid input",
			body:       validRegisterBody,
			reg:        &fakeUserRegistrar{loginErr: service.ErrInvalidInput},
			wantStatus: http.StatusBadRequest,
			wantCode:   CodeValidation,
		},
		{
			name:       "wrong credentials are unauthorized",
			body:       validRegisterBody,
			reg:        &fakeUserRegistrar{loginErr: service.ErrInvalidCredentials},
			wantStatus: http.StatusUnauthorized,
			wantCode:   CodeUnauthorized,
		},
		{
			name:       "internal error",
			body:       validRegisterBody,
			reg:        &fakeUserRegistrar{loginErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
			wantCode:   CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestUserHandler(tt.reg)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.body))

			h.Login(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body errorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body %q: %v", rec.Body.String(), err)
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
		})
	}
}
