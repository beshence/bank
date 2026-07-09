package misc

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func PingV1(versions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"err":  "0",
			"ping": "beshence-pong!",
			"api": gin.H{
				"urls":     []string{"https://127.0.0.1:27462/api"},
				"versions": versions,
			},
			"auth": gin.H{
				"methods": []string{"usernameAndPassword"},
			},
		})
	}
}
