package response

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse описывает единый JSON-ответ с ошибкой.
type ErrorResponse struct {
	Error string `json:"error"`
}

// JSON записывает произвольный JSON-ответ.
func JSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}

// Error записывает JSON-ответ с ошибкой.
func Error(w http.ResponseWriter, statusCode int, message string) {
	JSON(w, statusCode, ErrorResponse{
		Error: message,
	})
}
