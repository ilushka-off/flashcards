package domain

import "time"

// CardCreate — данные для создания карты.
type CardCreate struct {
	Front   string `json:"front"`
	Back    string `json:"back"`
	Example string `json:"example"`
}

// CardUpdate — данные для частичного обновления карты (PATCH). nil означает
// «поле не передано, не менять»; указатель на "" — явное значение, например
// очистка описания.
type CardUpdate struct {
	Front   *string `json:"front"`
	Back    *string `json:"back"`
	Example *string `json:"example"`
}

// Card — информация о карте, как её отдаёт API.
type Card struct {
	ID           int64     `json:"id"`
	DeckID       int64     `json:"deck_id"`
	Front        string    `json:"front"`
	Back         string    `json:"back"`
	Example      string    `json:"example"`
	EaseFactor   float64   `json:"ease_factor"`
	IntervalDays int       `json:"interval_days"`
	Repetitions  int       `json:"repetitions"`
	DueAt        time.Time `json:"due_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
