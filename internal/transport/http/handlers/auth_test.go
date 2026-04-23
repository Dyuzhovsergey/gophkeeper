package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	authservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
	httpmiddleware "github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/middleware"
)

type authServiceStub struct {
	registerFn func(ctx context.Context, login, password string) (*domain.User, error)
	loginFn    func(ctx context.Context, login, password string) (*authservice.LoginResult, error)
}

func (s *authServiceStub) Register(ctx context.Context, login, password string) (*domain.User, error) {
	return s.registerFn(ctx, login, password)
}

func (s *authServiceStub) Login(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
	return s.loginFn(ctx, login, password)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			if login != "sergey" {
				t.Fatalf("unexpected login: %q", login)
			}
			if password != "qwerty" {
				t.Fatalf("unexpected password: %q", password)
			}

			return &domain.User{
				ID:    "user-1",
				Login: "sergey",
			}, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	body := []byte(`{"login":"sergey","password":"qwerty"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if resp.ID != "user-1" {
		t.Fatalf("unexpected id: got %q, want %q", resp.ID, "user-1")
	}

	if resp.Login != "sergey" {
		t.Fatalf("unexpected login: got %q, want %q", resp.Login, "sergey")
	}
}

func TestAuthHandler_Register_UserAlreadyExists(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			return nil, domain.ErrUserAlreadyExists
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	body := []byte(`{"login":"sergey","password":"qwerty"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte(`{"login":`)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	expiresAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			if login != "sergey" {
				t.Fatalf("unexpected login: %q", login)
			}
			if password != "qwerty" {
				t.Fatalf("unexpected password: %q", password)
			}

			return &authservice.LoginResult{
				Token: "jwt-token",
				Session: &domain.Session{
					ID:        "session-1",
					UserID:    "user-1",
					ExpiresAt: expiresAt,
					CreatedAt: expiresAt.Add(-time.Hour),
				},
				User: &domain.User{
					ID:    "user-1",
					Login: "sergey",
				},
			}, nil
		},
	})

	body := []byte(`{"login":"sergey","password":"qwerty"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if resp.Token != "jwt-token" {
		t.Fatalf("unexpected token: got %q, want %q", resp.Token, "jwt-token")
	}

	if !resp.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("unexpected expires_at: got %v, want %v", resp.ExpiresAt, expiresAt)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			return nil, domain.ErrInvalidCredentials
		},
	})

	body := []byte(`{"login":"sergey","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Login_InternalError(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			return nil, errors.New("unexpected db error")
		},
	})

	body := []byte(`{"login":"sergey","password":"qwerty"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestAuthHandler_Register_MethodNotAllowed(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/register", nil)
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestAuthHandler_Register_EmptyLoginOrPassword(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	body := []byte(`{"login":"   ","password":"qwerty"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_Register_InternalError(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			return nil, errors.New("unexpected db error")
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	body := []byte(`{"login":"sergey","password":"qwerty"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestAuthHandler_Login_MethodNotAllowed(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/login", nil)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestAuthHandler_Login_InvalidBody(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader([]byte(`{"login":`)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_Login_EmptyLoginOrPassword(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	body := []byte(`{"login":"sergey","password":"   "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_Me_Success(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	req = req.WithContext(httpmiddleware.ContextWithIdentity(req.Context(), &authservice.Identity{
		UserID:    "user-1",
		SessionID: "session-1",
	}))

	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		UserID    string `json:"user_id"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if resp.UserID != "user-1" {
		t.Fatalf("unexpected user_id: got %q, want %q", resp.UserID, "user-1")
	}
	if resp.SessionID != "session-1" {
		t.Fatalf("unexpected session_id: got %q, want %q", resp.SessionID, "session-1")
	}
}

func TestAuthHandler_Me_IdentityMissing(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestAuthHandler_Me_MethodNotAllowed(t *testing.T) {
	handler := NewAuthHandler(&authServiceStub{
		registerFn: func(ctx context.Context, login, password string) (*domain.User, error) {
			t.Fatal("register should not be called")
			return nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (*authservice.LoginResult, error) {
			t.Fatal("login should not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/me", nil)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
