package data

import (
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type videoSocialRepoImpl struct {
	db *gorm.DB
}

func NewVideoSocialRepo(db *gorm.DB) VideoSocialRepo {
	return &videoSocialRepoImpl{db: db}
}

func (r *videoSocialRepoImpl) dbFrom(d DB) *gorm.DB {
	if u, ok := d.(*tx.Unit); ok {
		return u.DB()
	}
	return d.(*gorm.DB)
}

func (r *videoSocialRepoImpl) CreateComment(d DB, comment *model.VideoComment) error {
	return r.dbFrom(d).Create(comment).Error
}

func (r *videoSocialRepoImpl) FindCommentsByVideo(d DB, videoID uint) ([]model.VideoComment, error) {
	var comments []model.VideoComment
	err := r.dbFrom(d).
		Where("video_id = ? AND parent_id IS NULL", videoID).
		Preload("User").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Preload("User").Order("created_at ASC")
		}).
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (r *videoSocialRepoImpl) FindVideoLike(d DB, videoID, userID uint) (*model.VideoLike, error) {
	var like model.VideoLike
	err := r.dbFrom(d).Where("video_id = ? AND user_id = ?", videoID, userID).First(&like).Error
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (r *videoSocialRepoImpl) CreateVideoLike(d DB, like *model.VideoLike) error {
	return r.dbFrom(d).Create(like).Error
}

func (r *videoSocialRepoImpl) DeleteVideoLike(d DB, videoID, userID uint) error {
	return r.dbFrom(d).Where("video_id = ? AND user_id = ?", videoID, userID).Delete(&model.VideoLike{}).Error
}

func (r *videoSocialRepoImpl) CountVideoLikes(d DB, videoID uint) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.VideoLike{}).Where("video_id = ?", videoID).Count(&count).Error
	return count, err
}

func (r *videoSocialRepoImpl) CountVideoComments(d DB, videoID uint) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.VideoComment{}).Where("video_id = ?", videoID).Count(&count).Error
	return count, err
}

func (r *videoSocialRepoImpl) IncrementCommentCount(d DB, videoID uint) error {
	return r.dbFrom(d).Model(&model.File{}).Where("id = ?", videoID).
		UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
}

func (r *videoSocialRepoImpl) DecrementCommentCount(d DB, videoID uint) error {
	return r.dbFrom(d).Model(&model.File{}).Where("id = ?", videoID).
		UpdateColumn("comment_count", gorm.Expr("comment_count - 1")).Error
}

func (r *videoSocialRepoImpl) IncrementLikeCount(d DB, videoID uint) error {
	return r.dbFrom(d).Model(&model.File{}).Where("id = ?", videoID).
		UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
}

func (r *videoSocialRepoImpl) DecrementLikeCount(d DB, videoID uint) error {
	return r.dbFrom(d).Model(&model.File{}).Where("id = ?", videoID).
		UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error
}
