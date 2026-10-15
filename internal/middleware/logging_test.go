package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggerLogsRequest(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus float64 // JSON numbers decode to float64
	}{
		{
			name: "explicit WriteHeader is logged",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "body without WriteHeader defaults to 200",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("ok"))
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))

			h := Logger(logger)(tt.handler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/decks/1", nil)
			h.ServeHTTP(rec, req)

			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("parse log line %q: %v", buf.String(), err)
			}
			if entry["http_status_code"] != tt.wantStatus {
				t.Errorf("http_status_code = %v, want %v", entry["http_status_code"], tt.wantStatus)
			}
			if entry["method"] != http.MethodGet {
				t.Errorf("method = %v, want %s", entry["method"], http.MethodGet)
			}
			if entry["path"] != "/api/v1/decks/1" {
				t.Errorf("path = %v, want /api/v1/decks/1", entry["path"])
			}
		})
	}
}
