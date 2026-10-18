package config

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	const validDB = "postgres://user:pass@localhost:5432/flashcards"
	const validSecret = "test-secret-at-least-32-bytes-long"

	tests := []struct {
		name         string
		env          map[string]string
		wantErr      error
		wantErrText  string
		wantPort     int
		wantDB       string
		wantLogLevel slog.Level
		// wantLogFormat пустой означает ожидаемый дефолт "json".
		wantLogFormat string
	}{
		{
			name:         "defaults when only database uri and secret are set",
			env:          map[string]string{"DATABASE_URI": validDB, "JWT_SECRET": validSecret},
			wantPort:     8080,
			wantDB:       validDB,
			wantLogLevel: slog.LevelInfo,
		},
		{
			name:          "log format text from environment",
			env:           map[string]string{"DATABASE_URI": validDB, "JWT_SECRET": validSecret, "LOG_FORMAT": "text"},
			wantPort:      8080,
			wantDB:        validDB,
			wantLogLevel:  slog.LevelInfo,
			wantLogFormat: "text",
		},
		{
			name:        "unknown log format",
			env:         map[string]string{"DATABASE_URI": validDB, "LOG_FORMAT": "xml"},
			wantErrText: "LOG_FORMAT",
		},
		{
			name: "all values from environment",
			env: map[string]string{
				"HTTP_PORT":    "9090",
				"DATABASE_URI": validDB,
				"JWT_SECRET":   validSecret,
				"LOG_LEVEL":    "debug",
			},
			wantPort:     9090,
			wantDB:       validDB,
			wantLogLevel: slog.LevelDebug,
		},
		{
			name:         "log level is case insensitive",
			env:          map[string]string{"DATABASE_URI": validDB, "JWT_SECRET": validSecret, "LOG_LEVEL": "WARN"},
			wantPort:     8080,
			wantDB:       validDB,
			wantLogLevel: slog.LevelWarn,
		},
		{
			name:         "lowest valid port",
			env:          map[string]string{"DATABASE_URI": validDB, "JWT_SECRET": validSecret, "HTTP_PORT": "1"},
			wantPort:     1,
			wantDB:       validDB,
			wantLogLevel: slog.LevelInfo,
		},
		{
			name:         "highest valid port",
			env:          map[string]string{"DATABASE_URI": validDB, "JWT_SECRET": validSecret, "HTTP_PORT": "65535"},
			wantPort:     65535,
			wantDB:       validDB,
			wantLogLevel: slog.LevelInfo,
		},
		{
			name:        "non numeric http port",
			env:         map[string]string{"DATABASE_URI": validDB, "HTTP_PORT": "abc"},
			wantErrText: "HTTP_PORT",
		},
		{
			name:        "empty http port",
			env:         map[string]string{"DATABASE_URI": validDB, "HTTP_PORT": ""},
			wantErrText: "HTTP_PORT",
		},
		{
			name:        "http port below range",
			env:         map[string]string{"DATABASE_URI": validDB, "HTTP_PORT": "0"},
			wantErrText: "HTTP_PORT",
		},
		{
			name:        "http port negative",
			env:         map[string]string{"DATABASE_URI": validDB, "HTTP_PORT": "-1"},
			wantErrText: "HTTP_PORT",
		},
		{
			name:        "http port above range",
			env:         map[string]string{"DATABASE_URI": validDB, "HTTP_PORT": "65536"},
			wantErrText: "HTTP_PORT",
		},
		{
			name:        "unknown log level",
			env:         map[string]string{"DATABASE_URI": validDB, "LOG_LEVEL": "verbose"},
			wantErrText: "LOG_LEVEL",
		},
		{
			name:    "missing database uri",
			env:     map[string]string{},
			wantErr: ErrDatabaseURINotFound,
		},
		{
			name:    "empty database uri",
			env:     map[string]string{"DATABASE_URI": ""},
			wantErr: ErrDatabaseURINotFound,
		},
		{
			name:    "missing jwt secret",
			env:     map[string]string{"DATABASE_URI": validDB},
			wantErr: ErrJWTSecretNotFound,
		},
		{
			name:    "empty jwt secret",
			env:     map[string]string{"DATABASE_URI": validDB, "JWT_SECRET": ""},
			wantErr: ErrJWTSecretNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			got, err := Load()

			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Load() error = %v, want errors.Is %v", err, tt.wantErr)
				}
			case tt.wantErrText != "":
				if err == nil {
					t.Fatalf("Load() error = nil, want error containing %q", tt.wantErrText)
				}
				if !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("Load() error = %q, want it to contain %q", err.Error(), tt.wantErrText)
				}
			default:
				if err != nil {
					t.Fatalf("Load() unexpected error = %v", err)
				}
				if got.HTTPPort != tt.wantPort {
					t.Errorf("HTTPPort = %d, want %d", got.HTTPPort, tt.wantPort)
				}
				if got.DatabaseURI != tt.wantDB {
					t.Errorf("DatabaseURI = %q, want %q", got.DatabaseURI, tt.wantDB)
				}
				if got.LogLevel != tt.wantLogLevel {
					t.Errorf("LogLevel = %v, want %v", got.LogLevel, tt.wantLogLevel)
				}
				wantFormat := tt.wantLogFormat
				if wantFormat == "" {
					wantFormat = "json"
				}
				if got.LogFormat != wantFormat {
					t.Errorf("LogFormat = %q, want %q", got.LogFormat, wantFormat)
				}
			}
		})
	}
}

// clearConfigEnv removes every environment variable Load reads so each case
// starts from a clean slate. t.Setenv registers restoration after the test.
func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"HTTP_PORT", "DATABASE_URI", "LOG_LEVEL", "LOG_FORMAT", "JWT_SECRET", "COOKIE_SECURE"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}
}
