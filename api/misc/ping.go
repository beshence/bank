package misc

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func PingV1(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
