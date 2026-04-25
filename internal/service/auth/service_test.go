package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	"github.com/Dyuzhovsergey/gophkeeper/internal/security"
)

// userRepositoryStub реализует тестовый stub репозитория пользователей.
type userRepositoryStub struct {
	createFn     func(ctx context.Context, user *domain.User) error
	getByLoginFn func(ctx context.Context, login string) (*domain.User, error)
	getByIDFn    func(ctx context.Context, userID string) (*domain.User, error)
}

// Create сохраняет пользователя в тестовом stub.
func (s *userRepositoryStub) Create(ctx context.Context, user *domain.User) error {
	if s.createFn == nil {
		return nil
	}
	return s.createFn(ctx, user)
}

// GetByLogin возвращает пользователя по логину из тестового stub.
func (s *userRepositoryStub) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	if s.getByLoginFn == nil {
		return nil, nil
	}
	return s.getByLoginFn(ctx, login)
}

// GetByID возвращает пользователя по идентификатору из тестового stub.
func (s *userRepositoryStub) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	if s.getByIDFn == nil {
		return nil, nil
	}
	return s.getByIDFn(ctx, userID)
}

// sessionRepositoryStub реализует тестовый stub репозитория сессий.
type sessionRepositoryStub struct {
	createFn         func(ctx context.Context, session *domain.Session) error
	getByIDFn        func(ctx context.Context, sessionID string) (*domain.Session, error)
	deleteByIDFn     func(ctx context.Context, sessionID string) error
	deleteByUserIDFn func(ctx context.Context, userID string) error
}

// Create сохраняет сессию в тестовом stub.
func (s *sessionRepositoryStub) Create(ctx context.Context, session *domain.Session) error {
	if s.createFn == nil {
		return nil
	}
	return s.createFn(ctx, session)
}

// GetByID возвращает сессию по идентификатору из тестового stub.
func (s *sessionRepositoryStub) GetByID(ctx context.Context, sessionID string) (*domain.Session, error) {
	if s.getByIDFn == nil {
		return nil, nil
	}
	return s.getByIDFn(ctx, sessionID)
}

// DeleteByID удаляет одну сессию в тестовом stub.
func (s *sessionRepositoryStub) DeleteByID(ctx context.Context, sessionID string) error {
	if s.deleteByIDFn == nil {
		return nil
	}
	return s.deleteByIDFn(ctx, sessionID)
}

// DeleteByUserID удаляет все сессии пользователя в тестовом stub.
func (s *sessionRepositoryStub) DeleteByUserID(ctx context.Context, userID string) error {
	if s.deleteByUserIDFn == nil {
		return nil
	}
	return s.deleteByUserIDFn(ctx, userID)
}

// TestService_Register_Success проверяет успешную регистрацию пользователя.
func TestService_Register_Success(t *testing.T) {
	ctx := context.Background()
	fixedNow := time.Date(2026, 4, 23, 15, 0, 0, 0, time.UTC)

	var createdUser *domain.User

	users := &userRepositoryStub{
		getByLoginFn: func(ctx context.Context, login string) (*domain.User, error) {
			if login != "sergey" {
				t.Fatalf("unexpected login: got %q, want %q", login, "sergey")
			}
			return nil, nil
		},
		createFn: func(ctx context.Context, user *domain.User) error {
			createdUser = user
			return nil
		},
	}

	sessions := &sessionRepositoryStub{}
	passwordManager := security.NewBcryptManager(0)
	jwtManager := security.NewJWTManager("test-secret")

	svc := NewService(users, sessions, passwordManager, jwtManager, time.Hour)
	svc.now = func() time.Time { return fixedNow }

	user, err := svc.Register(ctx, "sergey", "qwerty")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.Login != "sergey" {
		t.Fatalf("unexpected login: got %q, want %q", user.Login, "sergey")
	}
	if createdUser == nil {
		t.Fatal("expected created user in repository")
	}
	if createdUser.PasswordHash == "" {
		t.Fatal("expected non-empty password hash")
	}
	if createdUser.PasswordHash == "qwerty" {
		t.Fatal("password hash must not equal raw password")
	}
}

// TestService_Register_DuplicateUser проверяет ошибку при попытке повторной регистрации.
func TestService_Register_DuplicateUser(t *testing.T) {
	ctx := context.Background()

	users := &userRepositoryStub{
		createFn: func(ctx context.Context, user *domain.User) error {
			if user == nil {
				t.Fatal("expected non-nil user")
			}
			if user.Login != "sergey" {
				t.Fatalf("unexpected login: got %q, want %q", user.Login, "sergey")
			}

			return domain.ErrUserAlreadyExists
		},
	}

	sessions := &sessionRepositoryStub{}
	passwordManager := security.NewBcryptManager(0)
	jwtManager := security.NewJWTManager("test-secret")

	svc := NewService(users, sessions, passwordManager, jwtManager, time.Hour)

	_, err := svc.Register(ctx, "sergey", "qwerty")
	if err == nil {
		t.Fatal("expected error for duplicate user")
	}
	if !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrUserAlreadyExists)
	}
}

