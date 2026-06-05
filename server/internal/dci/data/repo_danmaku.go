package data

import (
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"

	"gorm.io/gorm"
)

type danmakuRepoImpl struct {
	db *gorm.DB
}

func NewDanmakuRepo(db *gorm.DB) DanmakuRepo {
	return &danmakuRepoImpl{db: db}
}

func (r *danmakuRepoImpl) dbFrom(d DB) *gorm.DB {
	if u, ok := d.(*tx.Unit); ok {
		return u.DB()
	}
	return d.(*gorm.DB)
}

func (r *danmakuRepoImpl) CreateDanmaku(d DB, dm *model.Danmaku) error {
	return r.dbFrom(d).Create(dm).Error
}

func (r *danmakuRepoImpl) FindDanmakuByVideo(d DB, videoID uint, status string) ([]model.Danmaku, error) {
	var danmaku []model.Danmaku
	err := r.dbFrom(d).
		Where("video_id = ? AND status = ?", videoID, status).
		Preload("User").
		Order("time_sec ASC").
		Find(&danmaku).Error
	return danmaku, err
}

func (r *danmakuRepoImpl) FindDanmakuByID(d DB, id uint) (*model.Danmaku, error) {
	var dm model.Danmaku
	err := r.dbFrom(d).Preload("User").First(&dm, id).Error
	if err != nil {
		return nil, err
	}
	return &dm, nil
}

func (r *danmakuRepoImpl) ListDanmakuAdmin(d DB, page, pageSize int) ([]model.Danmaku, int64, error) {
	var danmaku []model.Danmaku
	var total int64
	offset := (page - 1) * pageSize
	if err := r.dbFrom(d).Model(&model.Danmaku{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.dbFrom(d).
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&danmaku).Error
	return danmaku, total, err
}

func (r *danmakuRepoImpl) UpdateDanmakuStatus(d DB, id uint, status string) error {
	return r.dbFrom(d).Model(&model.Danmaku{}).Where("id = ?", id).Update("status", status).Error
}

func (r *danmakuRepoImpl) DeleteDanmaku(d DB, id uint) error {
	return r.dbFrom(d).Delete(&model.Danmaku{}, id).Error
}
