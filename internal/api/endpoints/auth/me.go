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

func MeV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "database is not configured",
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

		c.JSON(http.StatusOK, gin.H{
			"err":      "0",
			"id":       account.ID,
			"username": account.Username,
		})
	}
}
