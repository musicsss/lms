package model

import (
	"time"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Email        string    `gorm:"size:255" json:"email"`
	Role         string    `gorm:"size:16;default:user" json:"role"`
	AvatarURL    string    `gorm:"size:512" json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
