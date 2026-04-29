package database

import (
	"fmt"

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
	if err := db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.Repository{},
		&models.Event{},
		&models.File{},
	); err != nil {
		return err
	}

	queries := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_repository_root_unique ON events (repository_id) WHERE parent_id IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_repository_parent_unique ON events (repository_id, parent_id) WHERE parent_id IS NOT NULL`,
	}

	for _, query := range queries {
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("create event constraints: %w", err)
		}
	}

	return nil
}
