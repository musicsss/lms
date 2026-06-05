package model

import "time"

type Danmaku struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"index;not null" json:"video_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Content   string    `gorm:"size:256;not null" json:"content"`
	TimeSec   float64   `gorm:"not null" json:"time_sec"`
	Color     string    `gorm:"size:7;default:#ffffff" json:"color"`
	FontSize  int       `gorm:"default:25" json:"font_size"`
	Type      int       `gorm:"default:1" json:"type"`
	Status    string    `gorm:"size:16;default:pending" json:"status"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
