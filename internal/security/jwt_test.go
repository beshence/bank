package security

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 5*time.Minute)

	token, expiresAt, err := manager.GenerateToken("user-id", "user-login")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken() token is empty")
	}

	if expiresAt <= time.Now().UTC().Unix() {
		t.Fatalf("GenerateToken() expiresAt = %d, expected in the future", expiresAt)
	}
}
