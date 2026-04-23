package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

// newSecretsRepoMock создаёт sql.DB с sqlmock и SecretsRepository поверх него.
func newSecretsRepoMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *SecretsRepository) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock, NewSecretsRepository(db)
}

// mustJSON сериализует payload в JSON для тестовых rows.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()

	payload, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	return payload
}

// TestSecretsRepository_Create_Success проверяет успешное сохранение секрета.
func TestSecretsRepository_Create_Success(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	item := &domain.SecretItem{
		ID:        "secret-1",
		OwnerID:   "user-1",
		Type:      domain.SecretTypeText,
		Meta:      "note",
		Data:      domain.TextData{Text: "hello"},
		Version:   0, // специально, чтобы проверить нормализацию в 1
		CreatedAt: now,
		UpdatedAt: now,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
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
	`)).
		WithArgs(
			item.ID,
			item.OwnerID,
			item.Type,
			item.Meta,
			mustJSON(t, item.Data),
			1,
			item.CreatedAt,
			item.UpdatedAt,
			sql.NullTime{},
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), item)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_Create_InvalidPayload проверяет ошибку marshal для nil payload.
func TestSecretsRepository_Create_InvalidPayload(t *testing.T) {
	_, _, repo := newSecretsRepoMock(t)

	item := &domain.SecretItem{
		ID:      "secret-1",
		OwnerID: "user-1",
		Type:    domain.SecretTypeText,
		Meta:    "note",
		Data:    nil,
	}

	err := repo.Create(context.Background(), item)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if !errors.Is(err, domain.ErrInvalidSecretData) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrInvalidSecretData)
	}
}

// TestSecretsRepository_GetByID_Success проверяет успешное получение секрета.
func TestSecretsRepository_GetByID_Success(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	payload := mustJSON(t, domain.CredentialData{
		Login:    "sergey",
		Password: "qwerty",
	})

	rows := sqlmock.NewRows([]string{
		"id", "owner_id", "type", "meta", "payload", "version", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		"secret-1",
		"user-1",
		string(domain.SecretTypeCredentials),
		"github",
		payload,
		2,
		now,
		now,
		nil,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs("user-1", "secret-1").
		WillReturnRows(rows)

	item, err := repo.GetByID(context.Background(), "user-1", "secret-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.ID != "secret-1" {
		t.Fatalf("unexpected id: got %q, want %q", item.ID, "secret-1")
	}
	if item.Type != domain.SecretTypeCredentials {
		t.Fatalf("unexpected type: got %q, want %q", item.Type, domain.SecretTypeCredentials)
	}

	data, ok := item.Data.(domain.CredentialData)
	if !ok {
		t.Fatalf("unexpected data type: %T", item.Data)
	}
	if data.Login != "sergey" {
		t.Fatalf("unexpected login: got %q, want %q", data.Login, "sergey")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_GetByID_NotFound проверяет ErrSecretNotFound.
func TestSecretsRepository_GetByID_NotFound(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs("user-1", "missing-secret").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), "user-1", "missing-secret")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, domain.ErrSecretNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSecretNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_GetByID_Deleted проверяет ErrSecretDeleted.
func TestSecretsRepository_GetByID_Deleted(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	deletedAt := now.Add(time.Hour)
	payload := mustJSON(t, domain.TextData{Text: "hello"})

	rows := sqlmock.NewRows([]string{
		"id", "owner_id", "type", "meta", "payload", "version", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		"secret-1",
		"user-1",
		string(domain.SecretTypeText),
		"note",
		payload,
		1,
		now,
		now,
		deletedAt,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs("user-1", "secret-1").
		WillReturnRows(rows)

	_, err := repo.GetByID(context.Background(), "user-1", "secret-1")
	if err == nil {
		t.Fatal("expected deleted error")
	}
	if !errors.Is(err, domain.ErrSecretDeleted) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSecretDeleted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_ListByOwner_Success проверяет получение списка активных секретов.
func TestSecretsRepository_ListByOwner_Success(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"id", "owner_id", "type", "meta", "payload", "version", "created_at", "updated_at", "deleted_at",
	}).
		AddRow(
			"secret-1",
			"user-1",
			string(domain.SecretTypeText),
			"note",
			mustJSON(t, domain.TextData{Text: "hello"}),
			1,
			now,
			now,
			nil,
		).
		AddRow(
			"secret-2",
			"user-1",
			string(domain.SecretTypeCard),
			"visa",
			mustJSON(t, domain.CardData{
				Number:      "4111111111111111",
				Cardholder:  "SERGEY DYUZHOV",
				ExpiryMonth: 12,
				ExpiryYear:  2030,
				CVV:         "123",
			}),
			2,
			now,
			now,
			nil,
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs("user-1").
		WillReturnRows(rows)

	items, err := repo.ListByOwner(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListByOwner returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected items len: got %d, want %d", len(items), 2)
	}
	if items[0].ID != "secret-1" {
		t.Fatalf("unexpected first id: got %q, want %q", items[0].ID, "secret-1")
	}
	if items[1].Type != domain.SecretTypeCard {
		t.Fatalf("unexpected second type: got %q, want %q", items[1].Type, domain.SecretTypeCard)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_Update_Success проверяет успешное обновление секрета.
func TestSecretsRepository_Update_Success(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC)

	item := &domain.SecretItem{
		ID:        "secret-1",
		OwnerID:   "user-1",
		Type:      domain.SecretTypeText,
		Meta:      "updated note",
		Data:      domain.TextData{Text: "updated text"},
		Version:   2,
		UpdatedAt: now,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
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
	`)).
		WithArgs(
			item.Type,
			item.Meta,
			mustJSON(t, item.Data),
			item.Version,
			item.UpdatedAt,
			item.ID,
			item.OwnerID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(context.Background(), item)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_Update_NotFound проверяет ErrSecretNotFound через resolveSecretStateError.
func TestSecretsRepository_Update_NotFound(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC)

	item := &domain.SecretItem{
		ID:        "missing-secret",
		OwnerID:   "user-1",
		Type:      domain.SecretTypeText,
		Meta:      "updated note",
		Data:      domain.TextData{Text: "updated text"},
		Version:   2,
		UpdatedAt: now,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
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
	`)).
		WithArgs(
			item.Type,
			item.Meta,
			mustJSON(t, item.Data),
			item.Version,
			item.UpdatedAt,
			item.ID,
			item.OwnerID,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND id = $2;
	`)).
		WithArgs("user-1", "missing-secret").
		WillReturnError(sql.ErrNoRows)

	err := repo.Update(context.Background(), item)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, domain.ErrSecretNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSecretNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_Update_Deleted проверяет ErrSecretDeleted через resolveSecretStateError.
