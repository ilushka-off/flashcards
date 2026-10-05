package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// writeServiceError переводит ошибку сервиса в HTTP-ответ.
// notFoundMsg — текст для клиента, если сущность не найдена, например "колода не найдена".
func writeServiceError(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) {
	var verr *domain.ValidationError
	switch {
	case errors.As(err, &verr):
		WriteError(w, http.StatusBadRequest, CodeValidation, verr.Error())
	case errors.Is(err, domain.ErrNotFound):
		WriteError(w, http.StatusNotFound, CodeNotFound, notFoundMsg)
	default:
		slog.Error("unhandled service error", "method", r.Method, "path", r.URL.Path, "err", err)
		WriteError(w, http.StatusInternalServerError, CodeInternal, msgInternal)
	}
}
