package local

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// TestSession_Validate_Success проверяет корректную валидацию непустой локальной сессии.
func TestSession_Validate_Success(t *testing.T) {
	session := Session{
		Token: "jwt-token",
	}

	if err := session.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

// TestSession_Validate_EmptyToken проверяет ошибку при пустом токене.
func TestSession_Validate_EmptyToken(t *testing.T) {
	session := Session{}

	if err := session.Validate(); err == nil {
		t.Fatal("expected error for empty token")
	}
}

// TestNewFileStore_Success проверяет создание FileStore с валидным путём.
func TestNewFileStore_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}

	if got := store.Path(); got != path {
		t.Fatalf("unexpected store path: got %q, want %q", got, path)
	}
}

// TestNewFileStore_EmptyPath проверяет ошибку при пустом пути.
func TestNewFileStore_EmptyPath(t *testing.T) {
	_, err := NewFileStore("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

// TestFileStore_SaveAndLoadSession проверяет полный happy path сохранения и загрузки сессии.
func TestFileStore_SaveAndLoadSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	expiresAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	lastSyncAt := time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)

	want := Session{
		Token:      "jwt-token",
		UserID:     "user-1",
		SessionID:  "session-1",
		ExpiresAt:  expiresAt,
		LastSyncAt: lastSyncAt,
	}

	if err := store.SaveSession(want); err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	got, err := store.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}

	if got == nil {
		t.Fatal("expected non-nil loaded session")
	}

	if got.Token != want.Token {
		t.Fatalf("unexpected token: got %q, want %q", got.Token, want.Token)
	}
	if got.UserID != want.UserID {
		t.Fatalf("unexpected user id: got %q, want %q", got.UserID, want.UserID)
	}
	if got.SessionID != want.SessionID {
		t.Fatalf("unexpected session id: got %q, want %q", got.SessionID, want.SessionID)
	}
	if !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("unexpected expires_at: got %v, want %v", got.ExpiresAt, want.ExpiresAt)
	}
	if !got.LastSyncAt.Equal(want.LastSyncAt) {
		t.Fatalf("unexpected last_sync_at: got %v, want %v", got.LastSyncAt, want.LastSyncAt)
	}
}

// TestFileStore_SaveSession_InvalidSession проверяет, что невалидная сессия не сохраняется.
func TestFileStore_SaveSession_InvalidSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	err = store.SaveSession(Session{})
	if err == nil {
		t.Fatal("expected error for invalid session")
	}
}

// TestFileStore_LoadSession_NotFound проверяет корректную ошибку при отсутствии файла сессии.
func TestFileStore_LoadSession_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	_, err = store.LoadSession()
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, ErrSessionNotFound)
	}
}

// TestFileStore_ClearSession_Success проверяет удаление сохранённой сессии.
func TestFileStore_ClearSession_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	if err := store.SaveSession(Session{Token: "jwt-token"}); err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	if err := store.ClearSession(); err != nil {
		t.Fatalf("ClearSession returned error: %v", err)
	}

	_, err = store.LoadSession()
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("unexpected error after clear: got %v, want wrapped %v", err, ErrSessionNotFound)
	}
}

// TestFileStore_ClearSession_NotFound проверяет, что очистка отсутствующей сессии не считается ошибкой.
func TestFileStore_ClearSession_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	if err := store.ClearSession(); err != nil {
		t.Fatalf("ClearSession returned error: %v", err)
	}
}

func TestFileStore_LoadLastSyncAt_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	lastSyncAt := time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)

	err = store.SaveSession(Session{
		Token:      "jwt-token",
		LastSyncAt: lastSyncAt,
	})
	if err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	got, err := store.LoadLastSyncAt()
	if err != nil {
		t.Fatalf("LoadLastSyncAt returned error: %v", err)
	}

	if !got.Equal(lastSyncAt) {
		t.Fatalf("unexpected last_sync_at: got %v, want %v", got, lastSyncAt)
	}
}

func TestFileStore_UpdateLastSyncAt_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	err = store.SaveSession(Session{
		Token: "jwt-token",
	})
	if err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	lastSyncAt := time.Date(2026, 4, 24, 10, 30, 0, 0, time.UTC)

	if err := store.UpdateLastSyncAt(lastSyncAt); err != nil {
		t.Fatalf("UpdateLastSyncAt returned error: %v", err)
	}

	session, err := store.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}

	if !session.LastSyncAt.Equal(lastSyncAt) {
		t.Fatalf("unexpected last_sync_at: got %v, want %v", session.LastSyncAt, lastSyncAt)
	}
}
