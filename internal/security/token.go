// Package security предоставляет хэширование паролей и выпуск/проверку токенов доступа.
package security

import (
	"fmt"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// TokenManager описывает выпуск и проверку токенов доступа.
type TokenManager interface {
	// Generate создаёт новый JWT для пользователя и сессии.
	Generate(userID, sessionID string, expiresAt time.Time) (string, error)

	// Parse проверяет JWT и возвращает claims.
	Parse(tokenString string) (*TokenClaims, error)
}

// TokenClaims описывает claims токена доступа.
type TokenClaims struct {
	jwt.RegisteredClaims
}

// JWTManager реализует TokenManager через JWT HS256.
type JWTManager struct {
	secret []byte
}

// NewJWTManager создаёт JWT-менеджер.
func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
	}
}

// Generate создаёт новый JWT для пользователя и сессии.
func (m *JWTManager) Generate(userID, sessionID string, expiresAt time.Time) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("user id is empty")
	}

	if strings.TrimSpace(sessionID) == "" {
		return "", fmt.Errorf("session id is empty")
	}

	if len(m.secret) == 0 {
		return "", fmt.Errorf("jwt secret is empty")
	}

	now := time.Now().UTC()

	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        sessionID,
			ExpiresAt: jwt.NewNumericDate(expiresAt.UTC()),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err)
	}

	return signedToken, nil
}

// Parse проверяет JWT и возвращает claims.
func (m *JWTManager) Parse(tokenString string) (*TokenClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, fmt.Errorf("jwt token is empty")
	}

	if len(m.secret) == 0 {
		return nil, fmt.Errorf("jwt secret is empty")
	}

	claims := &TokenClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, fmt.Errorf("parse jwt token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("jwt token is invalid")
	}

	if strings.TrimSpace(claims.Subject) == "" {
		return nil, fmt.Errorf("jwt subject is empty")
	}

	if strings.TrimSpace(claims.ID) == "" {
		return nil, fmt.Errorf("jwt jti is empty")
	}

	return claims, nil
}

var _ TokenManager = (*JWTManager)(nil)
