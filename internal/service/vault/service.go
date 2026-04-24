package vault

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	repo "github.com/Dyuzhovsergey/gophkeeper/internal/repository"
)

const maxBinaryPayloadSize = 1 << 20 // 1 MiB

// Service реализует бизнес-логику работы с пользовательскими секретами.
type Service struct {
	secrets repo.SecretRepository
	now     func() time.Time
}

// CreateInput описывает входные данные для создания секрета.
type CreateInput struct {
	// OwnerID — идентификатор владельца секрета.
	OwnerID string

	// Type — тип секрета.
	Type domain.SecretType

	// Meta — произвольная текстовая метаинформация.
	Meta string

	// Data — полезная нагрузка секрета.
	Data domain.SecretData
}

// UpdateInput описывает входные данные для обновления секрета.
type UpdateInput struct {
	// ID — идентификатор секрета.
	ID string

	// OwnerID — идентификатор владельца секрета.
	OwnerID string

	// Type — тип секрета.
	Type domain.SecretType

	// Meta — произвольная текстовая метаинформация.
	Meta string

	// Data — полезная нагрузка секрета.
	Data domain.SecretData
}

// NewService создаёт VaultService.
func NewService(secrets repo.SecretRepository) *Service {
	return &Service{
		secrets: secrets,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// Create создаёт новый секрет пользователя.
func (s *Service) Create(ctx context.Context, input CreateInput) (*domain.SecretItem, error) {
	ownerID := strings.TrimSpace(input.OwnerID)
	if ownerID == "" {
		return nil, fmt.Errorf("owner id is empty")
	}

	data, err := normalizeAndValidateData(input.Type, input.Data)
	if err != nil {
		return nil, err
	}

	now := s.now()

	item := &domain.SecretItem{
		ID:        newID(),
		OwnerID:   ownerID,
		Type:      input.Type,
		Meta:      strings.TrimSpace(input.Meta),
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	if err := item.Validate(); err != nil {
		return nil, fmt.Errorf("validate secret item: %w", err)
	}

	if err := s.secrets.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return item, nil
}

// GetByID возвращает один секрет пользователя.
func (s *Service) GetByID(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error) {
	ownerID = strings.TrimSpace(ownerID)
	secretID = strings.TrimSpace(secretID)

	if ownerID == "" {
		return nil, fmt.Errorf("owner id is empty")
	}

	if secretID == "" {
		return nil, fmt.Errorf("secret id is empty")
	}

	item, err := s.secrets.GetByID(ctx, ownerID, secretID)
	if err != nil {
		return nil, fmt.Errorf("get secret by id: %w", err)
	}

	return item, nil
}

// ListByOwner возвращает список активных секретов пользователя.
func (s *Service) ListByOwner(ctx context.Context, ownerID string) ([]*domain.SecretItem, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, fmt.Errorf("owner id is empty")
	}

	items, err := s.secrets.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list secrets by owner: %w", err)
	}

	return items, nil
}

// ListChangesSince возвращает изменения пользователя после указанного времени.
func (s *Service) ListChangesSince(ctx context.Context, ownerID string, since time.Time) ([]*domain.SecretItem, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, fmt.Errorf("owner id is empty")
	}

	items, err := s.secrets.ListChangesSince(ctx, ownerID, since)
	if err != nil {
		return nil, fmt.Errorf("list secret changes since: %w", err)
	}

	return items, nil
}

// Update обновляет существующий секрет пользователя.
func (s *Service) Update(ctx context.Context, input UpdateInput) (*domain.SecretItem, error) {
	ownerID := strings.TrimSpace(input.OwnerID)
	secretID := strings.TrimSpace(input.ID)

	if ownerID == "" {
		return nil, fmt.Errorf("owner id is empty")
	}

	if secretID == "" {
		return nil, fmt.Errorf("secret id is empty")
	}

	existing, err := s.secrets.GetByID(ctx, ownerID, secretID)
	if err != nil {
		return nil, fmt.Errorf("get existing secret: %w", err)
	}

	data, err := normalizeAndValidateData(input.Type, input.Data)
	if err != nil {
		return nil, err
	}

	existing.Type = input.Type
	existing.Meta = strings.TrimSpace(input.Meta)
	existing.Data = data
	existing.UpdatedAt = s.now()
	existing.Version++

	if err := existing.Validate(); err != nil {
		return nil, fmt.Errorf("validate updated secret item: %w", err)
	}

	if err := s.secrets.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update secret: %w", err)
	}

	return existing, nil
}

