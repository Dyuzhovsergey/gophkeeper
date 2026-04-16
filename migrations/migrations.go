// Package migrations содержит миграции схемы базы данных
package migrations

import (
	"context"
	"database/sql"
	"fmt"
)

// Run применяет idempotent-миграции схемы базы данных.
func Run(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migrations transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	statements := []struct {
		name  string
		query string
	}{
		{
			name: "create users table",
			query: `
				CREATE TABLE IF NOT EXISTS users (
					id TEXT PRIMARY KEY,
					login TEXT NOT NULL UNIQUE,
					password_hash TEXT NOT NULL,
					created_at TIMESTAMPTZ NOT NULL,
					updated_at TIMESTAMPTZ NOT NULL
				);
			`,
		},
		{
			name: "create sessions table",
			query: `
				CREATE TABLE IF NOT EXISTS sessions (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					expires_at TIMESTAMPTZ NOT NULL,
					created_at TIMESTAMPTZ NOT NULL
				);
			`,
		},
		{
			name: "drop legacy sessions token column",
			query: `
				ALTER TABLE sessions
				DROP COLUMN IF EXISTS token;
			`,
		},
		{
			name: "create secrets table",
			query: `
				CREATE TABLE IF NOT EXISTS secrets (
					id TEXT PRIMARY KEY,
					owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					type TEXT NOT NULL,
					meta TEXT NOT NULL DEFAULT '',
					payload JSONB NOT NULL,
					version BIGINT NOT NULL DEFAULT 1,
					created_at TIMESTAMPTZ NOT NULL,
					updated_at TIMESTAMPTZ NOT NULL,
					deleted_at TIMESTAMPTZ NULL,
					CONSTRAINT secrets_type_check CHECK (type IN ('credentials', 'text', 'card', 'binary')),
					CONSTRAINT secrets_version_check CHECK (version >= 1)
				);
			`,
		},
		{
			name: "create sessions user index",
			query: `
				CREATE INDEX IF NOT EXISTS idx_sessions_user_id
				ON sessions (user_id);
			`,
		},
		{
			name: "create secrets owner index",
			query: `
				CREATE INDEX IF NOT EXISTS idx_secrets_owner_id
				ON secrets (owner_id);
			`,
		},
		{
			name: "create secrets owner updated_at index",
			query: `
				CREATE INDEX IF NOT EXISTS idx_secrets_owner_updated_at
				ON secrets (owner_id, updated_at);
			`,
		},
		{
			name: "create secrets owner type index",
			query: `
				CREATE INDEX IF NOT EXISTS idx_secrets_owner_type
				ON secrets (owner_id, type);
			`,
		},
		{
			name: "create secrets owner deleted_at index",
			query: `
				CREATE INDEX IF NOT EXISTS idx_secrets_owner_deleted_at
				ON secrets (owner_id, deleted_at);
			`,
		},
		{
			name: "create secrets active sync index",
			query: `
				CREATE INDEX IF NOT EXISTS idx_secrets_owner_updated_at_active
				ON secrets (owner_id, updated_at)
				WHERE deleted_at IS NULL;
			`,
		},
	}

	for _, stmt := range statements {
		if _, err = tx.ExecContext(ctx, stmt.query); err != nil {
			return fmt.Errorf("apply migration %q: %w", stmt.name, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations transaction: %w", err)
	}

	return nil
}
