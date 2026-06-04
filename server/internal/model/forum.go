package model

import (
	"time"
)

type Board struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Slug        string    `gorm:"uniqueIndex;size:128;not null" json:"slug"`
	Description string    `gorm:"size:512" json:"description"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`

	Posts []Post `gorm:"foreignKey:BoardID" json:"-"`
}

type Post struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BoardID   uint      `gorm:"index;not null" json:"board_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"size:255" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	ParentID  *uint     `gorm:"index" json:"parent_id"`
	ViewCount int       `gorm:"default:0" json:"view_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User     User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Board    Board      `gorm:"foreignKey:BoardID" json:"-"`
	Replies  []Post     `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
	Likes    []PostLike `gorm:"foreignKey:PostID" json:"-"`
	LikeCount int       `gorm:"-" json:"like_count"`
}

type PostLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PostID    uint      `gorm:"index;not null" json:"post_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
