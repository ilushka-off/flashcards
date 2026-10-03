package srs

import (
	"fmt"
	"math"
	"time"
)

type Grade string

const (
	GradeAgain Grade = "again"
	GradeHard  Grade = "hard"
	GradeGood  Grade = "good"
	GradeEasy  Grade = "easy"
)

const (
	InitialEase = 2.5
	MinEase     = 1.3
)

// State — расписание карточки, всё, что нужно SM-2.
type State struct {
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	DueAt        time.Time
}

// quality переводит оценку в число q из SM-2.
func quantity(g Grade) (int, error) {
	switch g {
	case GradeAgain:
		return 1, nil
	case GradeHard:
		return 3, nil
	case GradeGood:
		return 4, nil
	case GradeEasy:
		return 5, nil
	}
	return 0, fmt.Errorf("unknown error: %q", g)
}

func Next(s State, g Grade, now time.Time) (State, error) {
	q, err := quantity(g)
	if err != nil {
		return State{}, err
	}

	next := s
	if q < 3 {
		next.Repetitions = 0
		next.IntervalDays = 1
	} else {
		switch s.Repetitions {
		case 0:
			next.IntervalDays = 1
		case 1:
			next.IntervalDays = 6
		default:
			next.IntervalDays = int(math.Round(float64(s.IntervalDays) * s.EaseFactor))
		}
		next.Repetitions = s.Repetitions + 1
	}

	d := float64(5 - q)
	ef := s.EaseFactor + (0.1 - d*(0.08+d*0.02))
	ef = math.Round(ef*100) / 100
	next.EaseFactor = math.Max(ef, MinEase)

	next.DueAt = now.AddDate(0, 0, next.IntervalDays)
	return next, nil
}
