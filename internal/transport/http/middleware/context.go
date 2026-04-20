package middleware

import (
	"context"

	authservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
)

type contextKey string

const identityContextKey contextKey = "auth_identity"

// ContextWithIdentity сохраняет identity пользователя в context.
func ContextWithIdentity(ctx context.Context, identity *authservice.Identity) context.Context {
	return context.WithValue(ctx, identityContextKey, identity)
}

// IdentityFromContext извлекает identity пользователя из context.
func IdentityFromContext(ctx context.Context) (*authservice.Identity, bool) {
	identity, ok := ctx.Value(identityContextKey).(*authservice.Identity)
	if !ok || identity == nil {
		return nil, false
	}

	return identity, true
}
