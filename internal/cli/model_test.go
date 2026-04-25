package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dyuzhovsergey/gophkeeper/internal/clientapi"
	"github.com/Dyuzhovsergey/gophkeeper/internal/storage/local"
)

// newTestClient создаёт clientapi.Client поверх httptest.Server.
func newTestClient(t *testing.T, handler http.HandlerFunc) *clientapi.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := clientapi.New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("clientapi.New returned error: %v", err)
	}

	return client
}

// newTestStore создаёт FileStore с сохранённой локальной сессией.
func newTestStore(t *testing.T) *local.FileStore {
	t.Helper()

	path := filepath.Join(t.TempDir(), "session.json")

	store, err := local.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	err = store.SaveSession(local.Session{
		Token:     "jwt-token",
		UserID:    "user-1",
		SessionID: "session-1",
		ExpiresAt: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	return store
}

// menuIndexByName находит индекс пункта меню по имени.
func menuIndexByName(t *testing.T, name string) int {
	t.Helper()

	for i, item := range menuItems {
		if item == name {
			return i
		}
	}

	t.Fatalf("menu item %q not found", name)
	return -1
}

// TestSelectMenuItem_AddFile_OpensBinaryForm проверяет открытие формы add file из меню.
func TestSelectMenuItem_AddFile_OpensBinaryForm(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP server should not be called")
	})
	store := newTestStore(t)

	model := NewModel(client, store, nil)
	model.menuIndex = menuIndexByName(t, "add file")

	gotModel, cmd := model.selectMenuItem()
	if cmd != nil {
		t.Fatal("expected nil cmd for opening add file form")
	}

	got, ok := gotModel.(*Model)
	if !ok {
		t.Fatalf("unexpected model type: %T", gotModel)
	}

	if got.screen != screenCreateBinarySecret {
		t.Fatalf("unexpected screen: got %v, want %v", got.screen, screenCreateBinarySecret)
	}

	if len(got.inputs) != 2 {
		t.Fatalf("unexpected inputs len: got %d, want %d", len(got.inputs), 2)
	}

	if got.inputs[0].Placeholder != "Meta" {
		t.Fatalf("unexpected first placeholder: got %q, want %q", got.inputs[0].Placeholder, "Meta")
	}
	if got.inputs[1].Placeholder != "File path" {
		t.Fatalf("unexpected second placeholder: got %q, want %q", got.inputs[1].Placeholder, "File path")
	}
}

// TestUpdateSecretDetails_DeleteOpensConfirm проверяет, что по d открывается экран подтверждения удаления.
func TestUpdateSecretDetails_DeleteOpensConfirm(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP server should not be called")
	})
	store := newTestStore(t)

	model := NewModel(client, store, nil)
	model.screen = screenSecretDetails
	model.secretDetails = &clientapi.SecretResponse{
		ID:   "secret-1",
		Type: "text",
		Meta: "note",
	}

	gotModel, cmd := model.updateSecretDetails(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Fatal("expected nil cmd for opening delete confirm")
	}

	got, ok := gotModel.(*Model)
	if !ok {
		t.Fatalf("unexpected model type: %T", gotModel)
	}

	if got.screen != screenDeleteConfirm {
		t.Fatalf("unexpected screen: got %v, want %v", got.screen, screenDeleteConfirm)
	}
	if got.deletingSecretID != "secret-1" {
		t.Fatalf("unexpected deletingSecretID: got %q, want %q", got.deletingSecretID, "secret-1")
	}
}

// TestUpdateDeleteConfirm_Cancel проверяет отмену удаления по Esc.
func TestUpdateDeleteConfirm_Cancel(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP server should not be called")
	})
	store := newTestStore(t)

	model := NewModel(client, store, nil)
	model.screen = screenDeleteConfirm
	model.deletingSecretID = "secret-1"
	model.secretDetails = &clientapi.SecretResponse{
		ID:   "secret-1",
		Type: "text",
		Meta: "note",
	}

	gotModel, cmd := model.updateDeleteConfirm(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("expected nil cmd for cancel delete")
	}

	got, ok := gotModel.(*Model)
	if !ok {
		t.Fatalf("unexpected model type: %T", gotModel)
	}

	if got.screen != screenSecretDetails {
		t.Fatalf("unexpected screen: got %v, want %v", got.screen, screenSecretDetails)
	}
	if got.deletingSecretID != "" {
		t.Fatalf("expected empty deletingSecretID after cancel, got %q", got.deletingSecretID)
	}
}

