package auth

import (
	"errors"
	"net/http"
	"vault/internal/auth"

	"vault/internal/app"
	"vault/internal/database/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type registerRequest struct {
	Login    string `json:"login" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

func RegisterV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil || deps.AccessJWTManager == nil || deps.RefreshJWTManager == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "auth is not configured",
			})
			return
		}

		var request registerRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid request body",
			})
			return
		}

		passwordHash, err := auth.HashPassword(request.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to hash password",
			})
			return
		}

		account := models.Account{
			Login:        request.Login,
			PasswordHash: passwordHash,
		}

		if err := deps.DB.Create(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{
					"message": "login is already taken",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to create account",
			})
			return
		}

		tokens, err := auth.IssueTokenPairForNewSession(deps.DB, deps.AccessJWTManager, deps.RefreshJWTManager, account, c.GetHeader("User-Agent"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to generate tokens",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
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
