package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"vault/internal/database/models"
)

func New(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		TranslateError: true,
	})
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{})
}
