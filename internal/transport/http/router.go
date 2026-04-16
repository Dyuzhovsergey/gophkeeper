// Package httptransport предоставляет HTTP-роутер серверного приложения.
package httptransport

import (
	"net/http"

	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/handlers"
)

// NewRouter создаёт и возвращает базовый HTTP-роутер приложения.
func NewRouter(authService handlers.AuthService) http.Handler {
	mux := http.NewServeMux()

	authHandler := handlers.NewAuthHandler(authService)

	mux.HandleFunc("/health", handlers.Health)
	mux.HandleFunc("/api/user/register", authHandler.Register)
	mux.HandleFunc("/api/user/login", authHandler.Login)

	return mux
}
