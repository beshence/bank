package app

import (
	"gorm.io/gorm"

	"vault/internal/security"
)

type Dependencies struct {
	DB         *gorm.DB
	JWTManager *security.JWTManager
}

func NewDependencies(db *gorm.DB, jwtManager *security.JWTManager) *Dependencies {
	return &Dependencies{
		DB:         db,
		JWTManager: jwtManager,
	}
}
