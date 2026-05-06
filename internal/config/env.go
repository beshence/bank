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
	ErrDatabaseURLRequired  = errors.New("DATABASE_URL is required")
	ErrDatabaseURLInvalid   = errors.New("DATABASE_URL is invalid")
	ErrJWTSecretRequired    = errors.New("JWT_SECRET is required")
	ErrInvalidAccessJWTTTL  = errors.New("ACCESS_JWT_TTL_SECONDS must be a positive integer")
	ErrInvalidRefreshJWTTTL = errors.New("REFRESH_JWT_TTL_SECONDS must be a positive integer")
)

type Env struct {
	DatabaseURL          string
	JWTSecret            string
	AccessJWTTTLSeconds  time.Duration
	RefreshJWTTTLSeconds time.Duration
}

func Load() (Env, error) {
	_ = godotenv.Load()

	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return Env{}, err
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Env{}, ErrJWTSecretRequired
	}

	accessJWTTTLRaw := os.Getenv("ACCESS_JWT_TTL_SECONDS")
	if accessJWTTTLRaw == "" {
		accessJWTTTLRaw = os.Getenv("JWT_TTL_SECONDS")
	}
	if accessJWTTTLRaw == "" {
		accessJWTTTLRaw = "3600"
	}

	accessJWTTTLSeconds, err := strconv.Atoi(accessJWTTTLRaw)
	if err != nil || accessJWTTTLSeconds <= 0 {
		return Env{}, ErrInvalidAccessJWTTTL
	}

	refreshJWTTTLRaw := os.Getenv("REFRESH_JWT_TTL_SECONDS")
	if refreshJWTTTLRaw == "" {
		refreshJWTTTLRaw = "31536000"
	}

	refreshJWTTTLSeconds, err := strconv.Atoi(refreshJWTTTLRaw)
	if err != nil || refreshJWTTTLSeconds <= 0 {
		return Env{}, ErrInvalidRefreshJWTTTL
	}

	return Env{
		DatabaseURL:          databaseURL,
		JWTSecret:            jwtSecret,
		AccessJWTTTLSeconds:  time.Duration(accessJWTTTLSeconds) * time.Second,
		RefreshJWTTTLSeconds: time.Duration(refreshJWTTTLSeconds) * time.Second,
	}, nil
}

func resolveDatabaseURL() (string, error) {
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
