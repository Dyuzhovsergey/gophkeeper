// Package domain содержит основные доменные сущности приложения.
package domain

import (
	"fmt"
	"strings"
	"time"
)

// User описывает пользователя системы.
type User struct {
	// ID — уникальный идентификатор пользователя.
	ID string

	// Login — логин пользователя.
	Login string

	// PasswordHash — хэш пароля пользователя.
	PasswordHash string

	// CreatedAt — время создания пользователя.
	CreatedAt time.Time

	// UpdatedAt — время последнего изменения пользователя.
	UpdatedAt time.Time
}

// Validate проверяет корректность пользовательской сущности.
func (u User) Validate() error {
	if strings.TrimSpace(u.Login) == "" {
		return fmt.Errorf("user login is empty")
	}

	if strings.TrimSpace(u.PasswordHash) == "" {
		return fmt.Errorf("user password hash is empty")
	}
	return nil
}
