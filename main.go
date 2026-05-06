package main

import (
	"log"
	"net/http"
	"vault/internal/app"
	"vault/internal/auth"
	"vault/internal/config"
	"vault/internal/database"
	"vault/internal/middleware"
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

	accessJWTManager := auth.NewJWTManager(env.JWTSecret, env.AccessJWTTTLSeconds, auth.TokenTypeAccess)
	refreshJWTManager := auth.NewJWTManager(env.JWTSecret, env.RefreshJWTTTLSeconds, auth.TokenTypeRefresh)
	deps := app.NewDependencies(db, accessJWTManager, refreshJWTManager)
	handlersByVersion := versioning.NewHandlersByVersion(deps)

	router := gin.Default()

	api := router.Group("/api")

	// ping
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointPing)

	// auth
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointRegister)
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointLogin)
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointRefresh, middleware.RequireJWT(refreshJWTManager, auth.TokenTypeRefresh))
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointMe, middleware.RequireJWT(accessJWTManager, auth.TokenTypeAccess))

	// vaults
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointVaults, middleware.RequireJWT(accessJWTManager, auth.TokenTypeAccess))
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointVaults, middleware.RequireJWT(accessJWTManager, auth.TokenTypeAccess))

	// chains
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointVaultChains, middleware.RequireJWT(accessJWTManager, auth.TokenTypeAccess))
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointVaultChains, middleware.RequireJWT(accessJWTManager, auth.TokenTypeAccess))
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodGet, versioning.EndpointVaultChainEvents, middleware.RequireJWT(accessJWTManager, auth.TokenTypeAccess))
	versioning.RegisterVersionedRoute(api, handlersByVersion, http.MethodPost, versioning.EndpointVaultChainEvents, middleware.RequireJWT(accessJWTManager, auth.TokenTypeAccess))

	err = router.Run(":27462")
	if err != nil {
		log.Fatal(err)
	}
}
