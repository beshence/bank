package auth

import (
	"errors"
	"net/http"

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
		if deps == nil || deps.DB == nil || deps.JWTManager == nil {
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

		token, expiresAt, err := deps.JWTManager.GenerateToken(user.ID, user.Login)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to generate token",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":         user.ID,
			"login":      user.Login,
			"token":      token,
			"token_type": "Bearer",
			"expires_at": expiresAt,
		})
	}
}
