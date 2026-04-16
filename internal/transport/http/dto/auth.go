// Package dto содержит структуры JSON запросов и ответов при регистрации и входа.
package dto

import "time"

// RegisterRequest описывает JSON-запрос регистрации пользователя.
type RegisterRequest struct {
	// Login — логин пользователя.
	Login string `json:"login"`

	// Password — пароль пользователя.
	Password string `json:"password"`
}

// RegisterResponse описывает JSON-ответ после успешной регистрации.
type RegisterResponse struct {
	// ID — идентификатор созданного пользователя.
	ID string `json:"id"`

	// Login — логин созданного пользователя.
	Login string `json:"login"`
}

// LoginRequest описывает JSON-запрос входа пользователя.
type LoginRequest struct {
	// Login — логин пользователя.
	Login string `json:"login"`

	// Password — пароль пользователя.
	Password string `json:"password"`
}

// LoginResponse описывает JSON-ответ после успешного входа.
type LoginResponse struct {
	// Token — JWT токен доступа.
	Token string `json:"token"`

	// ExpiresAt — время истечения срока действия сессии.
	ExpiresAt time.Time `json:"expires_at"`
}
