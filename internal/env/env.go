package env

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

type Env struct {
	DatabaseURL          string
	JWTSecret            string
	AccessJWTTTLSeconds  time.Duration
	RefreshJWTTTLSeconds time.Duration
}

func Load() (Env, error) {
	_ = godotenv.Load()

	databaseUrl, err := resolveDatabaseURL()
	if err != nil {
		return Env{}, err
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Env{}, ErrJWTSecretRequired
	}
	if jwtSecret == "change_me_to_a_long_random_secret" {
		return Env{}, ErrJWTSecretRequired
	}

	accessJWTTTLRaw := os.Getenv("ACCESS_JWT_TTL_SECONDS")
	if accessJWTTTLRaw == "" {
		return Env{}, ErrAccessJWTTTLRequired
	}

	accessJWTTTLSeconds, err := strconv.Atoi(accessJWTTTLRaw)
	if err != nil || accessJWTTTLSeconds <= 0 {
		return Env{}, ErrAccessJWTTTLInvalid
	}

	refreshJWTTTLRaw := os.Getenv("REFRESH_JWT_TTL_SECONDS")
	if refreshJWTTTLRaw == "" {
		return Env{}, ErrRefreshJWTTTLRequired
	}

	refreshJWTTTLSeconds, err := strconv.Atoi(refreshJWTTTLRaw)
	if err != nil || refreshJWTTTLSeconds <= 0 {
		return Env{}, ErrRefreshJWTTTLInvalid
	}

	return Env{
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
