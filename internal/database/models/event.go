package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	EventID      uuid.UUID  `gorm:"column:event_id;type:char(36);primaryKey" json:"event_id"`
	RepositoryID uuid.UUID  `gorm:"column:repository_id;type:char(36);not null;index:idx_events_repository_parent,priority:1" json:"repository_id"`
	ParentID     *uuid.UUID `gorm:"column:parent_id;type:char(36);index:idx_events_repository_parent,priority:2" json:"parent_id,omitempty"`
	SessionID    uuid.UUID  `gorm:"column:session_id;type:char(36);not null;index" json:"session_id"`
	Payload      string     `gorm:"type:text;not null" json:"payload"`
	CreatedAt    time.Time  `json:"created_at"`

	Parent     *Event     `gorm:"foreignKey:ParentID;references:EventID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	Repository Repository `gorm:"foreignKey:RepositoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Session    Session    `gorm:"foreignKey:SessionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (e *Event) BeforeCreate(_ *gorm.DB) error {
	if e.EventID == uuid.Nil {
		e.EventID = uuid.New()
	}

	return nil
}
