package model

import (
	"time"
)

type VideoTranscode struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	FileID     uint      `gorm:"index;not null" json:"file_id"`
	Resolution string    `gorm:"size:16;not null" json:"resolution"`
	HLSPath    string    `gorm:"size:512;not null" json:"hls_path"`
	Status     string    `gorm:"size:16;default:pending" json:"status"`
	CreatedAt  time.Time `json:"created_at"`

	File File `gorm:"foreignKey:FileID" json:"-"`
}
