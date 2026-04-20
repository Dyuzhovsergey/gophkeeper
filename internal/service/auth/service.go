package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	repo "github.com/Dyuzhovsergey/gophkeeper/internal/repository"
	"github.com/Dyuzhovsergey/gophkeeper/internal/security"
)

const defaultSessionTTL = 24 * time.Hour

// Service реализует бизнес-логику аутентификации и авторизации.
type Service struct {
	users      repo.UserRepository
	sessions   repo.SessionRepository
	passwords  security.PasswordManager
	tokens     security.TokenManager
	sessionTTL time.Duration
	now        func() time.Time
}

// LoginResult описывает результат успешного входа пользователя.
type LoginResult struct {
	// Token — JWT токен доступа.
	Token string

	// Session — созданная сервером сессия.
	Session *domain.Session

	// User — пользователь, для которого выполнен вход.
	User *domain.User
}

// Identity описывает подтверждённую авторизацию пользователя.
type Identity struct {
	// UserID — идентификатор пользователя.
	UserID string

	// SessionID — идентификатор активной сессии.
	SessionID string
}

// NewService создаёт AuthService.
func NewService(
	users repo.UserRepository,
	sessions repo.SessionRepository,
	passwords security.PasswordManager,
	tokens security.TokenManager,
	sessionTTL time.Duration,
) *Service {
	if sessionTTL <= 0 {
		sessionTTL = defaultSessionTTL
	}

	return &Service{
		users:      users,
		sessions:   sessions,
		passwords:  passwords,
		tokens:     tokens,
		sessionTTL: sessionTTL,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// Register регистрирует нового пользователя.
func (s *Service) Register(ctx context.Context, login, password string) (*domain.User, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, fmt.Errorf("login is empty")
	}

	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("password is empty")
	}

	passwordHash, err := s.passwords.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := s.now()

	user := &domain.User{
		ID:           newID(),
		Login:        login,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("validate user: %w", err)
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login выполняет вход пользователя и создаёт новую сессию.
func (s *Service) Login(ctx context.Context, login, password string) (*LoginResult, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, fmt.Errorf("login is empty")
	}

	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}

		return nil, fmt.Errorf("get user by login: %w", err)
	}

	matched, err := s.passwords.Compare(user.PasswordHash, password)
	if err != nil {
		return nil, fmt.Errorf("compare password: %w", err)
	}
	if !matched {
		return nil, domain.ErrInvalidCredentials
	}

	now := s.now()

	session := &domain.Session{
		ID:        newID(),
		UserID:    user.ID,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
	}

	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("validate session: %w", err)
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	token, err := s.tokens.Generate(user.ID, session.ID, session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("generate jwt token: %w", err)
	}

	return &LoginResult{
		Token:   token,
		Session: session,
		User:    user,
	}, nil
}

// Authenticate проверяет JWT и подтверждает, что серверная сессия активна.
func (s *Service) Authenticate(ctx context.Context, token string) (*Identity, error) {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	session, err := s.sessions.GetByID(ctx, claims.ID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return nil, domain.ErrUnauthorized
		}

		return nil, fmt.Errorf("get session by id: %w", err)
	}

	now := s.now()

	if now.After(session.ExpiresAt) {
		return nil, domain.ErrSessionExpired
	}

	if session.UserID != claims.Subject {
		return nil, domain.ErrUnauthorized
	}

	return &Identity{
		UserID:    session.UserID,
		SessionID: session.ID,
	}, nil
}

// newID создаёт новый строковый идентификатор.
func newID() string {
	buf := make([]byte, 16)

	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("generate random id: %w", err))
	}

	return hex.EncodeToString(buf)
}
