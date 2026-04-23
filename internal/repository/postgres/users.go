package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

// Create сохраняет нового пользователя.
func (r *UsersRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (
			id,
			login,
			password_hash,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5);
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Login,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}

		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

// GetByID возвращает пользователя по идентификатору.
func (r *UsersRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `
		SELECT id, login, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1;
	`

	var user domain.User

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("select user by id: %w", err)
	}

	return &user, nil
}

// GetByLogin возвращает пользователя по логину.
func (r *UsersRepository) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	const query = `
		SELECT id, login, password_hash, created_at, updated_at
		FROM users
		WHERE login = $1;
	`

	var user domain.User

	err := r.db.QueryRowContext(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("select user by login: %w", err)
	}

	return &user, nil
}