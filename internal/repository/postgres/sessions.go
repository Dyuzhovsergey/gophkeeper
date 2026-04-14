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
			token,
			expires_at,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5);
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// GetByToken возвращает сессию по токену.
func (r *SessionsRepository) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	const query = `
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions
		WHERE token = $1;
	`

	var session domain.Session

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}

		return nil, fmt.Errorf("select session by token: %w", err)
	}

	return &session, nil
}

// DeleteByToken удаляет сессию по токену.
func (r *SessionsRepository) DeleteByToken(ctx context.Context, token string) error {
	const query = `
		DELETE FROM sessions
		WHERE token = $1;
	`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("delete session by token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows after delete session by token: %w", err)
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
