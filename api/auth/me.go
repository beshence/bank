package auth

import (
	"net/http"
	"vault/internal/app"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
)

func MeV1dot0(_ *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, login, ok := middleware.GetCurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":    userID,
			"login": login,
		})
	}
}
