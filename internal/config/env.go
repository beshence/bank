package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var (
	ErrDatabaseURLRequired = errors.New("DATABASE_URL is required")
	ErrJWTSecretRequired   = errors.New("JWT_SECRET is required")
	ErrInvalidJWTTTL       = errors.New("JWT_TTL_SECONDS must be a positive integer")
)

const internalDatabaseURLDefault = "postgres://vault:vault@postgres:5432/vault?sslmode=disable"

type Env struct {
	DatabaseURL   string
	JWTSecret     string
	JWTTTLSeconds time.Duration
}

func Load() (Env, error) {
	useInternalDB := os.Getenv("USE_INTERNAL_DB") == "true"
	if !useInternalDB {
		_ = godotenv.Load()
	}

	databaseURL := resolveDatabaseURL(useInternalDB)
	if databaseURL == "" {
		return Env{}, ErrDatabaseURLRequired
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Env{}, ErrJWTSecretRequired
	}

	jwtTTLRaw := os.Getenv("JWT_TTL_SECONDS")
	if jwtTTLRaw == "" {
		jwtTTLRaw = "3600"
	}

	jwtTTLSeconds, err := strconv.Atoi(jwtTTLRaw)
	if err != nil || jwtTTLSeconds <= 0 {
		return Env{}, ErrInvalidJWTTTL
	}

	return Env{
		DatabaseURL:   databaseURL,
		JWTSecret:     jwtSecret,
		JWTTTLSeconds: time.Duration(jwtTTLSeconds) * time.Second,
	}, nil
}

func resolveDatabaseURL(useInternalDB bool) string {
	if useInternalDB {
		if internalDatabaseURL := os.Getenv("INTERNAL_DATABASE_URL"); internalDatabaseURL != "" {
			return internalDatabaseURL
		}

		return internalDatabaseURLDefault
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return ""
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ""
	}

	return databaseURL
}
