package domain

import "time"

// Language — изучаемый язык колоды.
type Language string

const (
	LanguageEN Language = "en"
	LanguageES Language = "es"
)

// DeckCreate — данные для создания колоды.
type DeckCreate struct {
	Name        string   `json:"name"`
	Language    Language `json:"language"`
	Description string   `json:"description"`
}

// Deck — колода со счётчиками карточек, как её отдаёт API.
type Deck struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Language    Language  `json:"language"`
	Description string    `json:"description"`
	TotalCards  int       `json:"total_cards"`
	DueCards    int       `json:"due_cards"`
	CreatedAt   time.Time `json:"created_at"`
}
