package domain

import "errors"

var (
	// ErrUserNotFound означает, что пользователь не найден.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists означает, что пользователь с таким логином уже существует.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrSessionNotFound означает, что сессия не найдена.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSecretNotFound означает, что секрет не найден.
	ErrSecretNotFound = errors.New("secret not found")

	// ErrSecretDeleted означает, что секрет уже помечен как удалённый.
	ErrSecretDeleted = errors.New("secret is deleted")

	// ErrInvalidSecretType означает, что передан неподдерживаемый тип секрета.
	ErrInvalidSecretType = errors.New("invalid secret type")

	// ErrInvalidSecretData означает, что полезная нагрузка секрета некорректна.
	ErrInvalidSecretData = errors.New("invalid secret data")
)
