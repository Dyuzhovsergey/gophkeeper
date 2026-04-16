package domain

import (
	"fmt"
	"strings"
	"time"
)

// Session описывает пользовательскую сессию.
type Session struct {
	// ID — уникальный идентификатор сессии.
	ID string

	// UserID — идентификатор владельца сессии.
	UserID string

	// ExpiresAt — время истечения срока действия сессии.
	ExpiresAt time.Time

	// CreatedAt — время создания сессии.
	CreatedAt time.Time
}

// Validate проверяет корректность сессии.
func (s Session) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("session id is empty")
	}

	if strings.TrimSpace(s.UserID) == "" {
		return fmt.Errorf("session user id is empty")
	}

	if s.ExpiresAt.IsZero() {
		return fmt.Errorf("session expires at is zero")
	}

	return nil
}
