package auth

import (
	"errors"
	"net/http"
	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func MeV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "database is not configured",
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

		c.JSON(http.StatusOK, gin.H{
			"id":    account.ID,
			"login": account.Login,
		})
	}
}
