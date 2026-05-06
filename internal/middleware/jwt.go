package middleware

import (
	"net/http"
	"strings"
	"vault/internal/auth"

	"github.com/gin-gonic/gin"
)

func RequireJWT(jwtManager *auth.JWTManager, expectedTokenType auth.TokenType) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtManager == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "auth is not configured",
			})
			return
		}

		authorizationHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "missing or invalid authorization header",
			})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "missing bearer token",
			})
			return
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "invalid token",
			})
			return
		}

		authClaims, claimsOk := auth.AuthClaimsFromToken(claims)
		if !claimsOk {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "invalid token claims",
			})
			return
		}

		if authClaims.TokenType != expectedTokenType {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "invalid token type",
			})
			return
		}

		c.Set(ContextAuthClaimsKey, claims)
		c.Set(ContextAuthSessionIDKey, authClaims.SessionID)
		c.Set(ContextAuthAccountIDKey, authClaims.AccountID)
		c.Set(ContextAuthRefreshTokenIDKey, authClaims.RefreshTokenID)
		c.Next()
	}
}
