package clientapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient создаёт Client поверх тестового HTTP-сервера.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	return client
}

// TestNew_EmptyBaseURL проверяет ошибку при пустом baseURL.
func TestNew_EmptyBaseURL(t *testing.T) {
	_, err := New("", nil)
	if err == nil {
		t.Fatal("expected error for empty baseURL")
	}
}

// TestRegister_Success проверяет успешную регистрацию пользователя.
func TestRegister_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/user/register" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/user/register")
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if req.Login != "sergey" {
			t.Fatalf("unexpected login: got %q, want %q", req.Login, "sergey")
		}
		if req.Password != "qwerty" {
			t.Fatalf("unexpected password: got %q, want %q", req.Password, "qwerty")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			ID:    "user-1",
			Login: "sergey",
		})
	})

	resp, err := client.Register(context.Background(), RegisterRequest{
		Login:    "sergey",
		Password: "qwerty",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "user-1" {
		t.Fatalf("unexpected id: got %q, want %q", resp.ID, "user-1")
	}
	if resp.Login != "sergey" {
		t.Fatalf("unexpected login: got %q, want %q", resp.Login, "sergey")
	}
}

// TestRegister_APIError проверяет разбор JSON-ошибки API.
func TestRegister_APIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "user already exists",
		})
	})

	_, err := client.Register(context.Background(), RegisterRequest{
		Login:    "sergey",
		Password: "qwerty",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("unexpected status code: got %d, want %d", apiErr.StatusCode, http.StatusConflict)
	}
	if apiErr.Message != "user already exists" {
		t.Fatalf("unexpected message: got %q, want %q", apiErr.Message, "user already exists")
	}
}

// TestLogin_Success проверяет успешный login.
func TestLogin_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/user/login" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/user/login")
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if req.Login != "sergey" {
			t.Fatalf("unexpected login: got %q, want %q", req.Login, "sergey")
		}
		if req.Password != "qwerty" {
			t.Fatalf("unexpected password: got %q, want %q", req.Password, "qwerty")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LoginResponse{
			Token:     "jwt-token",
			ExpiresAt: "2026-04-30T12:00:00Z",
		})
	})

	resp, err := client.Login(context.Background(), LoginRequest{
		Login:    "sergey",
		Password: "qwerty",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Token != "jwt-token" {
		t.Fatalf("unexpected token: got %q, want %q", resp.Token, "jwt-token")
	}
	if resp.ExpiresAt != "2026-04-30T12:00:00Z" {
		t.Fatalf("unexpected expires_at: got %q", resp.ExpiresAt)
	}
}

// TestMe_SendsAuthorizationHeader проверяет, что Me отправляет Bearer token.
func TestMe_SendsAuthorizationHeader(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/user/me" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/user/me")
		}

		gotAuth := r.Header.Get("Authorization")
		wantAuth := "Bearer jwt-token"
		if gotAuth != wantAuth {
			t.Fatalf("unexpected Authorization header: got %q, want %q", gotAuth, wantAuth)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MeResponse{
			UserID:    "user-1",
			SessionID: "session-1",
		})
	})

	resp, err := client.Me(context.Background(), "jwt-token")
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.UserID != "user-1" {
		t.Fatalf("unexpected user id: got %q, want %q", resp.UserID, "user-1")
	}
	if resp.SessionID != "session-1" {
		t.Fatalf("unexpected session id: got %q, want %q", resp.SessionID, "session-1")
	}
}

// TestListSecrets_Success проверяет успешную загрузку списка секретов.
func TestListSecrets_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/secrets" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/secrets")
		}
		if r.Header.Get("Authorization") != "Bearer jwt-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SecretListResponse{
			Items: []SecretResponse{
				{
					ID:   "secret-1",
					Type: "text",
					Meta: "note",
					Data: map[string]any{
						"text": "hello",
					},
					Version:   1,
					CreatedAt: "2026-04-23T10:00:00Z",
					UpdatedAt: "2026-04-23T10:00:00Z",
				},
			},
		})
	})

	resp, err := client.ListSecrets(context.Background(), "jwt-token")
	if err != nil {
		t.Fatalf("ListSecrets returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Items) != 1 {
		t.Fatalf("unexpected items len: got %d, want %d", len(resp.Items), 1)
	}
	if resp.Items[0].ID != "secret-1" {
		t.Fatalf("unexpected id: got %q, want %q", resp.Items[0].ID, "secret-1")
	}
	if resp.Items[0].Type != "text" {
		t.Fatalf("unexpected type: got %q, want %q", resp.Items[0].Type, "text")
	}
}

// TestCreateSecret_Success проверяет создание секрета и корректную отправку payload.
func TestCreateSecret_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/secrets" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/secrets")
		}
		if r.Header.Get("Authorization") != "Bearer jwt-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if req["type"] != "text" {
			t.Fatalf("unexpected type: got %v, want %v", req["type"], "text")
		}
		if req["meta"] != "note" {
			t.Fatalf("unexpected meta: got %v, want %v", req["meta"], "note")
		}

		data, ok := req["data"].(map[string]any)
		if !ok {
			t.Fatalf("unexpected data type: %T", req["data"])
		}
		if data["text"] != "hello" {
			t.Fatalf("unexpected text: got %v, want %v", data["text"], "hello")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SecretResponse{
			ID:   "secret-1",
			Type: "text",
			Meta: "note",
			Data: map[string]any{
				"text": "hello",
			},
			Version:   1,
			CreatedAt: "2026-04-23T10:00:00Z",
			UpdatedAt: "2026-04-23T10:00:00Z",
		})
	})

	resp, err := client.CreateSecret(context.Background(), "jwt-token", SecretUpsertRequest{
		Type: "text",
		Meta: "note",
		Data: TextData{Text: "hello"},
	})
	if err != nil {
		t.Fatalf("CreateSecret returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "secret-1" {
		t.Fatalf("unexpected id: got %q, want %q", resp.ID, "secret-1")
	}
}

