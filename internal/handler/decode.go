package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// maxBodyBytes ограничивает размер тела запроса. Самый большой валидный
// запрос (колода или карточка) занимает пару килобайт, 1 МБ — с запасом.
const maxBodyBytes = 1 << 20

// decodeJSON читает тело запроса в dst. Ошибки разбора возвращает
// как *domain.ValidationError с текстами из спеки.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		// encoding/json не даёт отдельного типа для неизвестного поля,
		// поэтому имя поля достаём из текста ошибки.
		if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return &domain.ValidationError{Field: strings.Trim(field, `"`), Message: "неизвестное поле"}
		}
		return &domain.ValidationError{Field: "body", Message: "невалидный JSON"}
	}

	// После объекта в теле ничего не должно остаться.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &domain.ValidationError{Field: "body", Message: "невалидный JSON"}
	}
	return nil
}
