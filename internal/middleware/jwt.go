package middleware

import (
	"net/http"
	"strings"

	"vault/internal/security"

	"github.com/gin-gonic/gin"
)

func RequireJWT(jwtManager *security.JWTManager, expectedTokenType security.TokenType) gin.HandlerFunc {
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

		authClaims, claimsOk := security.AuthClaimsFromToken(claims)
		if !claimsOk || authClaims.TokenType != expectedTokenType {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "invalid token type",
			})
			return
		}

		c.Set(ContextAuthClaimsKey, claims)
		c.Set(ContextAuthSessionIDKey, authClaims.SessionID)
		c.Set(ContextAuthUserIDKey, authClaims.AccountID)
		c.Set(ContextAuthAccessTokenIDKey, authClaims.AccessTokenID)
		c.Next()
	}
}
