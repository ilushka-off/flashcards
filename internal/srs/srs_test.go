package srs

import (
	"testing"
	"time"
)

func TestNext(t *testing.T) {
	now := time.Date(2026, 10, 3, 7, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		in    State
		grade Grade
		want  State
	}{
		{"новая, good", State{2.5, 0, 0, now}, GradeGood, State{2.5, 1, 1, now.AddDate(0, 0, 1)}},
		{"второй good", State{2.5, 1, 1, now}, GradeGood, State{2.5, 6, 2, now.AddDate(0, 0, 6)}},
		{"третий good", State{2.5, 6, 2, now}, GradeGood, State{2.5, 15, 3, now.AddDate(0, 0, 15)}},
		{"easy со старым EF", State{2.5, 6, 2, now}, GradeEasy, State{2.6, 15, 3, now.AddDate(0, 0, 15)}},
		{"again сбрасывает", State{2.5, 15, 3, now}, GradeAgain, State{1.96, 1, 0, now.AddDate(0, 0, 1)}},
		{"новая, hard", State{2.5, 0, 0, now}, GradeHard, State{2.36, 1, 1, now.AddDate(0, 0, 1)}},
		{"EF не ниже 1.3", State{1.3, 10, 4, now}, GradeHard, State{1.3, 13, 5, now.AddDate(0, 0, 13)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Next(tt.in, tt.grade, now)
			if err != nil {
				t.Fatalf("unknown error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, wait %+v", got, tt.want)
			}
		})
	}
}

func TestNextUnknownGrade(t *testing.T) {
	if _, err := Next(State{EaseFactor: 2.5}, "meh", time.Now()); err == nil {
		t.Fatal("expected an error for an unknown rating")
	}
}
