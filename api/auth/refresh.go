package auth

import (
	"net/http"

	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RefreshV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.AccessJWTManager == nil || deps.RefreshJWTManager == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "auth is not configured",
			})
			return
		}

		userID, login, ok := middleware.GetCurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		tokens, err := issueTokenPair(deps, models.User{ID: userID, Login: login})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to generate tokens",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":                 userID,
			"login":              login,
			"token_type":         "Bearer",
			"access_token":       tokens.AccessToken,
			"access_expires_at":  tokens.AccessExpiresAt,
			"refresh_token":      tokens.RefreshToken,
			"refresh_expires_at": tokens.RefreshExpiresAt,
		})
	}
}
