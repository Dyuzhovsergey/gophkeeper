// Package httptransport предоставляет HTTP-роутер серверного приложения.
package httptransport

import (
	"net/http"

	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/handlers"
	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/middleware"
)

// AuthService объединяет auth-возможности, нужные router.
type AuthService interface {
	handlers.AuthService
	middleware.AuthService
}

// NewRouter создаёт и возвращает базовый HTTP-роутер приложения.
func NewRouter(authService AuthService, vaultService handlers.VaultService) http.Handler {
	mux := http.NewServeMux()

	authHandler := handlers.NewAuthHandler(authService)
	authMiddleware := middleware.NewAuth(authService)
	secretHandler := handlers.NewSecretsHandler(vaultService)

	mux.HandleFunc("/health", handlers.Health)
	mux.HandleFunc("/api/user/register", authHandler.Register)
	mux.HandleFunc("/api/user/login", authHandler.Login)
	mux.Handle("/api/user/me", authMiddleware.RequireAuth(http.HandlerFunc(authHandler.Me)))
	mux.Handle("/api/sync", authMiddleware.RequireAuth(http.HandlerFunc(secretHandler.Sync)))
	mux.Handle("/api/secrets", authMiddleware.RequireAuth(http.HandlerFunc(secretHandler.Collection)))
	mux.Handle("/api/secrets/", authMiddleware.RequireAuth(http.HandlerFunc(secretHandler.Item)))

	return mux
}
