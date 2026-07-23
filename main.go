package main

import (
	"bank/internal/api"
	"bank/internal/api/versioning"
	"bank/internal/auth"
	"bank/internal/database"
	"bank/internal/environment"
	"bank/internal/gateway"
	"bank/internal/settings"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	env, err := environment.Load()
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

	refreshJWT := auth.NewJWTManager(
		env.JWTSecret,
		env.RefreshJWTTTLSeconds,
		auth.TokenTypeRefresh,
	)

	accessJWT := auth.NewJWTManager(
		env.JWTSecret,
		env.AccessJWTTTLSeconds,
		auth.TokenTypeAccess,
	)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"*"},
		MaxAge:          24 * time.Hour,
	}))

	dependencies := api.NewDependencies(
		db,
		refreshJWT,
		accessJWT,
	)

	versionedEndpoints := versioning.GetVersionedEndpoints(dependencies)

	apiRoute := router.Group("/api")
	versioning.RegisterVersionedRoutes(apiRoute, versionedEndpoints)

	port := os.Getenv("BANK_PORT")

	if port == "" {
		port = "27462"
	}

	settings.InitAPIUrls(port)

	gateway.StartPublisher(db)

	log.Fatal(router.Run(":" + port))
}
