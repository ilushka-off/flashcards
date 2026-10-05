package domain

import (
	"errors"
	"fmt"
)

// ErrNotFound означает, что запрошенной сущности нет.
// Сервисы оборачивают его с контекстом: fmt.Errorf("get deck %d: %w", id, ErrNotFound).
var ErrNotFound = errors.New("not found")

// ValidationError описывает невалидное поле запроса.
// Message уходит клиенту как есть, поэтому он на русском.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
