package vault

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

type secretRepositoryStub struct {
	createFn           func(ctx context.Context, item *domain.SecretItem) error
	updateFn           func(ctx context.Context, item *domain.SecretItem) error
	getByIDFn          func(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error)
	listByOwnerFn      func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error)
	softDeleteFn       func(ctx context.Context, ownerID, secretID string, deletedAt time.Time) error
	listChangesSinceFn func(ctx context.Context, ownerID string, since time.Time) ([]*domain.SecretItem, error)
}

func (s *secretRepositoryStub) Create(ctx context.Context, item *domain.SecretItem) error {
	return s.createFn(ctx, item)
}

func (s *secretRepositoryStub) Update(ctx context.Context, item *domain.SecretItem) error {
	return s.updateFn(ctx, item)
}

func (s *secretRepositoryStub) GetByID(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
	return s.getByIDFn(ctx, ownerID, secretID)
}

func (s *secretRepositoryStub) ListByOwner(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
	return s.listByOwnerFn(ctx, ownerID)
}

func (s *secretRepositoryStub) SoftDelete(ctx context.Context, ownerID, secretID string, deletedAt time.Time) error {
	return s.softDeleteFn(ctx, ownerID, secretID, deletedAt)
}

func (s *secretRepositoryStub) ListChangesSince(ctx context.Context, ownerID string, since time.Time) ([]*domain.SecretItem, error) {
	return s.listChangesSinceFn(ctx, ownerID, since)
}

func TestService_CreateCredentials_Success(t *testing.T) {
	now := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)

	repo := &secretRepositoryStub{
		createFn: func(ctx context.Context, item *domain.SecretItem) error {
			if item.OwnerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", item.OwnerID, "user-1")
			}
			if item.Type != domain.SecretTypeCredentials {
				t.Fatalf("unexpected type: got %q, want %q", item.Type, domain.SecretTypeCredentials)
			}
			if item.Meta != "github" {
				t.Fatalf("unexpected meta: got %q, want %q", item.Meta, "github")
			}
			if item.Version != 1 {
				t.Fatalf("unexpected version: got %d, want %d", item.Version, 1)
			}
			if !item.CreatedAt.Equal(now) {
				t.Fatalf("unexpected created_at: got %v, want %v", item.CreatedAt, now)
			}
			if !item.UpdatedAt.Equal(now) {
				t.Fatalf("unexpected updated_at: got %v, want %v", item.UpdatedAt, now)
			}

			data, ok := item.Data.(domain.CredentialData)
			if !ok {
				t.Fatalf("unexpected data type: %T", item.Data)
			}
			if data.Login != "sergey" {
				t.Fatalf("unexpected login: got %q, want %q", data.Login, "sergey")
			}
			if data.Password != "qwerty" {
				t.Fatalf("unexpected password: got %q, want %q", data.Password, "qwerty")
			}

			if item.ID == "" {
				t.Fatal("expected generated id")
			}

			return nil
		},
		updateFn:           mustNotCallUpdate(t),
		getByIDFn:          mustNotCallGetByID(t),
		listByOwnerFn:      mustNotCallListByOwner(t),
		softDeleteFn:       mustNotCallDelete(t),
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)
	svc.now = func() time.Time { return now }

	item, err := svc.Create(context.Background(), CreateInput{
		OwnerID: "user-1",
		Type:    domain.SecretTypeCredentials,
		Meta:    "github",
		Data: domain.CredentialData{
			Login:    "sergey",
			Password: "qwerty",
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if item == nil {
		t.Fatal("expected created item")
	}
	if item.Type != domain.SecretTypeCredentials {
		t.Fatalf("unexpected created item type: got %q, want %q", item.Type, domain.SecretTypeCredentials)
	}
}

func TestService_CreateBinary_Success(t *testing.T) {
	now := time.Date(2026, 4, 20, 12, 30, 0, 0, time.UTC)

	repo := &secretRepositoryStub{
		createFn: func(ctx context.Context, item *domain.SecretItem) error {
			data, ok := item.Data.(domain.BinaryData)
			if !ok {
				t.Fatalf("unexpected data type: %T", item.Data)
			}

			if data.Filename != "hello.txt" {
				t.Fatalf("unexpected filename: got %q, want %q", data.Filename, "hello.txt")
			}
			if data.MIMEType != "text/plain" {
				t.Fatalf("unexpected mime type: got %q, want %q", data.MIMEType, "text/plain")
			}
			if string(data.Content) != "hello" {
				t.Fatalf("unexpected content: got %q, want %q", string(data.Content), "hello")
			}
			if !item.CreatedAt.Equal(now) {
				t.Fatalf("unexpected created_at: got %v, want %v", item.CreatedAt, now)
			}

			return nil
		},
		updateFn:           mustNotCallUpdate(t),
		getByIDFn:          mustNotCallGetByID(t),
		listByOwnerFn:      mustNotCallListByOwner(t),
		softDeleteFn:       mustNotCallDelete(t),
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)
	svc.now = func() time.Time { return now }

	_, err := svc.Create(context.Background(), CreateInput{
		OwnerID: "user-1",
		Type:    domain.SecretTypeBinary,
		Meta:    "file",
		Data: domain.BinaryData{
			Filename: "hello.txt",
			MIMEType: "text/plain",
			Content:  []byte("hello"),
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
}

func TestService_CreateBinary_TooLarge(t *testing.T) {
	repo := &secretRepositoryStub{
		createFn:           mustNotCallCreate(t),
		updateFn:           mustNotCallUpdate(t),
		getByIDFn:          mustNotCallGetByID(t),
		listByOwnerFn:      mustNotCallListByOwner(t),
		softDeleteFn:       mustNotCallDelete(t),
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)

	tooLargeContent := make([]byte, maxBinaryPayloadSize+1)

	_, err := svc.Create(context.Background(), CreateInput{
		OwnerID: "user-1",
		Type:    domain.SecretTypeBinary,
		Meta:    "large file",
		Data: domain.BinaryData{
			Filename: "large.bin",
			MIMEType: "application/octet-stream",
			Content:  tooLargeContent,
		},
	})
	if err == nil {
		t.Fatal("expected error for too large binary payload")
	}

	if !errors.Is(err, domain.ErrBinaryPayloadTooLarge) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrBinaryPayloadTooLarge)
	}
}

func TestService_CreateInvalidText_ReturnsInvalidSecretData(t *testing.T) {
	repo := &secretRepositoryStub{
		createFn:           mustNotCallCreate(t),
		updateFn:           mustNotCallUpdate(t),
		getByIDFn:          mustNotCallGetByID(t),
		listByOwnerFn:      mustNotCallListByOwner(t),
		softDeleteFn:       mustNotCallDelete(t),
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		OwnerID: "user-1",
		Type:    domain.SecretTypeText,
		Meta:    "note",
		Data: domain.TextData{
			Text: "",
		},
	})
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if !errors.Is(err, domain.ErrInvalidSecretData) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrInvalidSecretData)
	}
}

