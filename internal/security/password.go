package security

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// PasswordManager описывает работу с паролями.
type PasswordManager interface {
	// Hash создаёт bcrypt-хэш пароля.
	Hash(password string) (string, error)

	// Compare сравнивает хэш и открытый пароль.
	// Если пароль не совпал, возвращает matched=false и err=nil.
	Compare(hash, password string) (matched bool, err error)
}

// BcryptManager реализует PasswordManager через bcrypt.
type BcryptManager struct {
	cost int
}

// NewBcryptManager создаёт менеджер паролей на основе bcrypt.
func NewBcryptManager(cost int) *BcryptManager {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	return &BcryptManager{cost: cost}
}

// Hash создаёт bcrypt-хэш пароля.
func (m *BcryptManager) Hash(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("password is empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), m.cost)
	if err != nil {
		return "", fmt.Errorf("generate password hash: %w", err)
	}

	return string(hash), nil
}

// Compare сравнивает хэш и открытый пароль.
func (m *BcryptManager) Compare(hash, password string) (bool, error) {
	if strings.TrimSpace(hash) == "" {
		return false, fmt.Errorf("password hash is empty")
	}

	if password == "" {
		return false, nil
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, nil
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}

	return false, fmt.Errorf("compare password and hash: %w", err)
}

var _ PasswordManager = (*BcryptManager)(nil)
