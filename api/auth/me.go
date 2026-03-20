package auth

import (
	"net/http"
	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
)

func MeV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "database is not configured",
			})
			return
		}

		userID, ok := middleware.GetCurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		var user models.User
		if err := deps.DB.Select("id", "login").Where("id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":    user.ID,
			"login": user.Login,
		})
	}
}
