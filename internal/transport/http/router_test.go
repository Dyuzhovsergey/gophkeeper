package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	authservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
	vaultservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/vault"
)

type routerAuthServiceStub struct {
	registerFn     func(ctx context.Context, login, password string) (*domain.User, error)
	loginFn        func(ctx context.Context, login, password string) (*authservice.LoginResult, error)
	authenticateFn func(ctx context.Context, token string) (*authservice.Identity, error)
}

func (s *routerAuthServiceStub) Register(ctx context.Context, login, password string) (*domain.User, error) {
	if s.registerFn == nil {
		return nil, nil
	}
	return s.registerFn(ctx, login, password)
}

func (s *routerAuthServiceStub) Login(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
	if s.loginFn == nil {
		return nil, nil
	}
	return s.loginFn(ctx, login, password)
}

func (s *routerAuthServiceStub) Authenticate(ctx context.Context, token string) (*authservice.Identity, error) {
	if s.authenticateFn == nil {
		return nil, nil
	}
	return s.authenticateFn(ctx, token)
}

type routerVaultServiceStub struct {
	createFn      func(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error)
	getByIDFn     func(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error)
	listByOwnerFn func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error)
	updateFn      func(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error)
	deleteFn      func(ctx context.Context, ownerID, secretID string) error
}

func (s *routerVaultServiceStub) Create(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error) {
	if s.createFn == nil {
		return nil, nil
	}
	return s.createFn(ctx, input)
}

func (s *routerVaultServiceStub) GetByID(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
	if s.getByIDFn == nil {
		return nil, nil
	}
	return s.getByIDFn(ctx, ownerID, secretID)
}

func (s *routerVaultServiceStub) ListByOwner(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
	if s.listByOwnerFn == nil {
		return nil, nil
	}
	return s.listByOwnerFn(ctx, ownerID)
}

func (s *routerVaultServiceStub) Update(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error) {
	if s.updateFn == nil {
		return nil, nil
	}
	return s.updateFn(ctx, input)
}

func (s *routerVaultServiceStub) Delete(ctx context.Context, ownerID, secretID string) error {
	if s.deleteFn == nil {
		return nil
	}
	return s.deleteFn(ctx, ownerID, secretID)
}

// TestNewRouter_HealthRoute проверяет, что публичный health-маршрут доступен.
func TestNewRouter_HealthRoute(t *testing.T) {
	authStub := &routerAuthServiceStub{}
	vaultStub := &routerVaultServiceStub{}

	router := NewRouter(authStub, vaultStub)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestNewRouter_ProtectedRoute_RequiresAuth проверяет, что защищённый маршрут требует авторизацию.
func TestNewRouter_ProtectedRoute_RequiresAuth(t *testing.T) {
	authStub := &routerAuthServiceStub{
		authenticateFn: func(ctx context.Context, token string) (*authservice.Identity, error) {
			t.Fatal("Authenticate should not be called without Authorization header")
			return nil, nil
		},
	}
	vaultStub := &routerVaultServiceStub{
		listByOwnerFn: func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
			t.Fatal("ListByOwner should not be called without auth")
			return nil, nil
		},
	}

	router := NewRouter(authStub, vaultStub)

	req := httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestNewRouter_SecretsRoute_WithAuth проверяет, что при валидной авторизации запрос доходит до secrets handler.
func TestNewRouter_SecretsRoute_WithAuth(t *testing.T) {
	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	authStub := &routerAuthServiceStub{
		authenticateFn: func(ctx context.Context, token string) (*authservice.Identity, error) {
			if token != "jwt-token" {
				t.Fatalf("unexpected token: got %q, want %q", token, "jwt-token")
			}
			return &authservice.Identity{
				UserID:    "user-1",
				SessionID: "session-1",
			}, nil
		},
	}

	vaultStub := &routerVaultServiceStub{
		listByOwnerFn: func(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
			if ownerID != "user-1" {
				t.Fatalf("unexpected owner id: got %q, want %q", ownerID, "user-1")
			}

			return []*domain.SecretItem{
				{
					ID:        "secret-1",
					OwnerID:   "user-1",
					Type:      domain.SecretTypeText,
					Meta:      "note",
					Data:      domain.TextData{Text: "hello"},
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}, nil
		},
	}

	router := NewRouter(authStub, vaultStub)

	req := httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
	req.Header.Set("Authorization", "Bearer jwt-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("unexpected items len: got %d, want %d", len(resp.Items), 1)
	}
	if resp.Items[0].ID != "secret-1" {
		t.Fatalf("unexpected secret id: got %q, want %q", resp.Items[0].ID, "secret-1")
	}
}
