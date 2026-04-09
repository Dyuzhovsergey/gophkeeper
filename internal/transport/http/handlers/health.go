// Package handlers предоставляет обработчики приложения
package handlers

import (
	"net/http"
)

// Health обрабатывает запрос проверки доступности сервера.
func Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte("ok"))
}
