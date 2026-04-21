package clientapi

import (
	"context"
	"fmt"
)

// SecretUpsertRequest описывает клиентский запрос на создание секрета.
type SecretUpsertRequest struct {
	// Type — тип секрета.
	Type string `json:"type"`

	// Meta — текстовая метаинформация.
	Meta string `json:"meta"`

	// Data — полезная нагрузка секрета.
	Data any `json:"data"`
}

// CredentialsData описывает payload секрета типа credentials.
type CredentialsData struct {
	// Login — логин во внешней системе.
	Login string `json:"login"`

	// Password — пароль во внешней системе.
	Password string `json:"password"`
}

// TextData описывает payload секрета типа text.
type TextData struct {
	// Text — произвольный текст секрета.
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

// SecretResponse описывает один секрет, возвращаемый сервером.
type SecretResponse struct {
	// ID — идентификатор секрета.
	ID string `json:"id"`

	// Type — тип секрета.
	Type string `json:"type"`

	// Meta — текстовая метаинформация.
	Meta string `json:"meta"`

	// Data — payload секрета.
	Data map[string]any `json:"data"`

	// Version — версия секрета.
	Version int64 `json:"version"`

	// CreatedAt — время создания.
	CreatedAt string `json:"created_at"`

	// UpdatedAt — время последнего изменения.
	UpdatedAt string `json:"updated_at"`
}

// SecretListResponse описывает ответ со списком секретов.
type SecretListResponse struct {
	// Items — список секретов пользователя.
	Items []SecretResponse `json:"items"`
}

// ListSecrets получает список секретов текущего пользователя.
func (c *Client) ListSecrets(ctx context.Context, token string) (*SecretListResponse, error) {
	var resp SecretListResponse

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	if err := c.doJSON(ctx, "GET", "/api/secrets", nil, &resp, headers); err != nil {
		return nil, fmt.Errorf("list secrets request: %w", err)
	}

	return &resp, nil
}

// CreateSecret создаёт новый секрет пользователя.
func (c *Client) CreateSecret(ctx context.Context, token string, req SecretUpsertRequest) (*SecretResponse, error) {
	var resp SecretResponse

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	if err := c.doJSON(ctx, "POST", "/api/secrets", req, &resp, headers); err != nil {
		return nil, fmt.Errorf("create secret request: %w", err)
	}

	return &resp, nil
}

// GetSecretByID получает один секрет пользователя по идентификатору.
func (c *Client) GetSecretByID(ctx context.Context, token, secretID string) (*SecretResponse, error) {
	var resp SecretResponse

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	if err := c.doJSON(ctx, "GET", "/api/secrets/"+secretID, nil, &resp, headers); err != nil {
		return nil, fmt.Errorf("get secret by id request: %w", err)
	}

	return &resp, nil
}
