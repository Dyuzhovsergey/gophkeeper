package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	authservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
	vaultservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/vault"
	httpmiddleware "github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/middleware"
)

type vaultServiceStub struct {
	createFn      func(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error)
	getByIDFn     func(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error)
	listByOwnerFn func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error)
	updateFn      func(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error)
	deleteFn      func(ctx context.Context, ownerID, secretID string) error
}

func (s *vaultServiceStub) Create(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error) {
	return s.createFn(ctx, input)
}

func (s *vaultServiceStub) GetByID(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
	return s.getByIDFn(ctx, ownerID, secretID)
}

func (s *vaultServiceStub) ListByOwner(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
	return s.listByOwnerFn(ctx, ownerID)
}

func (s *vaultServiceStub) Update(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error) {
	return s.updateFn(ctx, input)
}

func (s *vaultServiceStub) Delete(ctx context.Context, ownerID, secretID string) error {
	return s.deleteFn(ctx, ownerID, secretID)
}

func TestSecretsHandler_CreateCredentials_Success(t *testing.T) {
	handler := NewSecretsHandler(&vaultServiceStub{
		createFn: func(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error) {
			if input.OwnerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", input.OwnerID, "user-1")
			}
			if input.Type != domain.SecretTypeCredentials {
				t.Fatalf("unexpected secret type: got %q, want %q", input.Type, domain.SecretTypeCredentials)
			}

			data, ok := input.Data.(domain.CredentialData)
			if !ok {
				t.Fatalf("unexpected data type: %T", input.Data)
			}

			if data.Login != "sergey" {
				t.Fatalf("unexpected login: got %q, want %q", data.Login, "sergey")
			}
			if data.Password != "qwerty" {
				t.Fatalf("unexpected password: got %q, want %q", data.Password, "qwerty")
			}

			now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

			return &domain.SecretItem{
				ID:        "secret-1",
				OwnerID:   input.OwnerID,
				Type:      input.Type,
				Meta:      input.Meta,
				Data:      data,
				Version:   1,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
		getByIDFn:     mustNotCallGetByID(t),
		listByOwnerFn: mustNotCallListByOwner(t),
		updateFn:      mustNotCallUpdate(t),
		deleteFn:      mustNotCallDelete(t),
	})

	body := []byte(`{
		"type":"credentials",
		"meta":"github account",
		"data":{
			"login":"sergey",
			"password":"qwerty"
		}
	}`)

	req := authenticatedRequest(http.MethodPost, "/api/secrets", body)
	rec := httptest.NewRecorder()

	handler.Collection(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Meta string `json:"meta"`
		Data struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		} `json:"data"`
		Version int64 `json:"version"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if resp.ID != "secret-1" {
		t.Fatalf("unexpected id: got %q, want %q", resp.ID, "secret-1")
	}
	if resp.Type != "credentials" {
		t.Fatalf("unexpected type: got %q, want %q", resp.Type, "credentials")
	}
	if resp.Meta != "github account" {
		t.Fatalf("unexpected meta: got %q, want %q", resp.Meta, "github account")
	}
	if resp.Data.Login != "sergey" {
		t.Fatalf("unexpected data.login: got %q, want %q", resp.Data.Login, "sergey")
	}
	if resp.Data.Password != "qwerty" {
		t.Fatalf("unexpected data.password: got %q, want %q", resp.Data.Password, "qwerty")
	}
	if resp.Version != 1 {
		t.Fatalf("unexpected version: got %d, want %d", resp.Version, 1)
	}
}

func TestSecretsHandler_CreateBinary_Success(t *testing.T) {
	handler := NewSecretsHandler(&vaultServiceStub{
		createFn: func(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error) {
			if input.OwnerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", input.OwnerID, "user-1")
			}
			if input.Type != domain.SecretTypeBinary {
				t.Fatalf("unexpected secret type: got %q, want %q", input.Type, domain.SecretTypeBinary)
			}

			data, ok := input.Data.(domain.BinaryData)
			if !ok {
				t.Fatalf("unexpected data type: %T", input.Data)
			}

			if data.Filename != "hello.txt" {
				t.Fatalf("unexpected filename: got %q, want %q", data.Filename, "hello.txt")
			}
			if data.MIMEType != "text/plain" {
				t.Fatalf("unexpected mime type: got %q, want %q", data.MIMEType, "text/plain")
			}
			if string(data.Content) != "hello" {
				t.Fatalf("unexpected binary content: got %q, want %q", string(data.Content), "hello")
			}

			now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

			return &domain.SecretItem{
				ID:        "secret-binary",
				OwnerID:   input.OwnerID,
				Type:      input.Type,
				Meta:      input.Meta,
				Data:      data,
				Version:   1,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
		getByIDFn:     mustNotCallGetByID(t),
		listByOwnerFn: mustNotCallListByOwner(t),
		updateFn:      mustNotCallUpdate(t),
		deleteFn:      mustNotCallDelete(t),
	})

	body := []byte(`{
		"type":"binary",
		"meta":"test file",
		"data":{
			"filename":"hello.txt",
			"mime_type":"text/plain",
			"content_base64":"aGVsbG8="
		}
	}`)

	req := authenticatedRequest(http.MethodPost, "/api/secrets", body)
	rec := httptest.NewRecorder()

	handler.Collection(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestSecretsHandler_List_Success(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

	handler := NewSecretsHandler(&vaultServiceStub{
		createFn:  mustNotCallCreate(t),
		getByIDFn: mustNotCallGetByID(t),
		listByOwnerFn: func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
			if ownerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", ownerID, "user-1")
			}

			return []*domain.SecretItem{
				{
					ID:        "secret-text",
					OwnerID:   ownerID,
					Type:      domain.SecretTypeText,
					Meta:      "note",
					Data:      domain.TextData{Text: "hello"},
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:      "secret-card",
					OwnerID: ownerID,
					Type:    domain.SecretTypeCard,
					Meta:    "visa",
					Data: domain.CardData{
						Number:      "4111111111111111",
						Cardholder:  "SERGEY DYUZHOV",
						ExpiryMonth: 12,
						ExpiryYear:  2030,
						CVV:         "123",
					},
					Version:   2,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}, nil
		},
		updateFn: mustNotCallUpdate(t),
		deleteFn: mustNotCallDelete(t),
	})

	req := authenticatedRequest(http.MethodGet, "/api/secrets", nil)
	rec := httptest.NewRecorder()

	handler.Collection(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Items []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"items"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("unexpected items length: got %d, want %d", len(resp.Items), 2)
	}

	if resp.Items[0].ID != "secret-text" {
		t.Fatalf("unexpected first item id: got %q, want %q", resp.Items[0].ID, "secret-text")
	}
	if resp.Items[1].Type != "card" {
		t.Fatalf("unexpected second item type: got %q, want %q", resp.Items[1].Type, "card")
	}
}

func TestSecretsHandler_GetByID_OwnerIsolation(t *testing.T) {
	handler := NewSecretsHandler(&vaultServiceStub{
		createFn: mustNotCallCreate(t),
		getByIDFn: func(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
			if ownerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", ownerID, "user-1")
			}
			if secretID != "foreign-secret" {
				t.Fatalf("unexpected secret id: got %q, want %q", secretID, "foreign-secret")
			}

			// Имитируем ситуацию, когда секрет другого пользователя
			// недоступен текущему владельцу.
			return nil, domain.ErrSecretNotFound
		},
		listByOwnerFn: mustNotCallListByOwner(t),
		updateFn:      mustNotCallUpdate(t),
		deleteFn:      mustNotCallDelete(t),
	})

	req := authenticatedRequest(http.MethodGet, "/api/secrets/foreign-secret", nil)
	rec := httptest.NewRecorder()

	handler.Item(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSecretsHandler_UpdateCard_Success(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

	handler := NewSecretsHandler(&vaultServiceStub{
		createFn:      mustNotCallCreate(t),
		getByIDFn:     mustNotCallGetByID(t),
		listByOwnerFn: mustNotCallListByOwner(t),
		updateFn: func(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error) {
			if input.OwnerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", input.OwnerID, "user-1")
			}
			if input.ID != "secret-card" {
				t.Fatalf("unexpected secret id: got %q, want %q", input.ID, "secret-card")
			}
			if input.Type != domain.SecretTypeCard {
				t.Fatalf("unexpected secret type: got %q, want %q", input.Type, domain.SecretTypeCard)
			}

			data, ok := input.Data.(domain.CardData)
			if !ok {
				t.Fatalf("unexpected data type: %T", input.Data)
			}

			if data.Cardholder != "SERGEY DYUZHOV" {
				t.Fatalf("unexpected cardholder: got %q, want %q", data.Cardholder, "SERGEY DYUZHOV")
			}

			return &domain.SecretItem{
				ID:        input.ID,
				OwnerID:   input.OwnerID,
				Type:      input.Type,
				Meta:      input.Meta,
				Data:      data,
				Version:   3,
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
			}, nil
		},
		deleteFn: mustNotCallDelete(t),
	})

	body := []byte(`{
		"type":"card",
		"meta":"updated visa",
		"data":{
			"number":"4111111111111111",
			"cardholder":"SERGEY DYUZHOV",
			"expiry_month":12,
			"expiry_year":2030,
			"cvv":"123"
		}
	}`)

	req := authenticatedRequest(http.MethodPut, "/api/secrets/secret-card", body)
	rec := httptest.NewRecorder()

	handler.Item(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSecretsHandler_Delete_Success(t *testing.T) {
	handler := NewSecretsHandler(&vaultServiceStub{
		createFn:      mustNotCallCreate(t),
		getByIDFn:     mustNotCallGetByID(t),
		listByOwnerFn: mustNotCallListByOwner(t),
		updateFn:      mustNotCallUpdate(t),
		deleteFn: func(ctx context.Context, ownerID, secretID string) error {
			if ownerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", ownerID, "user-1")
			}
			if secretID != "secret-1" {
				t.Fatalf("unexpected secret id: got %q, want %q", secretID, "secret-1")
			}

			return nil
		},
	})

	req := authenticatedRequest(http.MethodDelete, "/api/secrets/secret-1", nil)
	rec := httptest.NewRecorder()

	handler.Item(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func authenticatedRequest(method, target string, body []byte) *http.Request {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}

	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	ctx := httpmiddleware.ContextWithIdentity(req.Context(), &authservice.Identity{
		UserID:    "user-1",
		SessionID: "session-1",
	})

	return req.WithContext(ctx)
}

func mustNotCallCreate(t *testing.T) func(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error) {
	return func(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error) {
		t.Fatal("create should not be called")
		return nil, nil
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

func mustNotCallUpdate(t *testing.T) func(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error) {
	return func(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error) {
		t.Fatal("update should not be called")
		return nil, nil
	}
}

func mustNotCallDelete(t *testing.T) func(ctx context.Context, ownerID, secretID string) error {
	return func(ctx context.Context, ownerID, secretID string) error {
		t.Fatal("delete should not be called")
		return nil
	}
}
