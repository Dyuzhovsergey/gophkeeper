package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	authservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/response"
)

// AuthService описывает проверку авторизации,
// которая нужна auth middleware.
type AuthService interface {
	// Authenticate проверяет токен и возвращает identity пользователя.
	Authenticate(ctx context.Context, token string) (*authservice.Identity, error)
}

// Auth отвечает за HTTP middleware авторизации.
type Auth struct {
	service AuthService
}

// NewAuth создаёт middleware авторизации.
func NewAuth(service AuthService) *Auth {
	return &Auth{service: service}
}

// RequireAuth пропускает запрос дальше только при валидной авторизации.
func (m *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		identity, err := m.service.Authenticate(r.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrInvalidToken),
				errors.Is(err, domain.ErrUnauthorized),
				errors.Is(err, domain.ErrSessionExpired):
				response.Error(w, http.StatusInternalServerError, "internal server error")
				return

			default:
				response.Error(w, http.StatusInternalServerError, "internal server error")
				return
			}
		}

		ctx := ContextWithIdentity(r.Context(), identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractBearerToken(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", domain.ErrUnauthorized
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", domain.ErrInvalidToken
	}

	scheme := strings.TrimSpace(parts[0])
	token := strings.TrimSpace(parts[1])

	if !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", domain.ErrInvalidToken
	}

	return token, nil
}
