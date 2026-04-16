// Package repository содержит контрактные интерфейсы слоя доступа к данным.
package repository

import (
	"context"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
)

// UserRepository описывает операции хранения пользователей.
type UserRepository interface {
	// Create сохраняет нового пользователя.
	Create(ctx context.Context, user *domain.User) error

	// GetByID возвращает пользователя по идентификатору.
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByLogin возвращает пользователя по логину.
	GetByLogin(ctx context.Context, login string) (*domain.User, error)
}

// SessionRepository описывает операции хранения пользовательских сессий.
type SessionRepository interface {
	// Create сохраняет новую сессию.
	Create(ctx context.Context, session *domain.Session) error

	// GetByID возвращает сессию по идентификатору.
	GetByID(ctx context.Context, id string) (*domain.Session, error)

	// DeleteByID удаляет сессию по идентификатору.
	DeleteByID(ctx context.Context, id string) error

	// DeleteByUserID удаляет все сессии пользователя.
	DeleteByUserID(ctx context.Context, userID string) error
}

// SecretRepository описывает операции хранения пользовательских секретов.
type SecretRepository interface {
	// Create сохраняет новый секрет.
	Create(ctx context.Context, item *domain.SecretItem) error

	// Update обновляет существующий секрет.
	Update(ctx context.Context, item *domain.SecretItem) error

	// GetByID возвращает секрет по идентификатору и владельцу.
	GetByID(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error)

	// ListByOwner возвращает все секреты пользователя.
	ListByOwner(ctx context.Context, ownerID string) ([]*domain.SecretItem, error)

	// SoftDelete выполняет мягкое удаление секрета.
	SoftDelete(ctx context.Context, ownerID, secretID string, deletedAt time.Time) error

	// ListChangesSince возвращает изменения пользователя,
	// произошедшие после указанного времени.
	ListChangesSince(ctx context.Context, ownerID string, since time.Time) ([]*domain.SecretItem, error)
}