// TestService_Login_Success проверяет успешный вход пользователя.
func TestService_Login_Success(t *testing.T) {
	ctx := context.Background()
	fixedNow := time.Date(2026, 4, 23, 16, 0, 0, 0, time.UTC)

	passwordManager := security.NewBcryptManager(0)
	hashedPassword, err := passwordManager.Hash("qwerty")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	users := &userRepositoryStub{
		getByLoginFn: func(ctx context.Context, login string) (*domain.User, error) {
			return &domain.User{
				ID:           "user-1",
				Login:        "sergey",
				PasswordHash: hashedPassword,
			}, nil
		},
	}

	var createdSession *domain.Session

	sessions := &sessionRepositoryStub{
		createFn: func(ctx context.Context, session *domain.Session) error {
			createdSession = session
			return nil
		},
	}

	jwtManager := security.NewJWTManager("test-secret")

	svc := NewService(users, sessions, passwordManager, jwtManager, time.Hour)
	svc.now = func() time.Time { return fixedNow }

	result, err := svc.Login(ctx, "sergey", "qwerty")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil login result")
	}
	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if result.Session == nil {
		t.Fatal("expected non-nil session")
	}
	if createdSession == nil {
		t.Fatal("expected created session in repository")
	}
	if createdSession.UserID != "user-1" {
		t.Fatalf("unexpected session user id: got %q, want %q", createdSession.UserID, "user-1")
	}
	if createdSession.ID == "" {
		t.Fatal("expected non-empty session id")
	}
}

// TestService_Login_InvalidPassword проверяет ошибку при неверном пароле.
func TestService_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()

	passwordManager := security.NewBcryptManager(0)
	hashedPassword, err := passwordManager.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	users := &userRepositoryStub{
		getByLoginFn: func(ctx context.Context, login string) (*domain.User, error) {
			return &domain.User{
				ID:           "user-1",
				Login:        "sergey",
				PasswordHash: hashedPassword,
			}, nil
		},
	}

	sessions := &sessionRepositoryStub{
		createFn: func(ctx context.Context, session *domain.Session) error {
			t.Fatal("Create session should not be called for invalid password")
			return nil
		},
	}

	jwtManager := security.NewJWTManager("test-secret")

	svc := NewService(users, sessions, passwordManager, jwtManager, time.Hour)

	_, err = svc.Login(ctx, "sergey", "wrong-password")
	if err == nil {
		t.Fatal("expected error for invalid password")
	}
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrInvalidCredentials)
	}
}

// TestService_Authenticate_Success проверяет успешную аутентификацию по токену.
func TestService_Authenticate_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 23, 17, 0, 0, 0, time.UTC)

	users := &userRepositoryStub{}
	sessions := &sessionRepositoryStub{
		getByIDFn: func(ctx context.Context, sessionID string) (*domain.Session, error) {
			if sessionID != "session-1" {
				t.Fatalf("unexpected session id: got %q, want %q", sessionID, "session-1")
			}

			return &domain.Session{
				ID:        "session-1",
				UserID:    "user-1",
				ExpiresAt: now.Add(time.Hour),
			}, nil
		},
	}

	passwordManager := security.NewBcryptManager(0)
	jwtManager := security.NewJWTManager("test-secret")

	svc := NewService(users, sessions, passwordManager, jwtManager, time.Hour)
	svc.now = func() time.Time { return now }

	token, err := jwtManager.Generate("user-1", "session-1", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	identity, err := svc.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}

	if identity == nil {
		t.Fatal("expected non-nil identity")
	}
	if identity.UserID != "user-1" {
		t.Fatalf("unexpected user id: got %q, want %q", identity.UserID, "user-1")
	}
	if identity.SessionID != "session-1" {
		t.Fatalf("unexpected session id: got %q, want %q", identity.SessionID, "session-1")
	}
}

// TestService_Authenticate_InvalidToken проверяет ошибку на невалидном токене.
func TestService_Authenticate_InvalidToken(t *testing.T) {
	ctx := context.Background()

	users := &userRepositoryStub{}
	sessions := &sessionRepositoryStub{}
	passwordManager := security.NewBcryptManager(0)
	jwtManager := security.NewJWTManager("test-secret")

	svc := NewService(users, sessions, passwordManager, jwtManager, time.Hour)

	_, err := svc.Authenticate(ctx, "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrInvalidToken)
	}
}

// TestService_Authenticate_SessionExpired проверяет ошибку на истёкшей серверной сессии.
func TestService_Authenticate_SessionExpired(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 23, 18, 0, 0, 0, time.UTC)

	users := &userRepositoryStub{}
	sessions := &sessionRepositoryStub{
		getByIDFn: func(ctx context.Context, sessionID string) (*domain.Session, error) {
			return &domain.Session{
				ID:        "session-1",
				UserID:    "user-1",
				ExpiresAt: now.Add(-time.Minute),
			}, nil
		},
	}

	passwordManager := security.NewBcryptManager(0)
	jwtManager := security.NewJWTManager("test-secret")

	svc := NewService(users, sessions, passwordManager, jwtManager, time.Hour)
	svc.now = func() time.Time { return now }

	token, err := jwtManager.Generate("user-1", "session-1", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	_, err = svc.Authenticate(ctx, token)
	if err == nil {
		t.Fatal("expected error for expired session")
	}
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Fatalf("unexpected error: got %v, want wrapped %v", err, domain.ErrSessionExpired)
	}
}
