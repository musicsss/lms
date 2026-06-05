package data

import (
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type fileRepoImpl struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) FileRepo {
	return &fileRepoImpl{db: db}
}

func (r *fileRepoImpl) dbFrom(d DB) *gorm.DB {
	if u, ok := d.(*tx.Unit); ok {
		return u.DB()
	}
	return d.(*gorm.DB)
}

func (r *fileRepoImpl) Create(u *tx.Unit, file *model.File) error {
	return u.DB().Create(file).Error
}

func (r *fileRepoImpl) FindByID(d DB, id uint) (*model.File, error) {
	var file model.File
	err := r.dbFrom(d).Preload("User").First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *fileRepoImpl) FindByParent(d DB, userID uint, parentID *uint) ([]model.File, error) {
	db := r.dbFrom(d)
	var files []model.File
	query := db.Where("user_id = ?", userID)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	err := query.Order("is_dir DESC, name ASC").Find(&files).Error
	return files, err
}

func (r *fileRepoImpl) Delete(u *tx.Unit, id uint) error {
	return u.DB().Delete(&model.File{}, id).Error
}

func (r *fileRepoImpl) UpdateVideoStatus(u *tx.Unit, id uint, status string) error {
	return u.DB().Model(&model.File{}).Where("id = ?", id).Update("video_status", status).Error
}

func (r *fileRepoImpl) FindChildren(d DB, parentID uint) ([]model.File, error) {
	var files []model.File
	err := r.dbFrom(d).Where("parent_id = ?", parentID).Find(&files).Error
	return files, err
}

func (r *fileRepoImpl) ListAll(d DB, offset, limit int) ([]model.File, int64, error) {
	db := r.dbFrom(d)
	var files []model.File
	var total int64
	if err := db.Model(&model.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Preload("User").Order("id DESC").Offset(offset).Limit(limit).Find(&files).Error
	return files, total, err
}

func (r *fileRepoImpl) CountAll(d DB) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.File{}).Where("is_dir = false").Count(&count).Error
	return count, err
}

func (r *fileRepoImpl) SumSize(d DB) (int64, error) {
	var sum int64
	row := r.dbFrom(d).Model(&model.File{}).Select("COALESCE(SUM(size), 0)").Row()
	if err := row.Scan(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

func (r *fileRepoImpl) RandomVideos(d DB, limit int) ([]model.File, error) {
	var files []model.File
	err := r.dbFrom(d).
		Where("is_video = true AND is_dir = false").
		Preload("User").
		Order("RANDOM()").
		Limit(limit).
		Find(&files).Error
	return files, err
}
