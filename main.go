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

	accessJWTManager := security.NewJWTManager(env.JWTSecret, env.AccessJWTTTLSeconds, security.TokenTypeAccess)
	refreshJWTManager := security.NewJWTManager(env.JWTSecret, env.RefreshJWTTTLSeconds, security.TokenTypeRefresh)
	deps := app.NewDependencies(db, accessJWTManager, refreshJWTManager)
	handlersByVersion := versioning.NewHandlersByVersion(deps)

	router := gin.Default()

	api := router.Group("/api")

	// ping
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointPing)

	// auth
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointRegister)
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointLogin)
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointRefresh, middleware.RequireJWT(refreshJWTManager, security.TokenTypeRefresh))
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointMe, middleware.RequireJWT(accessJWTManager, security.TokenTypeAccess))

	// repositories
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointRepo, middleware.RequireJWT(accessJWTManager, security.TokenTypeAccess))
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointRepo, middleware.RequireJWT(accessJWTManager, security.TokenTypeAccess))

	err = router.Run(":27462")
	if err != nil {
		log.Fatal(err)
	}
}
