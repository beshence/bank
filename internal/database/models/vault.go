package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Vault struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null;uniqueIndex:idx_account_name" json:"name"`
	AccountID uuid.UUID `gorm:"column:account_id;type:char(36);not null;index;uniqueIndex:idx_account_name" json:"account_id"`
	CreatedAt time.Time `json:"created_at"`

	Account Account `gorm:"foreignKey:AccountID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (v *Vault) BeforeCreate(_ *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}

	return nil
}
