package model

import (
	"time"
)

type FileShare struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FileID    uint      `gorm:"index;not null" json:"file_id"`
	Token     string    `gorm:"uniqueIndex;size:64;not null" json:"token"`
	Password  string    `gorm:"size:255" json:"-"`
	ExpireAt  *time.Time `json:"expire_at"`
	CreatedAt time.Time `json:"created_at"`

	File File `gorm:"foreignKey:FileID" json:"file,omitempty"`
}
