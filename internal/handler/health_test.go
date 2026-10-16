package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakePinger implements the pinger interface, returning whatever error it is given.
type fakePinger struct {
	err error
}

func (f fakePinger) Ping(_ context.Context) error {
	return f.err
}

func TestHealthHandlerCheck(t *testing.T) {
	tests := []struct {
		name            string
		pingErr         error
		wantStatus      int
		wantStatusField string
	}{
		{
			name:            "database reachable",
			pingErr:         nil,
			wantStatus:      http.StatusOK,
			wantStatusField: "ok",
		},
		{
			name:            "database unavailable",
			pingErr:         errors.New("dial tcp: connection refused"),
			wantStatus:      http.StatusServiceUnavailable,
			wantStatusField: "unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(fakePinger{err: tt.pingErr}, discardLogger())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)

			h.Check(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["status"] != tt.wantStatusField {
				t.Errorf("status field = %q, want %q", body["status"], tt.wantStatusField)
			}
		})
	}
}
