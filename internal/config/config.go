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
	ErrDatabaseURLRequired   = errors.New("DATABASE_URL is required")
	ErrDatabaseURLInvalid    = errors.New("DATABASE_URL is invalid")
	ErrJWTSecretRequired     = errors.New("JWT_SECRET is required")
	ErrRefreshJWTTTLRequired = errors.New("REFRESH_JWT_TTL_SECONDS is required")
	ErrAccessJWTTTLRequired  = errors.New("ACCESS_JWT_TTL_SECONDS is required")
	ErrRefreshJWTTTLInvalid  = errors.New("REFRESH_JWT_TTL_SECONDS must be a positive integer")
	ErrAccessJWTTTLInvalid   = errors.New("ACCESS_JWT_TTL_SECONDS must be a positive integer")
)

type Config struct {
	DatabaseURL          string
	JWTSecret            string
	AccessJWTTTLSeconds  time.Duration
	RefreshJWTTTLSeconds time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()

	databaseUrl, err := resolveDatabaseURL()
	if err != nil {
		return Config{}, err
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, ErrJWTSecretRequired
	}
	if jwtSecret == "change_me_to_a_long_random_secret" {
		return Config{}, ErrJWTSecretRequired
	}

	accessJWTTTLRaw := os.Getenv("ACCESS_JWT_TTL_SECONDS")
	if accessJWTTTLRaw == "" {
		return Config{}, ErrAccessJWTTTLRequired
	}

	accessJWTTTLSeconds, err := strconv.Atoi(accessJWTTTLRaw)
	if err != nil || accessJWTTTLSeconds <= 0 {
		return Config{}, ErrAccessJWTTTLInvalid
	}

	refreshJWTTTLRaw := os.Getenv("REFRESH_JWT_TTL_SECONDS")
	if refreshJWTTTLRaw == "" {
		return Config{}, ErrRefreshJWTTTLRequired
	}

	refreshJWTTTLSeconds, err := strconv.Atoi(refreshJWTTTLRaw)
	if err != nil || refreshJWTTTLSeconds <= 0 {
		return Config{}, ErrRefreshJWTTTLInvalid
	}

	return Config{
		DatabaseURL:          databaseUrl,
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
