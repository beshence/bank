package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	FileID    uuid.UUID `gorm:"column:file_id;type:char(36);primaryKey" json:"file_id"`
	EventID   uuid.UUID `gorm:"column:event_id;type:char(36);not null;index" json:"event_id"`
	SessionID uuid.UUID `gorm:"column:session_id;type:char(36);not null;index" json:"session_id"`
	BlobID    uuid.UUID `gorm:"column:blob_id;type:char(36);not null;index" json:"blob_id"`
	CreatedAt time.Time `json:"created_at"`

	Session Session `gorm:"foreignKey:SessionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (f *File) BeforeCreate(_ *gorm.DB) error {
	if f.FileID == uuid.Nil {
		f.FileID = uuid.New()
	}

	return nil
}
