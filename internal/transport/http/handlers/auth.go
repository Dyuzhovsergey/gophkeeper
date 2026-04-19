package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	"github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/dto"
	httpmiddleware "github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/middleware"
)

// AuthService описывает бизнес-логику аутентификации,
// которая нужна HTTP-обработчикам.
type AuthService interface {
	// Register регистрирует нового пользователя.
	Register(ctx context.Context, login, password string) (*domain.User, error)

	// Login выполняет вход пользователя.
	Login(ctx context.Context, login, password string) (*auth.LoginResult, error)
}

// AuthHandler обрабатывает HTTP-запросы регистрации и входа.
type AuthHandler struct {
	service AuthService
}

// NewAuthHandler создаёт новый AuthHandler.
func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Register обрабатывает регистрацию пользователя.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req dto.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || strings.TrimSpace(req.Password) == "" {
		writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	user, err := h.service.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserAlreadyExists):
			writeError(w, http.StatusConflict, "user already exists")
			return
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	resp := dto.RegisterResponse{
		ID:    user.ID,
		Login: user.Login,
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login обрабатывает вход пользователя.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || strings.TrimSpace(req.Password) == "" {
		writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	result, err := h.service.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	resp := dto.LoginResponse{
		Token:     result.Token,
		ExpiresAt: result.Session.ExpiresAt,
	}

	writeJSON(w, http.StatusOK, resp)
}

// Me возвращает identity текущего авторизованного пользователя.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	identity, ok := httpmiddleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "identity not found in context")
		return
	}

	resp := dto.MeResponse{
		UserID:    identity.UserID,
		SessionID: identity.SessionID,
	}

	writeJSON(w, http.StatusOK, resp)
}

type errorResponse struct {
	Error string `json:"error"`
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode json body: %w", err)
	}

	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, errorResponse{
		Error: message,
	})
}
