package app

import (
	"bank/internal/auth"

	"gorm.io/gorm"
)

type Dependencies struct {
	DB                *gorm.DB
	AccessJWTManager  *auth.JWTManager
	RefreshJWTManager *auth.JWTManager
}

func NewDependencies(db *gorm.DB, accessJWTManager *auth.JWTManager, refreshJWTManager *auth.JWTManager) *Dependencies {
	return &Dependencies{
		DB:                db,
		AccessJWTManager:  accessJWTManager,
		RefreshJWTManager: refreshJWTManager,
	}
}
