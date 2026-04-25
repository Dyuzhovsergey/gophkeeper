package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	authservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
)

type authServiceStub struct {
	authenticateFn func(ctx context.Context, token string) (*authservice.Identity, error)
}

func (s *authServiceStub) Authenticate(ctx context.Context, token string) (*authservice.Identity, error) {
	return s.authenticateFn(ctx, token)
}

func TestAuth_RequireAuth_MissingAuthorizationHeader(t *testing.T) {
	serviceCalled := false

	mw := NewAuth(&authServiceStub{
		authenticateFn: func(ctx context.Context, token string) (*authservice.Identity, error) {
			serviceCalled = true
			return nil, nil
		},
	})

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	rec := httptest.NewRecorder()

	mw.RequireAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	if serviceCalled {
		t.Fatal("authenticate service should not be called")
	}

	if nextCalled {
		t.Fatal("next handler should not be called")
	}
}

func TestAuth_RequireAuth_InvalidAuthorizationHeader(t *testing.T) {
	serviceCalled := false

	mw := NewAuth(&authServiceStub{
		authenticateFn: func(ctx context.Context, token string) (*authservice.Identity, error) {
			serviceCalled = true
			return nil, nil
		},
	})

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	req.Header.Set("Authorization", "BadHeaderValue")
	rec := httptest.NewRecorder()

	mw.RequireAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	if serviceCalled {
		t.Fatal("authenticate service should not be called")
	}

	if nextCalled {
		t.Fatal("next handler should not be called")
	}
}

func TestAuth_RequireAuth_AuthenticationErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "invalid token",
			err:  domain.ErrInvalidToken,
			want: http.StatusUnauthorized,
		},
		{
			name: "unauthorized",
			err:  domain.ErrUnauthorized,
			want: http.StatusUnauthorized,
		},
		{
			name: "session expired",
			err:  domain.ErrSessionExpired,
			want: http.StatusUnauthorized,
		},
		{
			name: "internal error",
			err:  errors.New("unexpected storage error"),
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := NewAuth(&authServiceStub{
				authenticateFn: func(ctx context.Context, token string) (*authservice.Identity, error) {
					if token != "jwt-token" {
						t.Fatalf("unexpected token: got %q, want %q", token, "jwt-token")
					}
					return nil, tt.err
				},
			})

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
			req.Header.Set("Authorization", "Bearer jwt-token")
			rec := httptest.NewRecorder()

			mw.RequireAuth(next).ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("unexpected status code: got %d, want %d", rec.Code, tt.want)
			}

			if nextCalled {
				t.Fatal("next handler should not be called")
			}
		})
	}
}

func TestAuth_RequireAuth_Success(t *testing.T) {
	mw := NewAuth(&authServiceStub{
		authenticateFn: func(ctx context.Context, token string) (*authservice.Identity, error) {
			if token != "jwt-token" {
				t.Fatalf("unexpected token: got %q, want %q", token, "jwt-token")
			}

			return &authservice.Identity{
				UserID:    "user-1",
				SessionID: "session-1",
			}, nil
		},
	})

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true

		identity, ok := IdentityFromContext(r.Context())
		if !ok {
			t.Fatal("expected identity in context")
		}

		if identity.UserID != "user-1" {
			t.Fatalf("unexpected user id: got %q, want %q", identity.UserID, "user-1")
		}

		if identity.SessionID != "session-1" {
			t.Fatalf("unexpected session id: got %q, want %q", identity.SessionID, "session-1")
		}

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	req.Header.Set("Authorization", "Bearer jwt-token")
	rec := httptest.NewRecorder()

	mw.RequireAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   error
	}{
		{
			name:      "valid bearer token",
			header:    "Bearer jwt-token",
			wantToken: "jwt-token",
			wantErr:   nil,
		},
		{
			name:      "valid bearer token lowercase scheme",
			header:    "bearer jwt-token",
			wantToken: "jwt-token",
			wantErr:   nil,
		},
		{
			name:    "empty header",
			header:  "",
			wantErr: domain.ErrUnauthorized,
		},
		{
			name:    "missing token",
			header:  "Bearer",
			wantErr: domain.ErrInvalidToken,
		},
		{
			name:    "wrong scheme",
			header:  "Basic abc",
			wantErr: domain.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractBearerToken(tt.header)

			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("unexpected error: got %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got != tt.wantToken {
				t.Fatalf("unexpected token: got %q, want %q", got, tt.wantToken)
			}
		})
	}
}

func TestAuth_RequireAuth_ErrorResponseBody(t *testing.T) {
	mw := NewAuth(&authServiceStub{
		authenticateFn: func(ctx context.Context, token string) (*authservice.Identity, error) {
			return nil, domain.ErrInvalidToken
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	req.Header.Set("Authorization", "Bearer jwt-token")
	rec := httptest.NewRecorder()

	mw.RequireAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var resp struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	if resp.Error != "unauthorized" {
		t.Fatalf("unexpected error message: got %q, want %q", resp.Error, "unauthorized")
	}
}
