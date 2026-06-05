// Package file provides file management DCI contexts including random video listing.
package file

import (
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

// RandomVideosContext returns random video files for the homepage.
type RandomVideosContext struct {
	db   *gorm.DB
	repo data.FileRepo

	Limit int

	result []model.File
}

func NewRandomVideosContext(db *gorm.DB, repo data.FileRepo, limit int) *RandomVideosContext {
	return &RandomVideosContext{db: db, repo: repo, Limit: limit}
}

func (c *RandomVideosContext) Execute() ([]model.File, error) {
	files, err := c.repo.RandomVideos(c.db, c.Limit)
	if err != nil {
		return nil, err
	}
	c.result = files
	return files, nil
}
