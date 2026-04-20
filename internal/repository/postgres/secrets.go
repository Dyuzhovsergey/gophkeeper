package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

// Create сохраняет новый секрет.
func (r *SecretsRepository) Create(ctx context.Context, item *domain.SecretItem) error {
	payload, err := marshalSecretData(item.Data)
	if err != nil {
		return fmt.Errorf("marshal secret payload: %w", err)
	}

	version := item.Version
	if version < 1 {
		version = 1
	}

	const query = `
		INSERT INTO secrets (
			id,
			owner_id,
			type,
			meta,
			payload,
			version,
			created_at,
			updated_at,
			deleted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		item.ID,
		item.OwnerID,
		item.Type,
		item.Meta,
		payload,
		version,
		item.CreatedAt,
		item.UpdatedAt,
		nullTimePtr(item.DeletedAt),
	)
	if err != nil {
		return fmt.Errorf("insert secret: %w", err)
	}

	return nil
}

// Update обновляет существующий секрет.
func (r *SecretsRepository) Update(ctx context.Context, item *domain.SecretItem) error {
	payload, err := marshalSecretData(item.Data)
	if err != nil {
		return fmt.Errorf("marshal secret payload: %w", err)
	}

	version := item.Version
	if version < 1 {
		version = 1
	}

	const query = `
		UPDATE secrets
		SET
			type = $1,
			meta = $2,
			payload = $3,
			version = $4,
			updated_at = $5
		WHERE id = $6
		  AND owner_id = $7
		  AND deleted_at IS NULL;
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		item.Type,
		item.Meta,
		payload,
		version,
		item.UpdatedAt,
		item.ID,
		item.OwnerID,
	)
	if err != nil {
		return fmt.Errorf("update secret: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows after update secret: %w", err)
	}

	if rowsAffected == 0 {
		return r.resolveSecretStateError(ctx, item.OwnerID, item.ID)
	}

	return nil
}

// GetByID возвращает секрет по идентификатору и владельцу.
func (r *SecretsRepository) GetByID(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
	const query = `
		SELECT
			id,
			owner_id,
			type,
			meta,
			payload,
			version,
			created_at,
			updated_at,
			deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND id = $2;
	`

	item, err := r.scanSecretRow(r.db.QueryRowContext(ctx, query, ownerID, secretID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSecretNotFound
		}

		return nil, fmt.Errorf("select secret by id: %w", err)
	}

	if item.IsDeleted() {
		return nil, domain.ErrSecretDeleted
	}

	return item, nil
}

// ListByOwner возвращает все активные секреты пользователя.
func (r *SecretsRepository) ListByOwner(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
	const query = `
		SELECT
			id,
			owner_id,
			type,
			meta,
			payload,
			version,
			created_at,
			updated_at,
			deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND deleted_at IS NULL
		ORDER BY updated_at DESC, id ASC;
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("select secrets by owner: %w", err)
	}
	defer rows.Close()

	items, err := r.collectSecrets(rows)
	if err != nil {
		return nil, fmt.Errorf("collect secrets by owner: %w", err)
	}

	return items, nil
}

// SoftDelete выполняет мягкое удаление секрета.
func (r *SecretsRepository) SoftDelete(ctx context.Context, ownerID, secretID string, deletedAt time.Time) error {
	const query = `
		UPDATE secrets
		SET
			deleted_at = $1,
			updated_at = $1,
			version = version + 1
		WHERE owner_id = $2
		  AND id = $3
		  AND deleted_at IS NULL;
	`

	result, err := r.db.ExecContext(ctx, query, deletedAt, ownerID, secretID)
	if err != nil {
		return fmt.Errorf("soft delete secret: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows after soft delete secret: %w", err)
	}

	if rowsAffected == 0 {
		return r.resolveSecretStateError(ctx, ownerID, secretID)
	}

	return nil
}

// ListChangesSince возвращает изменения пользователя после указанного времени.
// В результат попадают как активные, так и удалённые записи.
func (r *SecretsRepository) ListChangesSince(ctx context.Context, ownerID string, since time.Time) ([]*domain.SecretItem, error) {
	const query = `
		SELECT
			id,
			owner_id,
			type,
			meta,
			payload,
			version,
			created_at,
			updated_at,
			deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND updated_at > $2
		ORDER BY updated_at ASC, id ASC;
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID, since)
	if err != nil {
		return nil, fmt.Errorf("select secret changes since: %w", err)
	}
	defer rows.Close()

	items, err := r.collectSecrets(rows)
	if err != nil {
		return nil, fmt.Errorf("collect secret changes since: %w", err)
	}

	return items, nil
}

type secretScanner interface {
	Scan(dest ...any) error
}

func (r *SecretsRepository) scanSecretRow(scanner secretScanner) (*domain.SecretItem, error) {
	var (
		item       domain.SecretItem
		secretType string
		payload    []byte
		deletedAt  sql.NullTime
	)

	err := scanner.Scan(
		&item.ID,
		&item.OwnerID,
		&secretType,
		&item.Meta,
		&payload,
		&item.Version,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, err
	}

	item.Type = domain.SecretType(secretType)
	if deletedAt.Valid {
		item.DeletedAt = &deletedAt.Time
	}

	data, err := unmarshalSecretData(item.Type, payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal secret payload: %w", err)
	}
	item.Data = data

	return &item, nil
}

func (r *SecretsRepository) collectSecrets(rows *sql.Rows) ([]*domain.SecretItem, error) {
	var items []*domain.SecretItem

	for rows.Next() {
		item, err := r.scanSecretRow(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secret rows: %w", err)
	}

	return items, nil
}

func (r *SecretsRepository) resolveSecretStateError(ctx context.Context, ownerID, secretID string) error {
	const query = `
		SELECT deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND id = $2;
	`

	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, ownerID, secretID).Scan(&deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrSecretNotFound
		}

		return fmt.Errorf("resolve secret state: %w", err)
	}

	if deletedAt.Valid {
		return domain.ErrSecretDeleted
	}

	return domain.ErrSecretNotFound
}

func marshalSecretData(data domain.SecretData) ([]byte, error) {
	if data == nil {
		return nil, domain.ErrInvalidSecretData
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func unmarshalSecretData(secretType domain.SecretType, payload []byte) (domain.SecretData, error) {
	switch secretType {
	case domain.SecretTypeCredentials:
		var data domain.CredentialData
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, err
		}
		return data, nil

	case domain.SecretTypeText:
		var data domain.TextData
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, err
		}
		return data, nil

	case domain.SecretTypeCard:
		var data domain.CardData
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, err
		}
		return data, nil

	case domain.SecretTypeBinary:
		var data domain.BinaryData
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, err
		}
		return data, nil

	default:
		return nil, domain.ErrInvalidSecretType
	}
}

func nullTimePtr(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{
		Time:  *value,
		Valid: true,
	}
}
