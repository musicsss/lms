package model

import (
	"time"
)

type File struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	ParentID    *uint     `gorm:"index" json:"parent_id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	IsDir       bool      `gorm:"not null;default:false" json:"is_dir"`
	Size        int64     `gorm:"default:0" json:"size"`
	MimeType    string    `gorm:"size:255" json:"mime_type"`
	StorageKey  string    `gorm:"size:512" json:"storage_key"`
	MD5         string    `gorm:"size:32" json:"md5"`
	IsVideo     bool      `gorm:"default:false" json:"is_video"`
	VideoStatus string    `gorm:"size:16;default:none" json:"video_status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}
