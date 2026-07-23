package misc

import (
	"bank/internal/api"
	"bank/internal/settings"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PingV1(deps *api.Dependencies, versions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "UNKNOWN", "errmsg": "database is not configured"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"err":  "0",
			"ping": "beshence-bank-pong!",
			"id":   settings.GetBankID(deps.DB),
			"api": gin.H{
				"urls":     settings.GetAPIUrls(),
				"versions": versions,
			},
			"auth": gin.H{
				"login": gin.H{
					"methods": []string{"usernameAndPassword"},
				},
				"register": gin.H{
					"methods": []string{"usernameAndPassword"},
				},
			},
		})
	}
}
