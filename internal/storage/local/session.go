package local

import (
	"fmt"
	"strings"
	"time"
)

// Session описывает локально сохранённую клиентскую сессию.
type Session struct {
	// Token — JWT токен доступа.
	Token string `json:"token"`

	// UserID — идентификатор пользователя.
	UserID string `json:"user_id,omitempty"`

	// SessionID — идентификатор серверной сессии.
	SessionID string `json:"session_id,omitempty"`

	// ExpiresAt — срок действия токена.
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Validate проверяет корректность локальной сессии.
func (s Session) Validate() error {
	if strings.TrimSpace(s.Token) == "" {
		return fmt.Errorf("local session token is empty")
	}

	return nil
}
