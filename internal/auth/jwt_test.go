package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 5*time.Minute, TokenTypeAccess)

	token, expiresAt, err := manager.GenerateToken("session-id", "account-id", "refresh-token-id")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken() token is empty")
	}

	if expiresAt <= time.Now().UTC().Unix() {
		t.Fatalf("GenerateToken() expiresAt = %d, expected in the future", expiresAt)
	}

	claims, err := manager.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	tokenType, ok := TokenTypeFromClaims(claims)
	if !ok {
		t.Fatal("TokenTypeFromClaims() expected token type")
	}

	if tokenType != TokenTypeAccess {
		t.Fatalf("TokenTypeFromClaims() = %s, expected %s", tokenType, TokenTypeAccess)
	}

	authClaims, ok := AuthClaimsFromToken(claims)
	if !ok {
		t.Fatal("AuthClaimsFromToken() expected claims")
	}

	if authClaims.SessionID != "session-id" || authClaims.AccountID != "account-id" {
		t.Fatalf("AuthClaimsFromToken() unexpected claims = %+v", authClaims)
	}

	if authClaims.RefreshTokenID != "" {
		t.Fatalf("AuthClaimsFromToken() access token should not carry refresh token id, got %q", authClaims.RefreshTokenID)
	}
}

func TestTokenTypeFromClaims(t *testing.T) {
	accessManager := NewJWTManager("test-secret", 5*time.Minute, TokenTypeAccess)
	refreshManager := NewJWTManager("test-secret", 24*time.Hour, TokenTypeRefresh)

	accessToken, _, err := accessManager.GenerateToken("session-id", "account-id", "refresh-token-id")
	if err != nil {
		t.Fatalf("GenerateToken(access) error = %v", err)
	}

	refreshToken, _, err := refreshManager.GenerateToken("session-id", "account-id", "refresh-token-id")
	if err != nil {
		t.Fatalf("GenerateToken(refresh) error = %v", err)
	}

	accessClaims, err := accessManager.ParseToken(accessToken)
	if err != nil {
		t.Fatalf("ParseToken(access) error = %v", err)
	}

	refreshClaims, err := refreshManager.ParseToken(refreshToken)
	if err != nil {
		t.Fatalf("ParseToken(refresh) error = %v", err)
	}

	if tokenType, ok := TokenTypeFromClaims(accessClaims); !ok || tokenType != TokenTypeAccess {
		t.Fatalf("TokenTypeFromClaims(access) = %s, %v", tokenType, ok)
	}

	if tokenType, ok := TokenTypeFromClaims(refreshClaims); !ok || tokenType != TokenTypeRefresh {
		t.Fatalf("TokenTypeFromClaims(refresh) = %s, %v", tokenType, ok)
	}

	refreshAuthClaims, ok := AuthClaimsFromToken(refreshClaims)
	if !ok {
		t.Fatal("AuthClaimsFromToken(refresh) expected claims")
	}

	if refreshAuthClaims.RefreshTokenID != "refresh-token-id" {
		t.Fatalf("AuthClaimsFromToken(refresh) refresh token id = %q, expected %q", refreshAuthClaims.RefreshTokenID, "refresh-token-id")
	}
}
