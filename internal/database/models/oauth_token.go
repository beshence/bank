package models

import (
	"time"

	"github.com/google/uuid"
)

type OauthToken struct {
	ID        uuid.UUID `gorm:"column:token;type:char(36);primaryKey;not null" json:"token"`
	AccountID uuid.UUID `gorm:"column:account_id;type:char(36);primaryKey;not null" json:"account_id"`
	VaultID   uuid.UUID `gorm:"column:vault_id;type:char(36);primaryKey;not null" json:"vault_id"`
	Scope     string    `gorm:"column:scope;type:varchar;not null" json:"scope"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`

	Account Account `gorm:"foreignKey:AccountID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Vault   Vault   `gorm:"foreignKey:VaultID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (s *OauthToken) TableName() string {
	return "oauth_tokens"
}
