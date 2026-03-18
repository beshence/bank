package models

import (
	"errors"
	"regexp"
	"time"

	"gorm.io/gorm"
)

var repositoryNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var (
	ErrInvalidRepositoryName   = errors.New("repository name must contain only English letters, numbers, '_' or '-' characters")
	ErrRepositoryOwnerRequired = errors.New("repository owner is required")
)

type Repository struct {
	Name      string    `gorm:"size:128;not null;uniqueIndex:idx_owner_name" json:"name"`
	Owner     string    `gorm:"type:char(36);not null;index;uniqueIndex:idx_owner_name" json:"owner"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:Owner;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (r *Repository) BeforeCreate(_ *gorm.DB) error {
	return r.Validate()
}

func (r *Repository) Validate() error {
	if !repositoryNamePattern.MatchString(r.Name) {
		return ErrInvalidRepositoryName
	}

	if r.Owner == "" {
		return ErrRepositoryOwnerRequired
	}

	return nil
}
