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
	ErrDatabaseURLInvalid  = errors.New("DATABASE_URL is invalid")
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

	databaseURL, err := resolveDatabaseURL(useInternalDB)
	if err != nil {
		return Env{}, err
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

func resolveDatabaseURL(useInternalDB bool) (string, error) {
	if useInternalDB {
		if internalDatabaseURL := os.Getenv("INTERNAL_DATABASE_URL"); internalDatabaseURL != "" {
			return internalDatabaseURL, nil
		}

		return internalDatabaseURLDefault, nil
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return "", ErrDatabaseURLRequired
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", ErrDatabaseURLInvalid
	}

	return databaseURL, nil
}
