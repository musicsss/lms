package data

import (
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type forumRepoImpl struct {
	db *gorm.DB
}

func NewForumRepo(db *gorm.DB) ForumRepo {
	return &forumRepoImpl{db: db}
}

func (r *forumRepoImpl) dbFrom(d DB) *gorm.DB {
	if u, ok := d.(*tx.Unit); ok {
		return u.DB()
	}
	return d.(*gorm.DB)
}

func (r *forumRepoImpl) ListBoards(d DB) ([]model.Board, error) {
	var boards []model.Board
	err := r.dbFrom(d).Order("sort_order ASC").Find(&boards).Error
	return boards, err
}

func (r *forumRepoImpl) FindBoardByID(d DB, id uint) (*model.Board, error) {
	var board model.Board
	err := r.dbFrom(d).First(&board, id).Error
	return &board, err
}

func (r *forumRepoImpl) FindBoardBySlug(d DB, slug string) (*model.Board, error) {
	var board model.Board
	err := r.dbFrom(d).Where("slug = ?", slug).First(&board).Error
	return &board, err
}

func (r *forumRepoImpl) CreatePost(u *tx.Unit, post *model.Post) error {
	return u.DB().Create(post).Error
}

func (r *forumRepoImpl) FindPostByID(d DB, id uint) (*model.Post, error) {
	var post model.Post
	err := r.dbFrom(d).Preload("User").Preload("Replies.User").First(&post, id).Error
	return &post, err
}

func (r *forumRepoImpl) ListPosts(d DB, boardID uint, page, pageSize int) ([]model.Post, int64, error) {
	db := r.dbFrom(d)
	var posts []model.Post
	var total int64

	query := db.Model(&model.Post{}).Where("board_id = ? AND parent_id IS NULL", boardID)
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

func (r *forumRepoImpl) IncrementViewCount(u *tx.Unit, postID uint) error {
	return u.DB().Model(&model.Post{}).Where("id = ?", postID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *forumRepoImpl) FindLike(d DB, postID, userID uint) (*model.PostLike, error) {
	var like model.PostLike
	err := r.dbFrom(d).Where("post_id = ? AND user_id = ?", postID, userID).First(&like).Error
	return &like, err
}

func (r *forumRepoImpl) CreateLike(u *tx.Unit, like *model.PostLike) error {
	return u.DB().Create(like).Error
}

func (r *forumRepoImpl) DeleteLike(u *tx.Unit, postID, userID uint) error {
	return u.DB().Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostLike{}).Error
}

func (r *forumRepoImpl) CountLikes(d DB, postID uint) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.PostLike{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *forumRepoImpl) CountReplies(d DB, postID uint) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.Post{}).Where("parent_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *forumRepoImpl) CreateBoard(u *tx.Unit, board *model.Board) error {
	return u.DB().Create(board).Error
}

func (r *forumRepoImpl) UpdateBoard(u *tx.Unit, board *model.Board) error {
	return u.DB().Save(board).Error
}

func (r *forumRepoImpl) DeleteBoard(u *tx.Unit, id uint) error {
	return u.DB().Delete(&model.Board{}, id).Error
}

func (r *forumRepoImpl) DeletePost(u *tx.Unit, id uint) error {
	return u.DB().Delete(&model.Post{}, id).Error
}

func (r *forumRepoImpl) DeleteReplies(u *tx.Unit, postID uint) error {
	return u.DB().Where("parent_id = ?", postID).Delete(&model.Post{}).Error
}

func (r *forumRepoImpl) CountPosts(d DB) (int64, error) {
	var count int64
	err := r.dbFrom(d).Model(&model.Post{}).Count(&count).Error
	return count, err
}