// TestGetSecretByID_Success проверяет получение одного секрета.
func TestGetSecretByID_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/secrets/secret-1" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/secrets/secret-1")
		}
		if r.Header.Get("Authorization") != "Bearer jwt-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SecretResponse{
			ID:   "secret-1",
			Type: "credentials",
			Meta: "github",
			Data: map[string]any{
				"login":    "sergey",
				"password": "qwerty",
			},
			Version:   1,
			CreatedAt: "2026-04-23T10:00:00Z",
			UpdatedAt: "2026-04-23T10:00:00Z",
		})
	})

	resp, err := client.GetSecretByID(context.Background(), "jwt-token", "secret-1")
	if err != nil {
		t.Fatalf("GetSecretByID returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "secret-1" {
		t.Fatalf("unexpected id: got %q, want %q", resp.ID, "secret-1")
	}
	if resp.Type != "credentials" {
		t.Fatalf("unexpected type: got %q, want %q", resp.Type, "credentials")
	}
}

// TestUpdateSecret_Success проверяет обновление секрета.
func TestUpdateSecret_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodPut)
		}
		if r.URL.Path != "/api/secrets/secret-1" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/secrets/secret-1")
		}
		if r.Header.Get("Authorization") != "Bearer jwt-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if req["type"] != "text" {
			t.Fatalf("unexpected type: got %v, want %v", req["type"], "text")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SecretResponse{
			ID:   "secret-1",
			Type: "text",
			Meta: "updated note",
			Data: map[string]any{
				"text": "updated text",
			},
			Version:   2,
			CreatedAt: "2026-04-23T10:00:00Z",
			UpdatedAt: "2026-04-23T12:00:00Z",
		})
	})

	resp, err := client.UpdateSecret(context.Background(), "jwt-token", "secret-1", SecretUpsertRequest{
		Type: "text",
		Meta: "updated note",
		Data: TextData{Text: "updated text"},
	})
	if err != nil {
		t.Fatalf("UpdateSecret returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Version != 2 {
		t.Fatalf("unexpected version: got %d, want %d", resp.Version, 2)
	}
}

// TestDeleteSecret_Success проверяет удаление секрета.
func TestDeleteSecret_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodDelete)
		}
		if r.URL.Path != "/api/secrets/secret-1" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/secrets/secret-1")
		}
		if r.Header.Get("Authorization") != "Bearer jwt-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteSecret(context.Background(), "jwt-token", "secret-1")
	if err != nil {
		t.Fatalf("DeleteSecret returned error: %v", err)
	}
}

// TestDecodeAPIError_FallbackStatusText проверяет fallback на StatusText при не-JSON ошибке.
func TestDecodeAPIError_FallbackStatusText(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("plain text error"))
	})

	_, err := client.Me(context.Background(), "jwt-token")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
	}
	if !strings.EqualFold(apiErr.Message, http.StatusText(http.StatusUnauthorized)) {
		t.Fatalf("unexpected message: got %q, want %q", apiErr.Message, http.StatusText(http.StatusUnauthorized))
	}
}

func TestSync_Success(t *testing.T) {
	since := time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)
	serverTime := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 4, 24, 9, 30, 0, 0, time.UTC)

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: got %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/sync" {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, "/api/sync")
		}
		if r.Header.Get("Authorization") != "Bearer jwt-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		if got := r.URL.Query().Get("since"); got != since.Format(time.RFC3339) {
			t.Fatalf("unexpected since query: got %q, want %q", got, since.Format(time.RFC3339))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SecretSyncResponse{
			Items: []SecretResponse{
				{
					ID:        "secret-1",
					Type:      "text",
					Meta:      "note",
					Data:      map[string]any{"text": "hello"},
					Version:   2,
					CreatedAt: "2026-04-24T08:00:00Z",
					UpdatedAt: "2026-04-24T09:15:00Z",
				},
				{
					ID:        "secret-2",
					Type:      "text",
					Meta:      "deleted",
					Data:      map[string]any{"text": "bye"},
					Version:   3,
					CreatedAt: "2026-04-24T07:00:00Z",
					UpdatedAt: "2026-04-24T09:30:00Z",
					DeletedAt: &deletedAt,
				},
			},
			ServerTime: serverTime,
			Count:      2,
		})
	})

	resp, err := client.Sync(context.Background(), "jwt-token", since)
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Count != 2 {
		t.Fatalf("unexpected count: got %d, want %d", resp.Count, 2)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("unexpected items len: got %d, want %d", len(resp.Items), 2)
	}
	if resp.Items[1].DeletedAt == nil {
		t.Fatal("expected deleted item with deleted_at")
	}
	if !resp.ServerTime.Equal(serverTime) {
		t.Fatalf("unexpected server time: got %v, want %v", resp.ServerTime, serverTime)
	}
}

func TestSync_ZeroSinceUsesEpoch(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("since")
		want := time.Unix(0, 0).UTC().Format(time.RFC3339)

		if got != want {
			t.Fatalf("unexpected since query for zero time: got %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SecretSyncResponse{
			Items:      []SecretResponse{},
			ServerTime: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
			Count:      0,
		})
	})

	_, err := client.Sync(context.Background(), "jwt-token", time.Time{})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
}
