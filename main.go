package main

import (
	"net/http"
	"vault/misc/versioning"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	api := router.Group("/api")

	// ping
	versioning.RegisterVersionedRoute(api, http.MethodGet, versioning.EndpointPing)

	// auth
	versioning.RegisterVersionedRoute(api, http.MethodPost, versioning.EndpointRegister)
	versioning.RegisterVersionedRoute(api, http.MethodPost, versioning.EndpointLogin)

	err := router.Run(":27462")
	if err != nil {
		return
	}
}
