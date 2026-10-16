package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/middleware"
)

func TestRecoverReturns500AndLogsOnce(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(middleware.NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	panics := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})
	h := middleware.RequestID(Recover(logger)(panics))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
	req.Header.Set(middleware.RequestIDHeader, "xyz")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.Error.Code != CodeInternal {
		t.Errorf("code = %q, want %q", body.Error.Code, CodeInternal)
	}

	entry := singleLogLine(t, buf.String())
	if entry["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", entry["level"])
	}
	if entry["request_id"] != "xyz" {
		t.Errorf("request_id = %v, want xyz", entry["request_id"])
	}
}

// TestDeckHandlerLogsInternalErrorOnce защищает политику «одна ошибка — одна
// строка»: при 500 от сервиса handler логирует ровно раз, без дубля на каждом слое.
func TestDeckHandlerLogsInternalErrorOnce(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(middleware.NewContextHandler(slog.NewJSONHandler(&buf, nil)))
	svc := &fakeDeckService{err: errors.New("pq: connection refused to 10.0.0.1")}
	h := NewDeckHandler(svc, logger)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(`{"name":"Spanish","language":"es"}`))
	h.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	entry := singleLogLine(t, buf.String())
	if entry["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", entry["level"])
	}
	// Секреты из текста ошибки не должны утекать в лог сверх самого err-поля.
	if _, ok := entry["request_id"]; ok {
		t.Errorf("request_id must be absent without RequestID middleware, got %v", entry["request_id"])
	}
}

// singleLogLine парсит буфер, убеждаясь, что в нём ровно одна строка лога.
func singleLogLine(t *testing.T, out string) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("log lines = %d, want exactly 1: %q", len(lines), out)
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("parse log %q: %v", lines[0], err)
	}
	return entry
}
