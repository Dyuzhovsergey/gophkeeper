package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

// Create сохраняет новую сессию.
func (r *SessionsRepository) Create(ctx context.Context, session *domain.Session) error {
	const query = `
		INSERT INTO sessions (
			id,
			user_id,
			expires_at,
			created_at
		)
		VALUES ($1, $2, $3, $4);
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// GetByID возвращает сессию по идентификатору.
func (r *SessionsRepository) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	const query = `
		SELECT id, user_id, expires_at, created_at
		FROM sessions
		WHERE id = $1;
	`

	var session domain.Session

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}

		return nil, fmt.Errorf("select session by id: %w", err)
	}

	return &session, nil
}

// DeleteByID удаляет сессию по идентификатору.
func (r *SessionsRepository) DeleteByID(ctx context.Context, id string) error {
	const query = `
		DELETE FROM sessions
		WHERE id = $1;
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete session by id: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows after delete session by id: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}

// DeleteByUserID удаляет все сессии пользователя.
func (r *SessionsRepository) DeleteByUserID(ctx context.Context, userID string) error {
	const query = `
		DELETE FROM sessions
		WHERE user_id = $1;
	`

	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("delete sessions by user id: %w", err)
	}

	return nil
}
