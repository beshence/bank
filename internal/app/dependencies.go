package app

import (
	"gorm.io/gorm"

	"vault/internal/security"
)

type Dependencies struct {
	DB                *gorm.DB
	AccessJWTManager  *security.JWTManager
	RefreshJWTManager *security.JWTManager
}

func NewDependencies(db *gorm.DB, accessJWTManager *security.JWTManager, refreshJWTManager *security.JWTManager) *Dependencies {
	return &Dependencies{
		DB:                db,
		AccessJWTManager:  accessJWTManager,
		RefreshJWTManager: refreshJWTManager,
	}
}
