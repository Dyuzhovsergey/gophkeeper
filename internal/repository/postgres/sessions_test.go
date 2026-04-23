package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

// newSessionsRepoMock создаёт sql.DB с sqlmock и SessionsRepository поверх него.
func newSessionsRepoMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *SessionsRepository) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock, NewSessionsRepository(db)
}

// TestSessionsRepository_Create_Success проверяет успешное создание сессии.
func TestSessionsRepository_Create_Success(t *testing.T) {
	_, mock, repo := newSessionsRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	session := &domain.Session{
		ID:        "session-1",
		UserID:    "user-1",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	mock.ExpectExec(`INSERT INTO sessions`).
		WithArgs(session.ID, session.UserID, session.ExpiresAt, session.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSessionsRepository_GetByID_Success проверяет успешное получение сессии.
func TestSessionsRepository_GetByID_Success(t *testing.T) {
	_, mock, repo := newSessionsRepoMock(t)

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "expires_at", "created_at",
	}).AddRow("session-1", "user-1", now.Add(24*time.Hour), now)

	mock.ExpectQuery(`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = \$1;`).
		WithArgs("session-1").
		WillReturnRows(rows)

	session, err := repo.GetByID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.ID != "session-1" {
		t.Fatalf("unexpected id: got %q, want %q", session.ID, "session-1")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSessionsRepository_GetByID_NotFound проверяет ErrSessionNotFound.
func TestSessionsRepository_GetByID_NotFound(t *testing.T) {
	_, mock, repo := newSessionsRepoMock(t)

	mock.ExpectQuery(`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = \$1;`).
		WithArgs("missing-session").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), "missing-session")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSessionNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSessionsRepository_DeleteByID_Success проверяет успешное удаление одной сессии.
func TestSessionsRepository_DeleteByID_Success(t *testing.T) {
	_, mock, repo := newSessionsRepoMock(t)

	mock.ExpectExec(`DELETE FROM sessions WHERE id = \$1;`).
		WithArgs("session-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteByID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("DeleteByID returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSessionsRepository_DeleteByID_NotFound проверяет ErrSessionNotFound при удалении отсутствующей сессии.
func TestSessionsRepository_DeleteByID_NotFound(t *testing.T) {
	_, mock, repo := newSessionsRepoMock(t)

	mock.ExpectExec(`DELETE FROM sessions WHERE id = \$1;`).
		WithArgs("missing-session").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteByID(context.Background(), "missing-session")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSessionNotFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// TestSessionsRepository_DeleteByUserID_Success проверяет удаление всех сессий пользователя.
func TestSessionsRepository_DeleteByUserID_Success(t *testing.T) {
	_, mock, repo := newSessionsRepoMock(t)

	mock.ExpectExec(`DELETE FROM sessions WHERE user_id = \$1;`).
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := repo.DeleteByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("DeleteByUserID returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
