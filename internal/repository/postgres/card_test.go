package postgres

import "testing"

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text unchanged", in: "dead", want: "dead"},
		{name: "empty", in: "", want: ""},
		{name: "percent", in: "100%", want: `100\%`},
		{name: "underscore", in: "snake_case", want: `snake\_case`},
		// The backslash is escaped first and only once, so `\%` becomes `\\\%`, not `\\\\%`.
		{name: "backslash", in: `a\b`, want: `a\\b`},
		{name: "backslash before percent", in: `\%`, want: `\\\%`},
		{name: "cyrillic untouched", in: "крайний_срок", want: `крайний\_срок`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeLike(tt.in); got != tt.want {
				t.Errorf("escapeLike(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
