package main

import (
	"bank/internal/api"
	"bank/internal/api/versioning"
	"bank/internal/auth"
	"bank/internal/config"
	"bank/internal/database"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	refreshJWT := auth.NewJWTManager(
		cfg.JWTSecret,
		cfg.RefreshJWTTTLSeconds,
		auth.TokenTypeRefresh,
	)

	accessJWT := auth.NewJWTManager(
		cfg.JWTSecret,
		cfg.AccessJWTTTLSeconds,
		auth.TokenTypeAccess,
	)

	router := gin.Default()
	router.GET("/.well-known/beshence/bank", func(c *gin.Context) {
		c.Request.URL.Path = "/api/ping"
		c.Request.URL.RawPath = ""

		router.HandleContext(c)
	})

	dependencies := api.NewDependencies(
		db,
		refreshJWT,
		accessJWT,
	)

	versionedEndpoints := versioning.GetVersionedEndpoints(dependencies)

	apiRoute := router.Group("/api")
	versioning.RegisterVersionedRoutes(apiRoute, versionedEndpoints)

	err = router.Run(":27462")
	if err != nil {
		log.Fatal(err)
	}
}
