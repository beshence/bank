package auth

import (
	"net/http"

	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/security"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type loginRequest struct {
	Login    string `json:"login" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

func LoginV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil || deps.JWTManager == nil {
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

		var user models.User
		if err := deps.DB.Where("login = ?", request.Login).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "invalid credentials",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load user",
			})
			return
		}

		ok, err := security.VerifyPassword(request.Password, user.PasswordHash)
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

		token, expiresAt, err := deps.JWTManager.GenerateToken(user.ID, user.Login)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to generate token",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":         user.ID,
			"login":      user.Login,
			"token":      token,
			"token_type": "Bearer",
			"expires_at": expiresAt,
		})
	}
}
