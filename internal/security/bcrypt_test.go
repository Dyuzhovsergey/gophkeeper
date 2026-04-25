package security

import "testing"

// TestBcryptManager_Hash_Success проверяет успешное создание hash пароля.
func TestBcryptManager_Hash_Success(t *testing.T) {
	manager := NewBcryptManager(0)

	hash, err := manager.Hash("qwerty123")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "qwerty123" {
		t.Fatal("hash must not equal raw password")
	}
}

// TestBcryptManager_Compare_Success проверяет успешное сравнение корректного пароля.
func TestBcryptManager_Compare_Success(t *testing.T) {
	manager := NewBcryptManager(0)

	hash, err := manager.Hash("qwerty123")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	matched, err := manager.Compare(hash, "qwerty123")
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if !matched {
		t.Fatal("expected matched=true for correct password")
	}
}

// TestBcryptManager_Compare_WrongPassword проверяет отклонение неверного пароля.
func TestBcryptManager_Compare_WrongPassword(t *testing.T) {
	manager := NewBcryptManager(0)

	hash, err := manager.Hash("qwerty123")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	matched, err := manager.Compare(hash, "wrong-password")
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if matched {
		t.Fatal("expected matched=false for wrong password")
	}
}

// TestBcryptManager_Compare_InvalidHash проверяет поведение при некорректном hash.
func TestBcryptManager_Compare_InvalidHash(t *testing.T) {
	manager := NewBcryptManager(0)

	_, err := manager.Compare("not-a-valid-bcrypt-hash", "qwerty123")
	if err == nil {
		t.Fatal("expected error for invalid bcrypt hash")
	}
}
