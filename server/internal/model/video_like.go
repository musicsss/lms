package model

import "time"

type VideoLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"uniqueIndex:idx_video_user;not null" json:"video_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_video_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
