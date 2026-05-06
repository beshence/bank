package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Session struct {
	ID             string    `gorm:"type:char(36);primaryKey" json:"id"`
	AccountID      string    `gorm:"column:account_id;type:char(36);not null;index" json:"account_id"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	RefreshTokenID string    `gorm:"column:refresh_token_id;type:char(36);not null" json:"refresh_token_id"`
	CreatedAt      time.Time `json:"created_at"`

	Account Account `gorm:"foreignKey:AccountID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (s *Session) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}

	return nil
}
