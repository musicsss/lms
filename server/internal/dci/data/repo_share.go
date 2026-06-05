package data

import (
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type shareRepoImpl struct {
	db *gorm.DB
}

func NewShareRepo(db *gorm.DB) ShareRepo {
	return &shareRepoImpl{db: db}
}

func (r *shareRepoImpl) dbFrom(d DB) *gorm.DB {
	if u, ok := d.(*tx.Unit); ok {
		return u.DB()
	}
	return d.(*gorm.DB)
}

func (r *shareRepoImpl) Create(u *tx.Unit, share *model.FileShare) error {
	return u.DB().Create(share).Error
}

func (r *shareRepoImpl) FindByToken(d DB, token string) (*model.FileShare, error) {
	var share model.FileShare
	err := r.dbFrom(d).Preload("File").Where("token = ?", token).First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}
