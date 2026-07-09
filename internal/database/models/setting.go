package models

type Setting struct {
	Key   string `gorm:"primaryKey;size:255;not null"`
	Value string `gorm:"type:text;not null"`
}

func (s *Setting) TableName() string {
	return "settings"
}
