package dto

import (
	"encoding/json"
	"time"
)

// SecretUpsertRequest описывает запрос создания или обновления секрета.
type SecretUpsertRequest struct {
	// Type — тип секрета.
	Type string `json:"type"`

	// Meta — произвольная текстовая метаинформация.
	Meta string `json:"meta"`

	// Data — полезная нагрузка секрета.
	Data json.RawMessage `json:"data"`
}

// CredentialsData описывает payload секрета типа credentials.
type CredentialsData struct {
	// Login — логин во внешней системе.
	Login string `json:"login"`

	// Password — пароль во внешней системе.
	Password string `json:"password"`
}

// TextData describes payload секрета типа text.
type TextData struct {
	// Text — произвольный текст.
	Text string `json:"text"`
}

// SecretResponse описывает HTTP-ответ с одним секретом.
type SecretResponse struct {
	// ID — идентификатор секрета.
	ID string `json:"id"`

	// Type — тип секрета.
	Type string `json:"type"`

	// Meta — текстовая метаинформация.
	Meta string `json:"meta"`

	// Data — payload секрета.
	Data any `json:"data"`

	// Version — версия секрета.
	Version int64 `json:"version"`

	// CreatedAt — время создания.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt — время последнего изменения.
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretListResponse описывает ответ со списком секретов.
type SecretListResponse struct {
	// Items — список секретов пользователя.
	Items []SecretResponse `json:"items"`
}