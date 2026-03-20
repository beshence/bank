package auth

import (
	"errors"
	"net/http"
	"strings"

	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/security"

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

		passwordHash, err := security.HashPassword(request.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to hash password",
			})
			return
		}

		user := models.User{
			Login:        request.Login,
			PasswordHash: passwordHash,
		}

		if err := deps.DB.Create(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{
					"message": "login is already taken",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to create user",
			})
			return
		}

		sessionName := strings.TrimSpace(c.GetHeader("User-Agent"))
		tokens, err := issueTokenPairForNewSession(deps, user, sessionName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to generate tokens",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":                 user.ID,
			"login":              user.Login,
			"token_type":         "Bearer",
			"access_token":       tokens.AccessToken,
			"access_expires_at":  tokens.AccessExpiresAt,
			"refresh_token":      tokens.RefreshToken,
			"refresh_expires_at": tokens.RefreshExpiresAt,
		})
	}
}
