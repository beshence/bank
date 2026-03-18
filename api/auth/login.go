package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func LoginV1(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "login endpoint not implemented yet",
	})
}
