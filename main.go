package main

import (
	"vault/misc/versioning"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	api := router.Group("/api")

	// ping
	api.GET(versioning.EndpointPing, versioning.RouteByVersion(versioning.EndpointPing))

	// auth
	api.POST(versioning.EndpointRegister, versioning.RouteByVersion(versioning.EndpointRegister))
	api.POST(versioning.EndpointLogin, versioning.RouteByVersion(versioning.EndpointLogin))

	err := router.Run(":27462")
	if err != nil {
		return
	}
}
