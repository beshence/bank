package auth

import (
	"vault/internal/app"
	"vault/internal/database/models"
)

type tokenPair struct {
	AccessToken      string
	AccessExpiresAt  int64
	RefreshToken     string
	RefreshExpiresAt int64
}

func issueTokenPair(deps *app.Dependencies, user models.User) (tokenPair, error) {
	accessToken, accessExpiresAt, err := deps.AccessJWTManager.GenerateToken(user.ID, user.Login)
	if err != nil {
		return tokenPair{}, err
	}

	refreshToken, refreshExpiresAt, err := deps.RefreshJWTManager.GenerateToken(user.ID, user.Login)
	if err != nil {
		return tokenPair{}, err
	}

	return tokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}
