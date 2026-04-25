package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

// newUsersRepoMock создаёт sql.DB с sqlmock и UsersRepository поверх него.
func newUsersRepoMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *UsersRepository) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock, NewUsersRepository(db)
}

// TestUsersRepository_Create_Success проверяет успешное создание пользователя.
func TestUsersRepository_Create_Success(t *testing.T) {
	_, mock, repo := newUsersRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	user := &domain.User{
		ID:           "user-1",
		Login:        "sergey",
		PasswordHash: "hashed-password",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(user.ID, user.Login, user.PasswordHash, user.CreatedAt, user.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestUsersRepository_Create_DuplicateUser проверяет маппинг unique violation в ErrUserAlreadyExists.
func TestUsersRepository_Create_DuplicateUser(t *testing.T) {
	_, mock, repo := newUsersRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	user := &domain.User{
		ID:           "user-1",
		Login:        "sergey",
		PasswordHash: "hashed-password",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(user.ID, user.Login, user.PasswordHash, user.CreatedAt, user.UpdatedAt).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err := repo.Create(context.Background(), user)
	if err == nil {
		t.Fatal("expected duplicate user error")
	}
	if !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrUserAlreadyExists)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestUsersRepository_GetByID_Success проверяет успешное получение пользователя по id.
func TestUsersRepository_GetByID_Success(t *testing.T) {
	_, mock, repo := newUsersRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"id", "login", "password_hash", "created_at", "updated_at",
	}).AddRow("user-1", "sergey", "hashed-password", now, now)

	mock.ExpectQuery(`SELECT id, login, password_hash, created_at, updated_at FROM users WHERE id = \$1;`).
		WithArgs("user-1").
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.ID != "user-1" {
		t.Fatalf("unexpected id: got %q, want %q", user.ID, "user-1")
	}
	if user.Login != "sergey" {
		t.Fatalf("unexpected login: got %q, want %q", user.Login, "sergey")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestUsersRepository_GetByID_NotFound проверяет ErrUserNotFound при отсутствии записи.
func TestUsersRepository_GetByID_NotFound(t *testing.T) {
	_, mock, repo := newUsersRepoMock(t)

	mock.ExpectQuery(`SELECT id, login, password_hash, created_at, updated_at FROM users WHERE id = \$1;`).
		WithArgs("missing-user").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), "missing-user")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrUserNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestUsersRepository_GetByLogin_Success проверяет успешное получение пользователя по login.
func TestUsersRepository_GetByLogin_Success(t *testing.T) {
	_, mock, repo := newUsersRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"id", "login", "password_hash", "created_at", "updated_at",
	}).AddRow("user-1", "sergey", "hashed-password", now, now)

	mock.ExpectQuery(`SELECT id, login, password_hash, created_at, updated_at FROM users WHERE login = \$1;`).
		WithArgs("sergey").
		WillReturnRows(rows)

	user, err := repo.GetByLogin(context.Background(), "sergey")
	if err != nil {
		t.Fatalf("GetByLogin returned error: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.Login != "sergey" {
		t.Fatalf("unexpected login: got %q, want %q", user.Login, "sergey")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestUsersRepository_GetByLogin_NotFound проверяет ErrUserNotFound при отсутствии login.
func TestUsersRepository_GetByLogin_NotFound(t *testing.T) {
	_, mock, repo := newUsersRepoMock(t)

	mock.ExpectQuery(`SELECT id, login, password_hash, created_at, updated_at FROM users WHERE login = \$1;`).
		WithArgs("missing-login").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByLogin(context.Background(), "missing-login")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrUserNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