func TestService_Update_IncrementsVersion(t *testing.T) {
	now := time.Date(2026, 4, 20, 13, 0, 0, 0, time.UTC)
	createdAt := now.Add(-time.Hour)

	repo := &secretRepositoryStub{
		createFn: mustNotCallCreate(t),
		getByIDFn: func(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
			return &domain.SecretItem{
				ID:        "secret-1",
				OwnerID:   "user-1",
				Type:      domain.SecretTypeText,
				Meta:      "old",
				Data:      domain.TextData{Text: "old text"},
				Version:   3,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			}, nil
		},
		updateFn: func(ctx context.Context, item *domain.SecretItem) error {
			if item.ID != "secret-1" {
				t.Fatalf("unexpected id: got %q, want %q", item.ID, "secret-1")
			}
			if item.OwnerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", item.OwnerID, "user-1")
			}
			if item.Version != 4 {
				t.Fatalf("unexpected version: got %d, want %d", item.Version, 4)
			}
			if !item.UpdatedAt.Equal(now) {
				t.Fatalf("unexpected updated_at: got %v, want %v", item.UpdatedAt, now)
			}

			data, ok := item.Data.(domain.TextData)
			if !ok {
				t.Fatalf("unexpected data type: %T", item.Data)
			}
			if data.Text != "new text" {
				t.Fatalf("unexpected text: got %q, want %q", data.Text, "new text")
			}

			return nil
		},
		listByOwnerFn:      mustNotCallListByOwner(t),
		softDeleteFn:       mustNotCallDelete(t),
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)
	svc.now = func() time.Time { return now }

	item, err := svc.Update(context.Background(), UpdateInput{
		ID:      "secret-1",
		OwnerID: "user-1",
		Type:    domain.SecretTypeText,
		Meta:    "new",
		Data: domain.TextData{
			Text: "new text",
		},
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if item == nil {
		t.Fatal("expected updated item")
	}
	if item.Version != 4 {
		t.Fatalf("unexpected returned version: got %d, want %d", item.Version, 4)
	}
}

func TestService_Delete_UsesSoftDelete(t *testing.T) {
	now := time.Date(2026, 4, 20, 14, 0, 0, 0, time.UTC)

	repo := &secretRepositoryStub{
		createFn:      mustNotCallCreate(t),
		updateFn:      mustNotCallUpdate(t),
		getByIDFn:     mustNotCallGetByID(t),
		listByOwnerFn: mustNotCallListByOwner(t),
		softDeleteFn: func(ctx context.Context, ownerID, secretID string, deletedAt time.Time) error {
			if ownerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", ownerID, "user-1")
			}
			if secretID != "secret-1" {
				t.Fatalf("unexpected secret id: got %q, want %q", secretID, "secret-1")
			}
			if !deletedAt.Equal(now) {
				t.Fatalf("unexpected deleted_at: got %v, want %v", deletedAt, now)
			}
			return nil
		},
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)
	svc.now = func() time.Time { return now }

	if err := svc.Delete(context.Background(), "user-1", "secret-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func mustNotCallCreate(t *testing.T) func(ctx context.Context, item *domain.SecretItem) error {
	return func(ctx context.Context, item *domain.SecretItem) error {
		t.Fatal("create should not be called")
		return nil
	}
}

func mustNotCallUpdate(t *testing.T) func(ctx context.Context, item *domain.SecretItem) error {
	return func(ctx context.Context, item *domain.SecretItem) error {
		t.Fatal("update should not be called")
		return nil
	}
}

func mustNotCallGetByID(t *testing.T) func(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
	return func(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
		t.Fatal("get by id should not be called")
		return nil, nil
	}
}

func mustNotCallListByOwner(t *testing.T) func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
	return func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
		t.Fatal("list by owner should not be called")
		return nil, nil
	}
}

