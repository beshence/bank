package auth

import (
	"bank/internal/auth"
	"errors"
	"net/http"

	"bank/internal/app"
	"bank/internal/database/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type loginRequest struct {
	Login    string `json:"login" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

func LoginV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil || deps.AccessJWTManager == nil || deps.RefreshJWTManager == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "auth is not configured",
			})
			return
		}

		var request loginRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid request body",
			})
			return
		}

		var account models.Account
		if err := deps.DB.Where("login = ?", request.Login).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "invalid credentials",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load account",
			})
			return
		}

		ok, err := auth.VerifyPassword(request.Password, account.PasswordHash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to verify password",
			})
			return
		}

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "invalid credentials",
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
