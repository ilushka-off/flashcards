package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDReusesIncomingHeaderAndLogsIt(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	var fromCtx string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		logger.InfoContext(r.Context(), "handled")
		fromCtx = RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "abc")
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get(RequestIDHeader); got != "abc" {
		t.Errorf("response %s = %q, want abc", RequestIDHeader, got)
	}
	if fromCtx != "abc" {
		t.Errorf("context request id = %q, want abc", fromCtx)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse log %q: %v", buf.String(), err)
	}
	if entry["request_id"] != "abc" {
		t.Errorf("log request_id = %v, want abc", entry["request_id"])
	}
}

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	// 16 случайных байт в hex — 32 символа.
	if id := rec.Header().Get(RequestIDHeader); len(id) != 32 {
		t.Errorf("generated id = %q (len %d), want 32 hex chars", id, len(id))
	}
}