func mustNotCallDelete(t *testing.T) func(ctx context.Context, ownerID, secretID string, deletedAt time.Time) error {
	return func(ctx context.Context, ownerID, secretID string, deletedAt time.Time) error {
		t.Fatal("soft delete should not be called")
		return nil
	}
}

func mustNotCallListChangesSince(t *testing.T) func(ctx context.Context, ownerID string, since time.Time) ([]*domain.SecretItem, error) {
	return func(ctx context.Context, ownerID string, since time.Time) ([]*domain.SecretItem, error) {
		t.Fatal("list changes since should not be called")
		return nil, nil
	}
}

// TestService_CreateCard_Success проверяет успешное создание секрета типа card.
func TestService_CreateCard_Success(t *testing.T) {
	now := time.Date(2026, 4, 20, 12, 45, 0, 0, time.UTC)

	repo := &secretRepositoryStub{
		createFn: func(ctx context.Context, item *domain.SecretItem) error {
			data, ok := item.Data.(domain.CardData)
			if !ok {
				t.Fatalf("unexpected data type: %T", item.Data)
			}

			if data.Number != "4111111111111111" {
				t.Fatalf("unexpected number: got %q, want %q", data.Number, "4111111111111111")
			}
			if data.Cardholder != "SERGEY DYUZHOV" {
				t.Fatalf("unexpected cardholder: got %q, want %q", data.Cardholder, "SERGEY DYUZHOV")
			}
			if data.ExpiryMonth != 12 {
				t.Fatalf("unexpected expiry month: got %d, want %d", data.ExpiryMonth, 12)
			}
			if data.ExpiryYear != 2030 {
				t.Fatalf("unexpected expiry year: got %d, want %d", data.ExpiryYear, 2030)
			}
			if data.CVV != "123" {
				t.Fatalf("unexpected cvv: got %q, want %q", data.CVV, "123")
			}
			if !item.CreatedAt.Equal(now) {
				t.Fatalf("unexpected created_at: got %v, want %v", item.CreatedAt, now)
			}

			return nil
		},
		updateFn:           mustNotCallUpdate(t),
		getByIDFn:          mustNotCallGetByID(t),
		listByOwnerFn:      mustNotCallListByOwner(t),
		softDeleteFn:       mustNotCallDelete(t),
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)
	svc.now = func() time.Time { return now }

	_, err := svc.Create(context.Background(), CreateInput{
		OwnerID: "user-1",
		Type:    domain.SecretTypeCard,
		Meta:    "main visa",
		Data: domain.CardData{
			Number:      "4111111111111111",
			Cardholder:  "SERGEY DYUZHOV",
			ExpiryMonth: 12,
			ExpiryYear:  2030,
			CVV:         "123",
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
}

// TestService_CreateInvalidCard_ReturnsInvalidSecretData проверяет ошибку валидации card payload.
func TestService_CreateInvalidCard_ReturnsInvalidSecretData(t *testing.T) {
	repo := &secretRepositoryStub{
		createFn:           mustNotCallCreate(t),
		updateFn:           mustNotCallUpdate(t),
		getByIDFn:          mustNotCallGetByID(t),
		listByOwnerFn:      mustNotCallListByOwner(t),
		softDeleteFn:       mustNotCallDelete(t),
		listChangesSinceFn: mustNotCallListChangesSince(t),
	}

	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		OwnerID: "user-1",
		Type:    domain.SecretTypeCard,
		Meta:    "broken card",
		Data: domain.CardData{
			Number:      "",
			Cardholder:  "SERGEY DYUZHOV",
			ExpiryMonth: 12,
			ExpiryYear:  2030,
			CVV:         "123",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid card data")
	}
	if !errors.Is(err, domain.ErrInvalidSecretData) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrInvalidSecretData)
	}
}
