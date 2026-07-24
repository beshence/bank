package auth

import (
	"bank/internal/api"
	"bank/internal/auth"
	"bank/internal/database/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RefreshV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil || deps.AccessJWTManager == nil || deps.RefreshJWTManager == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "auth is not configured",
			})
			return
		}

		accountID, ok := auth.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "unauthorized",
			})
			return
		}

		sessionID, tokenRefreshTokenID, ok := auth.GetCurrentSession(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "unauthorized",
			})
			return
		}

		var session models.Session
		if err := deps.DB.Where("id = ? AND account_id = ?", sessionID, accountID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid session",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load session",
			})
			return
		}

		if session.RefreshTokenID != tokenRefreshTokenID {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "refresh token is no longer valid",
			})
			return
		}

		// TODO: use middleware
		var account models.Account
		if err := deps.DB.Select("id", "username").Where("id = ?", accountID.String()).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load account",
			})
			return
		}

		tokens, err := auth.IssueTokenPairForExistingSession(deps.DB, deps.RefreshJWTManager, deps.AccessJWTManager, account, session, tokenRefreshTokenID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "refresh token is no longer valid",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to generate tokens",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"err":           "0",
			"id":            account.ID,
			"username":      account.Username,
			"refresh_token": tokens.RefreshToken,
			"access_token":  tokens.AccessToken,
		})
	}
}
