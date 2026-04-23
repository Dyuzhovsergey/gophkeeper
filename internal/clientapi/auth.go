package clientapi

import (
	"context"
	"fmt"
)

// RegisterRequest описывает запрос регистрации пользователя.
type RegisterRequest struct {
	// Login — логин пользователя.
	Login string `json:"login"`

	// Password — пароль пользователя.
	Password string `json:"password"`
}

// RegisterResponse описывает ответ после успешной регистрации.
type RegisterResponse struct {
	// ID — идентификатор пользователя.
	ID string `json:"id"`

	// Login — логин пользователя.
	Login string `json:"login"`
}

// LoginRequest описывает запрос входа пользователя.
type LoginRequest struct {
	// Login — логин пользователя.
	Login string `json:"login"`

	// Password — пароль пользователя.
	Password string `json:"password"`
}

// LoginResponse описывает ответ после успешного входа.
type LoginResponse struct {
	// Token — JWT токен доступа.
	Token string `json:"token"`

	// ExpiresAt — время истечения срока действия токена в RFC3339 JSON-формате.
	ExpiresAt string `json:"expires_at"`
}

// MeResponse описывает ответ защищённого маршрута текущего пользователя.
type MeResponse struct {
	// UserID — идентификатор авторизованного пользователя.
	UserID string `json:"user_id"`

	// SessionID — идентификатор активной сессии.
	SessionID string `json:"session_id"`
}

// Register регистрирует нового пользователя на сервере.
func (c *Client) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	var resp RegisterResponse

	if err := c.doJSON(ctx, "POST", "/api/user/register", req, &resp, nil); err != nil {
		return nil, fmt.Errorf("register request: %w", err)
	}

	return &resp, nil
}

// Login выполняет вход пользователя и получает JWT токен.
func (c *Client) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	var resp LoginResponse

	if err := c.doJSON(ctx, "POST", "/api/user/login", req, &resp, nil); err != nil {
		return nil, fmt.Errorf("login request: %w", err)
	}

	return &resp, nil
}

// Me возвращает информацию о текущем авторизованном пользователе.
func (c *Client) Me(ctx context.Context, token string) (*MeResponse, error) {
	var resp MeResponse

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	if err := c.doJSON(ctx, "GET", "/api/user/me", nil, &resp, headers); err != nil {
		return nil, fmt.Errorf("me request: %w", err)
	}

	return &resp, nil
}
