package runner

import (
	"bank/internal/api"
	"bank/internal/api/versioning"
	"bank/internal/auth"
	"bank/internal/database"
	"bank/internal/environment"
	"bank/internal/gateway"
	"bank/internal/gateway/webrtc"
	"bank/internal/settings"
	"bank/internal/tls"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Run() error {
	env, err := environment.Load()
	if err != nil {
		return err
	}

	db, err := database.New(env.DatabaseURL)
	if err != nil {
		return err
	}

	if err := database.Migrate(db); err != nil {
		return err
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

	gatewayURL := "https://gateway.beshence.com/api"

	bankID := settings.GetBankID(db)

	token, err := gateway.GetGatewayToken(db, gatewayURL, bankID)

	if err != nil {

		return err
	}

	err = webrtc.Start(bankID, token, router)

	if err != nil {

		return err
	}

	disableTls := os.Getenv("BANK_DISABLE_TLS")
	disableTlsBool := false
	if disableTls == "true" || disableTls == "1" {
		disableTlsBool = true
	}

	if !disableTlsBool {
		tlsConfig, err := tls.GetTLSConfig(db)

		if err != nil {

			return err
		}

		server := &http.Server{
			Addr:      ":" + port,
			Handler:   router,
			TLSConfig: tlsConfig,
		}

		return server.ListenAndServeTLS("", "")
	} else {
		return router.Run(":" + port)
	}
}
