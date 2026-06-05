package data

import (
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type userRepoImpl struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepoImpl{db: db}
}

func (r *userRepoImpl) dbFrom(d DB) *gorm.DB {
	if u, ok := d.(*tx.Unit); ok {
		return u.DB()
	}
	return d.(*gorm.DB)
}

func (r *userRepoImpl) Create(u *tx.Unit, user *model.User) error {
	return u.DB().Create(user).Error
}

func (r *userRepoImpl) FindByUsername(d DB, username string) (*model.User, error) {
	var user model.User
	err := r.dbFrom(d).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepoImpl) FindByID(d DB, id uint) (*model.User, error) {
	var user model.User
	err := r.dbFrom(d).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepoImpl) List(d DB, offset, limit int, search string) ([]model.User, int64, error) {
	db := r.dbFrom(d)
	var users []model.User
	var total int64
	query := db.Model(&model.User{})
	if search != "" {
		query = query.Where("username LIKE ?", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id ASC").Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

func (r *userRepoImpl) UpdateRole(u *tx.Unit, id uint, role string) error {
	return u.DB().Model(&model.User{}).Where("id = ?", id).Update("role", role).Error
}

func (r *userRepoImpl) Delete(u *tx.Unit, id uint) error {
	return u.DB().Delete(&model.User{}, id).Error
}

func (r *userRepoImpl) Count(d DB) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.User{}).Count(&count).Error
	return count, err
}

func (r *userRepoImpl) CountByRole(d DB, role string) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.User{}).Where("role = ?", role).Count(&count).Error
	return count, err
}

func (r *userRepoImpl) UpdateProfile(u *tx.Unit, id uint, updates map[string]interface{}) error {
	return u.DB().Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *userRepoImpl) UpdatePassword(u *tx.Unit, id uint, hash string) error {
	return u.DB().Model(&model.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}

func (r *userRepoImpl) FindFilesByUser(d DB, userID uint, fileType string, page, pageSize int) ([]model.File, int64, error) {
	db := r.dbFrom(d)
	var files []model.File
	var total int64

	query := db.Model(&model.File{}).Where("user_id = ?", userID)
	switch fileType {
	case "video":
		query = query.Where("is_video = true AND is_dir = false")
	default:
		// "all" or empty — no additional filter
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files).Error
	return files, total, err
}

func (r *userRepoImpl) FindPostsByUser(d DB, userID uint, page, pageSize int) ([]model.Post, int64, error) {
	db := r.dbFrom(d)
	var posts []model.Post
	var total int64

	query := db.Model(&model.Post{}).Where("user_id = ? AND parent_id IS NULL", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error
	return posts, total, err
}

func (r *userRepoImpl) FindLikedVideos(d DB, userID uint, page, pageSize int) ([]model.File, int64, error) {
	db := r.dbFrom(d)
	var files []model.File
	var total int64

	query := db.Model(&model.File{}).
		Joins("JOIN video_likes ON video_likes.video_id = files.id").
		Where("video_likes.user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Preload("User").
		Order("files.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files).Error
	return files, total, err
}
