package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/middleware"
)

func testTokenManager() *auth.TokenManager {
	return auth.NewTokenManager([]byte(testJWTSecret), time.Hour)
}

// TestRouterLogsRoutePattern verifies the logging middleware records the route
// template ("/api/v1/decks/{id}"), not the concrete path. The pattern is only
// available once a request is matched by the real router, so this goes through
// NewRouter rather than the middleware in isolation.
func TestRouterLogsRoutePattern(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	tm := testTokenManager()
	token, err := tm.Issue(1)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	decks := NewDeckHandler(&fakeDeckService{deck: domain.Deck{ID: 7}}, discardLogger())
	health := NewHealthHandler(fakePinger{}, discardLogger())
	router := NewRouter(decks, NewCardHandler(&fakeCardService{}, discardLogger()), health, newTestUserHandler(&fakeUserRegistrar{}), tm, logger)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse log line %q: %v", buf.String(), err)
	}
	if entry["route"] != "/api/v1/decks/{id}" {
		t.Errorf("route = %v, want /api/v1/decks/{id}", entry["route"])
	}
	if entry["path"] != "/api/v1/decks/7" {
		t.Errorf("path = %v, want /api/v1/decks/7", entry["path"])
	}
}

// TestRouterRoundTripsRequestID — приёмочный сценарий: входящий X-Request-ID
// возвращается в ответе и попадает в строку HTTP-лога этого запроса.
func TestRouterRoundTripsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(middleware.NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	decks := NewDeckHandler(&fakeDeckService{deck: domain.Deck{ID: 7}}, logger)
	health := NewHealthHandler(fakePinger{}, logger)
	router := NewRouter(decks, NewCardHandler(&fakeCardService{}, discardLogger()), health, newTestUserHandler(&fakeUserRegistrar{}), testTokenManager(), logger)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil)
	req.Header.Set(middleware.RequestIDHeader, "abc")
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get(middleware.RequestIDHeader); got != "abc" {
		t.Errorf("response %s = %q, want abc", middleware.RequestIDHeader, got)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse log line %q: %v", buf.String(), err)
	}
	if entry["request_id"] != "abc" {
		t.Errorf("log request_id = %v, want abc", entry["request_id"])
	}
}

// TestRouterLogsUserID: у аутентифицированного запроса user_id, проставленный
// глубоко в Auth, доезжает до строки лога (logger стоит выше Auth), а у
// публичного запроса user_id в логе нет.
func TestRouterLogsUserID(t *testing.T) {
	tm := testTokenManager()
	token, err := tm.Issue(42)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	newRouter := func(buf *bytes.Buffer) http.Handler {
		logger := slog.New(slog.NewJSONHandler(buf, nil))
		decks := NewDeckHandler(&fakeDeckService{deck: domain.Deck{ID: 7}}, discardLogger())
		health := NewHealthHandler(fakePinger{}, discardLogger())
		return NewRouter(decks, NewCardHandler(&fakeCardService{}, discardLogger()), health, newTestUserHandler(&fakeUserRegistrar{}), tm, logger)
	}

	t.Run("authenticated request logs user_id", func(t *testing.T) {
		var buf bytes.Buffer
		req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		newRouter(&buf).ServeHTTP(httptest.NewRecorder(), req)

		var entry map[string]any
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("parse log line %q: %v", buf.String(), err)
		}
		// JSON-числа десериализуются в float64.
		if entry["user_id"] != float64(42) {
			t.Errorf("user_id = %v, want 42", entry["user_id"])
		}
	})

	t.Run("public request has no user_id", func(t *testing.T) {
		var buf bytes.Buffer
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		newRouter(&buf).ServeHTTP(httptest.NewRecorder(), req)

		var entry map[string]any
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("parse log line %q: %v", buf.String(), err)
		}
		if _, present := entry["user_id"]; present {
			t.Errorf("user_id present on a public request: %v", entry["user_id"])
		}
	})
}
