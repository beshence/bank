package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vault/internal/security"

	"github.com/gin-gonic/gin"
)

func TestRequireJWT_AllowsExpectedTokenType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	manager := security.NewJWTManager("test-secret", time.Hour, security.TokenTypeAccess)
	token, _, err := manager.GenerateToken("session-id", "user-id", "access-token-id")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx.Request = req

	called := false
	RequireJWT(manager, security.TokenTypeAccess)(ctx)
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

	refreshManager := security.NewJWTManager("test-secret", 24*time.Hour, security.TokenTypeRefresh)
	refreshToken, _, err := refreshManager.GenerateToken("session-id", "user-id", "access-token-id")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	ctx.Request = req

	accessParserManager := security.NewJWTManager("test-secret", time.Hour, security.TokenTypeAccess)
	RequireJWT(accessParserManager, security.TokenTypeAccess)(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("RequireJWT() status = %d, expected %d", recorder.Code, http.StatusUnauthorized)
	}
}