// Delete выполняет мягкое удаление секрета пользователя.
func (s *Service) Delete(ctx context.Context, ownerID, secretID string) error {
	ownerID = strings.TrimSpace(ownerID)
	secretID = strings.TrimSpace(secretID)

	if ownerID == "" {
		return fmt.Errorf("owner id is empty")
	}

	if secretID == "" {
		return fmt.Errorf("secret id is empty")
	}

	if err := s.secrets.SoftDelete(ctx, ownerID, secretID, s.now()); err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}

	return nil
}

func normalizeAndValidateData(secretType domain.SecretType, data domain.SecretData) (domain.SecretData, error) {
	switch secretType {
	case domain.SecretTypeCredentials:
		switch v := data.(type) {
		case domain.CredentialData:
			if err := validateCredentialData(v); err != nil {
				return nil, err
			}
			return v, nil
		case *domain.CredentialData:
			if v == nil {
				return nil, domain.ErrInvalidSecretData
			}
			if err := validateCredentialData(*v); err != nil {
				return nil, err
			}
			return *v, nil
		default:
			return nil, domain.ErrInvalidSecretData
		}

	case domain.SecretTypeText:
		switch v := data.(type) {
		case domain.TextData:
			if err := validateTextData(v); err != nil {
				return nil, err
			}
			return v, nil
		case *domain.TextData:
			if v == nil {
				return nil, domain.ErrInvalidSecretData
			}
			if err := validateTextData(*v); err != nil {
				return nil, err
			}
			return *v, nil
		default:
			return nil, domain.ErrInvalidSecretData
		}
	case domain.SecretTypeCard:
		switch v := data.(type) {
		case domain.CardData:
			if err := validateCardData(v); err != nil {
				return nil, err
			}
			return v, nil
		case *domain.CardData:
			if v == nil {
				return nil, domain.ErrInvalidSecretData
			}
			if err := validateCardData(*v); err != nil {
				return nil, err
			}
			return *v, nil
		default:
			return nil, domain.ErrInvalidSecretData
		}

	case domain.SecretTypeBinary:
		switch v := data.(type) {
		case domain.BinaryData:
			if err := validateBinaryData(v); err != nil {
				return nil, err
			}
			return v, nil
		case *domain.BinaryData:
			if v == nil {
				return nil, domain.ErrInvalidSecretData
			}
			if err := validateBinaryData(*v); err != nil {
				return nil, err
			}
			return *v, nil
		default:
			return nil, domain.ErrInvalidSecretData
		}
	default:
		return nil, domain.ErrInvalidSecretType
	}
}

func validateCredentialData(data domain.CredentialData) error {
	if strings.TrimSpace(data.Login) == "" {
		return fmt.Errorf("%w: credentials login is empty", domain.ErrInvalidSecretData)
	}

	if strings.TrimSpace(data.Password) == "" {
		return fmt.Errorf("%w: credentials password is empty", domain.ErrInvalidSecretData)
	}

	return nil
}

func validateTextData(data domain.TextData) error {
	if strings.TrimSpace(data.Text) == "" {
		return fmt.Errorf("%w: text secret is empty", domain.ErrInvalidSecretData)
	}

	return nil
}

func validateCardData(data domain.CardData) error {
	if strings.TrimSpace(data.Number) == "" {
		return fmt.Errorf("%w: card number is empty", domain.ErrInvalidSecretData)
	}

	if strings.TrimSpace(data.Cardholder) == "" {
		return fmt.Errorf("%w: cardholder is empty", domain.ErrInvalidSecretData)
	}

	if data.ExpiryMonth < 1 || data.ExpiryMonth > 12 {
		return fmt.Errorf("%w: invalid expiry month", domain.ErrInvalidSecretData)
	}

	if data.ExpiryYear == 0 {
		return fmt.Errorf("%w: invalid expiry year", domain.ErrInvalidSecretData)
	}

	if strings.TrimSpace(data.CVV) == "" {
		return fmt.Errorf("%w: card cvv is empty", domain.ErrInvalidSecretData)
	}

	return nil
}

func validateBinaryData(data domain.BinaryData) error {
	if strings.TrimSpace(data.Filename) == "" {
		return fmt.Errorf("%w: binary filename is empty", domain.ErrInvalidSecretData)
	}

	if strings.TrimSpace(data.MIMEType) == "" {
		return fmt.Errorf("%w: binary mime type is empty", domain.ErrInvalidSecretData)
	}

	if len(data.Content) == 0 {
		return fmt.Errorf("%w: binary content is empty", domain.ErrInvalidSecretData)
	}

	if len(data.Content) > maxBinaryPayloadSize {
		return fmt.Errorf("%w: binary content exceeds %d bytes", domain.ErrBinaryPayloadTooLarge, maxBinaryPayloadSize)
	}

	return nil
}

func newID() string {
	buf := make([]byte, 16)

	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("generate random id: %w", err))
	}

	return hex.EncodeToString(buf)
}