// TestInitSecretEditForm_Binary проверяет открытие edit-form для binary.
func TestInitSecretEditForm_Binary(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP server should not be called")
	})
	store := newTestStore(t)

	model := NewModel(client, store, nil)
	model.secretDetails = &clientapi.SecretResponse{
		ID:   "secret-bin",
		Type: "binary",
		Meta: "file meta",
		Data: map[string]any{
			"filename":       "hello.txt",
			"mime_type":      "text/plain",
			"content_base64": base64.StdEncoding.EncodeToString([]byte("hello")),
		},
	}

	model.initSecretEditForm()

	if model.screen != screenUpdateBinarySecret {
		t.Fatalf("unexpected screen: got %v, want %v", model.screen, screenUpdateBinarySecret)
	}
	if model.editingSecretID != "secret-bin" {
		t.Fatalf("unexpected editingSecretID: got %q, want %q", model.editingSecretID, "secret-bin")
	}
	if len(model.inputs) != 2 {
		t.Fatalf("unexpected inputs len: got %d, want %d", len(model.inputs), 2)
	}
	if got := model.inputs[0].Value(); got != "file meta" {
		t.Fatalf("unexpected meta value: got %q, want %q", got, "file meta")
	}
	if got := model.inputs[1].Value(); got != "" {
		t.Fatalf("expected empty file path for binary update form, got %q", got)
	}
}

// TestRunCreateBinarySecretCmd_Success проверяет успешное создание binary-секрета.
func TestRunCreateBinarySecretCmd_Success(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/secrets":
			if got := r.Header.Get("Authorization"); got != "Bearer jwt-token" {
				t.Fatalf("unexpected Authorization header: got %q, want %q", got, "Bearer jwt-token")
			}

			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			if req["type"] != "binary" {
				t.Fatalf("unexpected type: got %v, want %v", req["type"], "binary")
			}
			if req["meta"] != "test file" {
				t.Fatalf("unexpected meta: got %v, want %v", req["meta"], "test file")
			}

			data, ok := req["data"].(map[string]any)
			if !ok {
				t.Fatalf("unexpected data type: %T", req["data"])
			}

			if data["filename"] != "hello.txt" {
				t.Fatalf("unexpected filename: got %v, want %v", data["filename"], "hello.txt")
			}
			if data["mime_type"] != "text/plain; charset=utf-8" && data["mime_type"] != "text/plain" {
				t.Fatalf("unexpected mime_type: got %v", data["mime_type"])
			}
			if data["content_base64"] != base64.StdEncoding.EncodeToString([]byte("hello")) {
				t.Fatalf("unexpected content_base64: got %v", data["content_base64"])
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(clientapi.SecretResponse{
				ID:   "secret-created",
				Type: "binary",
				Meta: "test file",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/secrets":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(clientapi.SecretListResponse{
				Items: []clientapi.SecretResponse{
					{
						ID:   "secret-created",
						Type: "binary",
						Meta: "test file",
					},
				},
			})

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	store := newTestStore(t)
	model := NewModel(client, store, nil)
	model.inputs = makeBinaryFormInputs(t, "test file", filePath)

	msg := model.runCreateBinarySecretCmd()()
	listMsg, ok := msg.(secretsListMsg)
	if !ok {
		t.Fatalf("unexpected message type: %T", msg)
	}

	if len(listMsg.items) != 1 {
		t.Fatalf("unexpected items len: got %d, want %d", len(listMsg.items), 1)
	}
	if listMsg.items[0].ID != "secret-created" {
		t.Fatalf("unexpected id: got %q, want %q", listMsg.items[0].ID, "secret-created")
	}
}

// TestRunUpdateBinarySecretCmd_Success проверяет успешное обновление binary-секрета.
func TestRunUpdateBinarySecretCmd_Success(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "updated.txt")
	if err := os.WriteFile(filePath, []byte("updated content"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/secrets/secret-bin" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if req["type"] != "binary" {
			t.Fatalf("unexpected type: got %v, want %v", req["type"], "binary")
		}
		if req["meta"] != "updated meta" {
			t.Fatalf("unexpected meta: got %v, want %v", req["meta"], "updated meta")
		}

		data, ok := req["data"].(map[string]any)
		if !ok {
			t.Fatalf("unexpected data type: %T", req["data"])
		}
		if data["filename"] != "updated.txt" {
			t.Fatalf("unexpected filename: got %v, want %v", data["filename"], "updated.txt")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clientapi.SecretResponse{
			ID:   "secret-bin",
			Type: "binary",
			Meta: "updated meta",
			Data: map[string]any{
				"filename":       "updated.txt",
				"mime_type":      "text/plain",
				"content_base64": base64.StdEncoding.EncodeToString([]byte("updated content")),
			},
			Version:   2,
			CreatedAt: "2026-04-23T10:00:00Z",
			UpdatedAt: "2026-04-23T12:00:00Z",
		})
	})

	store := newTestStore(t)
	model := NewModel(client, store, nil)
	model.editingSecretID = "secret-bin"
	model.inputs = makeBinaryFormInputs(t, "updated meta", filePath)

	msg := model.runUpdateBinarySecretCmd()()
	detailsMsg, ok := msg.(secretDetailsMsg)
	if !ok {
		t.Fatalf("unexpected message type: %T", msg)
	}

	if detailsMsg.item.ID != "secret-bin" {
		t.Fatalf("unexpected id: got %q, want %q", detailsMsg.item.ID, "secret-bin")
	}
	if detailsMsg.item.Meta != "updated meta" {
		t.Fatalf("unexpected meta: got %q, want %q", detailsMsg.item.Meta, "updated meta")
	}
}

