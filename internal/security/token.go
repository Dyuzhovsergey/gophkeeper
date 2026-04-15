package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// TokenGenerator описывает генерацию токенов доступа.
type TokenGenerator interface {
	// Generate создаёт новый токен.
	Generate() (string, error)
}

// RandomTokenGenerator генерирует случайные токены.
type RandomTokenGenerator struct {
	size int
}

const defaultTokenSize = 32

// NewRandomTokenGenerator создаёт генератор случайных токенов.
func NewRandomTokenGenerator(size int) *RandomTokenGenerator {
	if size <= 0 {
		size = defaultTokenSize
	}

	return &RandomTokenGenerator{size: size}
}

// Generate создаёт новый случайный токен.
func (g *RandomTokenGenerator) Generate() (string, error) {
	buf := make([]byte, g.size)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

var _ TokenGenerator = (*RandomTokenGenerator)(nil)