package models

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var repositoryNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var (
	ErrInvalidRepositoryName   = errors.New("repository name must contain only English letters, numbers, '_' or '-' characters")
	ErrRepositoryOwnerRequired = errors.New("repository owner is required")
)

type Repository struct {
	ID          uuid.UUID  `gorm:"type:char(36);primaryKey" json:"id"`
	Name        string     `gorm:"size:128;not null;uniqueIndex:idx_owner_name" json:"name"`
	OwnerID     uuid.UUID  `gorm:"column:owner_id;type:char(36);not null;index;uniqueIndex:idx_owner_name" json:"owner_id"`
	LastEventID *uuid.UUID `gorm:"column:last_event_id;type:char(36);index" json:"last_event_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	LastEvent *Event `gorm:"foreignKey:LastEventID;references:EventID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	Owner     User   `gorm:"foreignKey:OwnerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (r *Repository) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}

	return r.Validate()
}

func (r *Repository) Validate() error {
	if !repositoryNamePattern.MatchString(r.Name) {
		return ErrInvalidRepositoryName
	}

	if r.OwnerID == uuid.Nil {
		return ErrRepositoryOwnerRequired
	}

	return nil
}
