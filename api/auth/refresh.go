package auth

import (
	"errors"
	"net/http"
	"vault/internal/app"
	"vault/internal/auth"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RefreshV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil || deps.AccessJWTManager == nil || deps.RefreshJWTManager == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "auth is not configured",
			})
			return
		}

		accountID, ok := middleware.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		sessionID, tokenRefreshTokenID, ok := middleware.GetCurrentSession(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		var session models.Session
		if err := deps.DB.Where("id = ? AND account_id = ?", sessionID, accountID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "invalid session",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load session",
			})
			return
		}

		if session.RefreshTokenID != tokenRefreshTokenID {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "refresh token is no longer valid",
			})
			return
		}

		var account models.Account
		if err := deps.DB.Select("id", "login").Where("id = ?", accountID).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "unauthorized",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load account",
			})
			return
		}

		tokens, err := auth.IssueTokenPairForExistingSession(deps.DB, deps.AccessJWTManager, deps.RefreshJWTManager, account, session, tokenRefreshTokenID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "refresh token is no longer valid",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to generate tokens",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":                 account.ID,
			"login":              account.Login,
			"token_type":         "Bearer",
			"access_token":       tokens.AccessToken,
			"access_expires_at":  tokens.AccessExpiresAt,
			"refresh_token":      tokens.RefreshToken,
			"refresh_expires_at": tokens.RefreshExpiresAt,
		})
	}
}
