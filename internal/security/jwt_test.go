package security

import (
	"testing"
	"time"
)

// TestJWTManager_GenerateAndParse_Success проверяет успешные generate и parse токена.
func TestJWTManager_GenerateAndParse_Success(t *testing.T) {
	manager := NewJWTManager("test-secret")

	expiresAt := time.Now().UTC().Add(time.Hour)

	token, err := manager.Generate("user-1", "session-1", expiresAt)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if claims.Subject != "user-1" {
		t.Fatalf("unexpected subject: got %q, want %q", claims.Subject, "user-1")
	}
	if claims.ID != "session-1" {
		t.Fatalf("unexpected id: got %q, want %q", claims.ID, "session-1")
	}
}

// TestJWTManager_Parse_InvalidSignature проверяет ошибку при неверном secret.
func TestJWTManager_Parse_InvalidSignature(t *testing.T) {
	generator := NewJWTManager("secret-1")
	parser := NewJWTManager("secret-2")

	expiresAt := time.Now().UTC().Add(time.Hour)

	token, err := generator.Generate("user-1", "session-1", expiresAt)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	_, err = parser.Parse(token)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

// TestJWTManager_Parse_MalformedToken проверяет ошибку при некорректном формате токена.
func TestJWTManager_Parse_MalformedToken(t *testing.T) {
	manager := NewJWTManager("test-secret")

	_, err := manager.Parse("not-a-jwt-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

// TestJWTManager_Parse_ExpiredToken проверяет ошибку для истёкшего токена.
func TestJWTManager_Parse_ExpiredToken(t *testing.T) {
	manager := NewJWTManager("test-secret")

	expiresAt := time.Now().UTC().Add(-time.Minute)

	token, err := manager.Generate("user-1", "session-1", expiresAt)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	_, err = manager.Parse(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}
