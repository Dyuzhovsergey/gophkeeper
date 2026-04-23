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

// CardData описывает payload секрета типа card.
type CardData struct {
	// Number — номер карты.
	Number string `json:"number"`

	// Cardholder — имя держателя карты.
	Cardholder string `json:"cardholder"`

	// ExpiryMonth — месяц окончания срока действия.
	ExpiryMonth uint8 `json:"expiry_month"`

	// ExpiryYear — год окончания срока действия.
	ExpiryYear uint16 `json:"expiry_year"`

	// CVV — защитный код карты.
	CVV string `json:"cvv"`
}

// BinaryData описывает payload секрета типа binary.
type BinaryData struct {
	// Filename — имя файла.
	Filename string `json:"filename"`

	// MIMEType — MIME-тип данных.
	MIMEType string `json:"mime_type"`

	// ContentBase64 — содержимое файла в base64.
	ContentBase64 string `json:"content_base64"`
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
