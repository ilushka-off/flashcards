package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/middleware"
)

// TestRouterLogsRoutePattern verifies the logging middleware records the route
// template ("/api/v1/decks/{id}"), not the concrete path. The pattern is only
// available once a request is matched by the real router, so this goes through
// NewRouter rather than the middleware in isolation.
func TestRouterLogsRoutePattern(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	decks := NewDeckHandler(&fakeDeckService{deck: domain.Deck{ID: 7}}, discardLogger())
	health := NewHealthHandler(fakePinger{}, discardLogger())
	router := NewRouter(decks, health, logger)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/7", nil)
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
	router := NewRouter(decks, health, logger)

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
