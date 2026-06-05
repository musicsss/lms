package model

import "time"

type VideoComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"index;not null" json:"video_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ParentID  *uint     `gorm:"index" json:"parent_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`

	User    User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Replies []VideoComment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}
