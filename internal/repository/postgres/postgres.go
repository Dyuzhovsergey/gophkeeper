// Package postgres содержит PostgreSQL-реализации репозиториев.
package postgres

import "database/sql"

// Repository хранит общее подключение к PostgreSQL
// и служит базовым каркасом для следующих репозиториев.
type Repository struct {
	db *sql.DB
}

// New создаёт базовый PostgreSQL-репозиторий.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}
