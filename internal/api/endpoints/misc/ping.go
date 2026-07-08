package misc

import (
	"bank/internal/config"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PingV1(versions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := config.Load()
		if err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, gin.H{
			"err":  "0",
			"ping": "beshence-pong!",
			"id":   cfg.BankID,
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