// TestRunDeleteSecretCmd_Success проверяет успешное удаление секрета с последующим refresh списка.
func TestRunDeleteSecretCmd_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/api/secrets/secret-1":
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodGet && r.URL.Path == "/api/secrets":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(clientapi.SecretListResponse{
				Items: []clientapi.SecretResponse{
					{ID: "secret-2", Type: "text", Meta: "still here"},
				},
			})

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	store := newTestStore(t)
	model := NewModel(client, store, nil)

	msg := model.runDeleteSecretCmd("secret-1")()
	listMsg, ok := msg.(secretsListMsg)
	if !ok {
		t.Fatalf("unexpected message type: %T", msg)
	}

	if len(listMsg.items) != 1 {
		t.Fatalf("unexpected items len: got %d, want %d", len(listMsg.items), 1)
	}
	if listMsg.items[0].ID != "secret-2" {
		t.Fatalf("unexpected id: got %q, want %q", listMsg.items[0].ID, "secret-2")
	}
}

// TestRunDeleteSecretCmd_NoSession проверяет ошибку при отсутствии локальной сессии.
func TestRunDeleteSecretCmd_NoSession(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP server should not be called")
	})

	storePath := filepath.Join(t.TempDir(), "session.json")
	store, err := local.NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}

	model := NewModel(client, store, nil)

	msg := model.runDeleteSecretCmd("secret-1")()
	errMsg, ok := msg.(actionErrorMsg)
	if !ok {
		t.Fatalf("unexpected message type: %T", msg)
	}

	if errMsg.err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(errMsg.err, local.ErrSessionNotFound) && errMsg.err.Error() == "" {
		t.Fatal("expected meaningful error for missing local session")
	}
}

// makeBinaryFormInputs создаёт inputs для binary-формы.
func makeBinaryFormInputs(t *testing.T, meta, path string) []textinput.Model {
	t.Helper()

	metaInput := textinput.New()
	metaInput.Placeholder = "Meta"
	metaInput.SetValue(meta)

	pathInput := textinput.New()
	pathInput.Placeholder = "File path"
	pathInput.SetValue(path)

	return []textinput.Model{metaInput, pathInput}
}
