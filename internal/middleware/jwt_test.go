package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"vault/internal/auth"

	"github.com/gin-gonic/gin"
)

func TestRequireJWT_AllowsExpectedTokenType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	manager := auth.NewJWTManager("test-secret", time.Hour, auth.TokenTypeAccess)
	token, _, err := manager.GenerateToken("session-id", "account-id", "refresh-token-id")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx.Request = req

	called := false
	RequireJWT(manager, auth.TokenTypeAccess)(ctx)
	if !ctx.IsAborted() {
		called = true
	}

	if !called {
		t.Fatal("RequireJWT() expected request to pass")
	}
}

func TestRequireJWT_RejectsUnexpectedTokenType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	refreshManager := auth.NewJWTManager("test-secret", 24*time.Hour, auth.TokenTypeRefresh)
	refreshToken, _, err := refreshManager.GenerateToken("session-id", "account-id", "refresh-token-id")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	ctx.Request = req

	accessParserManager := auth.NewJWTManager("test-secret", time.Hour, auth.TokenTypeAccess)
	RequireJWT(accessParserManager, auth.TokenTypeAccess)(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("RequireJWT() status = %d, expected %d", recorder.Code, http.StatusUnauthorized)
	}
}