func TestSecretsRepository_Update_Deleted(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC)
	deletedAt := now.Add(time.Hour)

	item := &domain.SecretItem{
		ID:        "deleted-secret",
		OwnerID:   "user-1",
		Type:      domain.SecretTypeText,
		Meta:      "updated note",
		Data:      domain.TextData{Text: "updated text"},
		Version:   2,
		UpdatedAt: now,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
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
	`)).
		WithArgs(
			item.Type,
			item.Meta,
			mustJSON(t, item.Data),
			item.Version,
			item.UpdatedAt,
			item.ID,
			item.OwnerID,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"deleted_at"}).AddRow(deletedAt)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND id = $2;
	`)).
		WithArgs("user-1", "deleted-secret").
		WillReturnRows(rows)

	err := repo.Update(context.Background(), item)
	if err == nil {
		t.Fatal("expected deleted error")
	}
	if !errors.Is(err, domain.ErrSecretDeleted) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSecretDeleted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_SoftDelete_Success проверяет успешное мягкое удаление.
func TestSecretsRepository_SoftDelete_Success(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	deletedAt := time.Date(2026, 4, 23, 14, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE secrets
		SET
			deleted_at = $1,
			updated_at = $1,
			version = version + 1
		WHERE owner_id = $2
		  AND id = $3
		  AND deleted_at IS NULL;
	`)).
		WithArgs(deletedAt, "user-1", "secret-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SoftDelete(context.Background(), "user-1", "secret-1", deletedAt)
	if err != nil {
		t.Fatalf("SoftDelete returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_SoftDelete_NotFound проверяет ErrSecretNotFound при мягком удалении.
func TestSecretsRepository_SoftDelete_NotFound(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	deletedAt := time.Date(2026, 4, 23, 14, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE secrets
		SET
			deleted_at = $1,
			updated_at = $1,
			version = version + 1
		WHERE owner_id = $2
		  AND id = $3
		  AND deleted_at IS NULL;
	`)).
		WithArgs(deletedAt, "user-1", "missing-secret").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND id = $2;
	`)).
		WithArgs("user-1", "missing-secret").
		WillReturnError(sql.ErrNoRows)

	err := repo.SoftDelete(context.Background(), "user-1", "missing-secret", deletedAt)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, domain.ErrSecretNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSecretNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_SoftDelete_Deleted проверяет ErrSecretDeleted при попытке удалить уже удалённый секрет.
func TestSecretsRepository_SoftDelete_Deleted(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 14, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE secrets
		SET
			deleted_at = $1,
			updated_at = $1,
			version = version + 1
		WHERE owner_id = $2
		  AND id = $3
		  AND deleted_at IS NULL;
	`)).
		WithArgs(now, "user-1", "deleted-secret").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"deleted_at"}).AddRow(now)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT deleted_at
		FROM secrets
		WHERE owner_id = $1
		  AND id = $2;
	`)).
		WithArgs("user-1", "deleted-secret").
		WillReturnRows(rows)

	err := repo.SoftDelete(context.Background(), "user-1", "deleted-secret", now)
	if err == nil {
		t.Fatal("expected deleted error")
	}
	if !errors.Is(err, domain.ErrSecretDeleted) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSecretDeleted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSecretsRepository_ListChangesSince_Success проверяет получение изменений после указанного времени.
func TestSecretsRepository_ListChangesSince_Success(t *testing.T) {
	_, mock, repo := newSecretsRepoMock(t)

	now := time.Date(2026, 4, 23, 15, 0, 0, 0, time.UTC)
	since := now.Add(-time.Hour)
	deletedAt := now.Add(-30 * time.Minute)

	rows := sqlmock.NewRows([]string{
		"id", "owner_id", "type", "meta", "payload", "version", "created_at", "updated_at", "deleted_at",
	}).
		AddRow(
			"secret-1",
			"user-1",
			string(domain.SecretTypeText),
			"note",
			mustJSON(t, domain.TextData{Text: "hello"}),
			1,
			now,
			now,
			nil,
		).
		AddRow(
			"secret-2",
			"user-1",
			string(domain.SecretTypeBinary),
			"file",
			mustJSON(t, domain.BinaryData{
				Filename: "hello.txt",
				MIMEType: "text/plain",
				Content:  []byte("hello"),
			}),
			2,
			now,
			now,
			deletedAt,
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs("user-1", since).
		WillReturnRows(rows)

	items, err := repo.ListChangesSince(context.Background(), "user-1", since)
	if err != nil {
		t.Fatalf("ListChangesSince returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected items len: got %d, want %d", len(items), 2)
	}
	if items[1].DeletedAt == nil {
		t.Fatal("expected deleted item in changes list")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestMarshalSecretData_Nil проверяет ErrInvalidSecretData для nil payload.
func TestMarshalSecretData_Nil(t *testing.T) {
	_, err := marshalSecretData(nil)
	if err == nil {
		t.Fatal("expected error for nil secret data")
	}
	if !errors.Is(err, domain.ErrInvalidSecretData) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrInvalidSecretData)
	}
}

// TestUnmarshalSecretData_InvalidType проверяет ErrInvalidSecretType.
func TestUnmarshalSecretData_InvalidType(t *testing.T) {
	_, err := unmarshalSecretData(domain.SecretType("unsupported"), []byte(`{}`))
	if err == nil {
		t.Fatal("expected invalid secret type error")
	}
	if !errors.Is(err, domain.ErrInvalidSecretType) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrInvalidSecretType)
	}
}

// TestNullTimePtr_Nil проверяет пустой NullTime для nil указателя.
func TestNullTimePtr_Nil(t *testing.T) {
	got := nullTimePtr(nil)
	if got.Valid {
		t.Fatal("expected invalid NullTime for nil pointer")
	}
}

// TestNullTimePtr_Value проверяет корректное заполнение NullTime.
func TestNullTimePtr_Value(t *testing.T) {
	value := time.Date(2026, 4, 23, 16, 0, 0, 0, time.UTC)

	got := nullTimePtr(&value)
	if !got.Valid {
		t.Fatal("expected valid NullTime")
	}
	if !got.Time.Equal(value) {
		t.Fatalf("unexpected time: got %v, want %v", got.Time, value)
	}
}
