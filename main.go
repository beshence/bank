package main

import (
	"log"
	"net/http"
	"vault/internal/app"
	"vault/internal/config"
	"vault/internal/database"
	"vault/internal/middleware"
	"vault/internal/security"
	"vault/internal/versioning"

	"github.com/gin-gonic/gin"
)

func main() {
	env, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.New(env.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	jwtManager := security.NewJWTManager(env.JWTSecret, env.JWTTTLSeconds)
	deps := app.NewDependencies(db, jwtManager)
	handlersByVersion := versioning.NewHandlersByVersion(deps)

	router := gin.Default()

	api := router.Group("/api")

	// ping
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointPing)

	// auth
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointRegister)
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointLogin)
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointMe, middleware.RequireJWT(jwtManager))

	err = router.Run(":27462")
	if err != nil {
		return
	}
}
