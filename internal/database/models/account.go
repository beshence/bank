package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	ID           string    `gorm:"type:char(36);primaryKey" json:"id"`
	Login        string    `gorm:"size:64;not null;uniqueIndex" json:"login"`
	PasswordHash string    `gorm:"column:password_hash;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (a *Account) TableName() string {
	return "accounts"
}

func (a *Account) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}

	return nil
}
