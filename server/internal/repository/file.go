package repository

import (
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type FileRepo struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

func (r *FileRepo) Create(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *FileRepo) FindByID(id uint) (*model.File, error) {
	var file model.File
	err := r.db.First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *FileRepo) FindByParent(userID uint, parentID *uint) ([]model.File, error) {
	var files []model.File
	query := r.db.Where("user_id = ?", userID)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	err := query.Order("is_dir DESC, name ASC").Find(&files).Error
	return files, err
}

func (r *FileRepo) Delete(id uint) error {
	return r.db.Delete(&model.File{}, id).Error
}

func (r *FileRepo) UpdateVideoStatus(id uint, status string) error {
	return r.db.Model(&model.File{}).Where("id = ?", id).Update("video_status", status).Error
}

func (r *FileRepo) FindChildren(parentID uint) ([]model.File, error) {
	var files []model.File
	err := r.db.Where("parent_id = ?", parentID).Find(&files).Error
	return files, err
}
