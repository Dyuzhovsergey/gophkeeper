// Package postgres содержит PostgreSQL-реализации репозиториев.
package postgres

import (
	"database/sql"

	repo "github.com/Dyuzhovsergey/gophkeeper/internal/repository"
)

// UsersRepository реализует UserRepository поверх PostgreSQL.
type UsersRepository struct {
	db *sql.DB
}

// SessionsRepository реализует SessionRepository поверх PostgreSQL.
type SessionsRepository struct {
	db *sql.DB
}

// NewUsersRepository создаёт репозиторий пользователей.
func NewUsersRepository(db *sql.DB) *UsersRepository {
	return &UsersRepository{db: db}
}

// NewSessionsRepository создаёт репозиторий сессий.
func NewSessionsRepository(db *sql.DB) *SessionsRepository {
	return &SessionsRepository{db: db}
}

var _ repo.UserRepository = (*UsersRepository)(nil)
var _ repo.SessionRepository = (*SessionsRepository)(nil)