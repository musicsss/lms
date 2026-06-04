package repository

import (
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type ShareRepo struct {
	db *gorm.DB
}

func NewShareRepo(db *gorm.DB) *ShareRepo {
	return &ShareRepo{db: db}
}

func (r *ShareRepo) Create(share *model.FileShare) error {
	return r.db.Create(share).Error
}

func (r *ShareRepo) FindByToken(token string) (*model.FileShare, error) {
	var share model.FileShare
	err := r.db.Preload("File").Where("token = ?", token).First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}
