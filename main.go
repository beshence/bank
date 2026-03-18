package main

import (
	"net/http"

	v1handlers "vault/api/v1"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	v1 := router.Group("/api/v1")

	v1.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	v1handlers.Auth(v1.Group("/auth"))

	err := router.Run(":27462")
	if err != nil {
		return
	}
}
