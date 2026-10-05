package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Коды ошибок из схемы Error в api/openapi.yaml.
const (
	CodeValidation = "validation_error"
	CodeNotFound   = "not_found"
	CodeInternal   = "internal"
)

const msgInternal = "внутренняя ошибка"

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON пишет v как JSON с указанным статусом.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Заголовок уже отправлен, клиенту ответить нечем, остаётся только лог.
		slog.Error("write json response", "err", err)
	}
}

// WriteError пишет ошибку в едином формате {"error":{"code","message"}}.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorBody{
		Error: errorDetail{
			Code:    code,
			Message: message,
		},
	})
}
