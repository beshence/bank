package auth

import (
	"vault/internal/app"
	"vault/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type tokenPair struct {
	AccessToken      string
	AccessExpiresAt  int64
	RefreshToken     string
	RefreshExpiresAt int64
}

func issueTokenPairForNewSession(deps *app.Dependencies, user models.User, sessionName string) (tokenPair, error) {
	if sessionName == "" {
		sessionName = "unknown"
	}

	accessTokenID := uuid.NewString()
	session := models.Session{
		AccountID:     user.ID,
		Name:          sessionName,
		AccessTokenID: accessTokenID,
	}

	var tokens tokenPair
	err := deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&session).Error; err != nil {
			return err
		}

		pair, err := generateTokenPairForSession(deps, user, session.ID, accessTokenID)
		if err != nil {
			return err
		}

		tokens = pair
		return nil
	})
	if err != nil {
		return tokenPair{}, err
	}

	return tokens, nil
}

func issueTokenPairForExistingSession(deps *app.Dependencies, user models.User, session models.Session) (tokenPair, error) {
	accessTokenID := uuid.NewString()

	var tokens tokenPair
	err := deps.DB.Transaction(func(tx *gorm.DB) error {
		pair, err := generateTokenPairForSession(deps, user, session.ID, accessTokenID)
		if err != nil {
			return err
		}

		updateResult := tx.Model(&models.Session{}).
			Where("id = ? AND account_id = ?", session.ID, user.ID).
			Update("access_token_id", accessTokenID)
		if updateResult.Error != nil {
			return updateResult.Error
		}

		if updateResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		tokens = pair
		return nil
	})
	if err != nil {
		return tokenPair{}, err
	}

	return tokens, nil
}

func generateTokenPairForSession(deps *app.Dependencies, user models.User, sessionID string, accessTokenID string) (tokenPair, error) {
	accessToken, accessExpiresAt, err := deps.AccessJWTManager.GenerateToken(sessionID, user.ID, accessTokenID)
	if err != nil {
		return tokenPair{}, err
	}

	refreshToken, refreshExpiresAt, err := deps.RefreshJWTManager.GenerateToken(sessionID, user.ID, accessTokenID)
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
