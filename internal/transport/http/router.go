// Package httptransport предоставляет HTTP-роутер серверного приложения.
package httptransport

import (
	"net/http"

	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/handlers"
)

// NewRouter создаёт и возвращает базовый HTTP-роутер приложения.
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handlers.Health)

	return mux
}