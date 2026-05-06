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
		&models.Account{},
		&models.Session{},
		&models.Vault{},
		&models.Chain{},
		&models.Event{},
		&models.File{},
	); err != nil {
		return err
	}

	queries := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_chain_root_unique ON events (chain_id) WHERE parent_id IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_chain_parent_unique ON events (chain_id, parent_id) WHERE parent_id IS NOT NULL`,
	}

	for _, query := range queries {
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("create event constraints: %w", err)
		}
	}

	return nil
}
