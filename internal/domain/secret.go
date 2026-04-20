package domain

import (
	"fmt"
	"strings"
	"time"
)

// SecretType описывает тип хранимого секрета.
type SecretType string

const (
	// SecretTypeCredentials — пара логин/пароль.
	SecretTypeCredentials SecretType = "credentials"

	// SecretTypeText — произвольные текстовые данные.
	SecretTypeText SecretType = "text"

	// SecretTypeCard — данные банковской карты.
	SecretTypeCard SecretType = "card"

	// SecretTypeBinary — произвольные бинарные данные.
	SecretTypeBinary SecretType = "binary"
)

// IsValid проверяет, поддерживается ли тип секрета.
func (t SecretType) IsValid() bool {
	switch t {
	case SecretTypeCredentials, SecretTypeText, SecretTypeCard, SecretTypeBinary:
		return true
	default:
		return false
	}
}

// SecretData описывает полезную нагрузку секрета.
type SecretData interface {
	// SecretType возвращает тип секрета.
	SecretType() SecretType
}

// SecretItem описывает универсальную сущность пользовательского секрета.
type SecretItem struct {
	// ID — уникальный идентификатор секрета.
	ID string

	// OwnerID — идентификатор владельца секрета.
	OwnerID string

	// Type — тип секрета.
	Type SecretType

	// Meta — произвольная текстовая метаинформация.
	Meta string

	// Data — типизированная полезная нагрузка секрета.
	Data SecretData

	// CreatedAt — время создания секрета.
	CreatedAt time.Time

	// UpdatedAt — время последнего изменения секрета.
	UpdatedAt time.Time

	// DeletedAt — время мягкого удаления секрета.
	// Если значение nil, секрет считается активным.
	DeletedAt *time.Time

	// Version — версия секрета для синхронизации.
	Version int64
}

// Validate проверяет базовую корректность секрета.
func (s SecretItem) Validate() error {
	if strings.TrimSpace(s.OwnerID) == "" {
		return fmt.Errorf("secret owner id is empty")
	}

	if !s.Type.IsValid() {
		return ErrInvalidSecretType
	}

	if s.Data == nil {
		return ErrInvalidSecretData
	}

	if s.Data.SecretType() != s.Type {
		return fmt.Errorf("secret type does not match payload type")
	}

	if s.Version < 0 {
		return fmt.Errorf("secret version must be non-negative")
	}

	return nil
}

// IsDeleted показывает, помечен ли секрет как удалённый.
func (s SecretItem) IsDeleted() bool {
	return s.DeletedAt != nil
}

// CredentialData описывает секрет типа логин/пароль.
type CredentialData struct {
	// Login — логин пользователя во внешней системе.
	Login string

	// Password — пароль пользователя во внешней системе.
	Password string
}

// SecretType возвращает тип секрета.
func (d CredentialData) SecretType() SecretType {
	return SecretTypeCredentials
}

// TextData описывает секрет типа текст.
type TextData struct {
	// Text — произвольный текст секрета.
	Text string
}

// SecretType возвращает тип секрета.
func (d TextData) SecretType() SecretType {
	return SecretTypeText
}

// CardData описывает секрет типа банковская карта.
type CardData struct {
	// Number — номер карты.
	Number string

	// Cardholder — имя держателя карты.
	Cardholder string

	// ExpiryMonth — месяц окончания срока действия.
	ExpiryMonth uint8

	// ExpiryYear — год окончания срока действия.
	ExpiryYear uint16

	// CVV — защитный код карты.
	CVV string
}

// SecretType возвращает тип секрета.
func (d CardData) SecretType() SecretType {
	return SecretTypeCard
}

// BinaryData описывает секрет типа бинарные данные.
type BinaryData struct {
	// Filename — имя файла.
	Filename string

	// MIMEType — MIME-тип данных.
	MIMEType string

	// Content — содержимое файла.
	Content []byte
}

// SecretType возвращает тип секрета.
func (d BinaryData) SecretType() SecretType {
	return SecretTypeBinary
}
