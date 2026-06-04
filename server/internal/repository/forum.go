package repository

import (
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type ForumRepo struct {
	db *gorm.DB
}

func NewForumRepo(db *gorm.DB) *ForumRepo {
	return &ForumRepo{db: db}
}

func (r *ForumRepo) ListBoards() ([]model.Board, error) {
	var boards []model.Board
	err := r.db.Order("sort_order ASC").Find(&boards).Error
	return boards, err
}

func (r *ForumRepo) FindBoardByID(id uint) (*model.Board, error) {
	var board model.Board
	err := r.db.First(&board, id).Error
	return &board, err
}

func (r *ForumRepo) FindBoardBySlug(slug string) (*model.Board, error) {
	var board model.Board
	err := r.db.Where("slug = ?", slug).First(&board).Error
	return &board, err
}

func (r *ForumRepo) CreatePost(post *model.Post) error {
	return r.db.Create(post).Error
}

func (r *ForumRepo) FindPostByID(id uint) (*model.Post, error) {
	var post model.Post
	err := r.db.Preload("User").Preload("Replies.User").First(&post, id).Error
	return &post, err
}

func (r *ForumRepo) ListPosts(boardID uint, page, pageSize int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := r.db.Model(&model.Post{}).Where("board_id = ? AND parent_id IS NULL", boardID)
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

func (r *ForumRepo) IncrementViewCount(postID uint) error {
	return r.db.Model(&model.Post{}).Where("id = ?", postID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *ForumRepo) FindLike(postID, userID uint) (*model.PostLike, error) {
	var like model.PostLike
	err := r.db.Where("post_id = ? AND user_id = ?", postID, userID).First(&like).Error
	return &like, err
}

func (r *ForumRepo) CreateLike(like *model.PostLike) error {
	return r.db.Create(like).Error
}

func (r *ForumRepo) DeleteLike(postID, userID uint) error {
	return r.db.Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostLike{}).Error
}

func (r *ForumRepo) CountLikes(postID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.PostLike{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *ForumRepo) CountReplies(postID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Post{}).Where("parent_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *ForumRepo) CreateBoard(board *model.Board) error {
	return r.db.Create(board).Error
}

func (r *ForumRepo) UpdateBoard(board *model.Board) error {
	return r.db.Save(board).Error
}

func (r *ForumRepo) DeleteBoard(id uint) error {
	return r.db.Delete(&model.Board{}, id).Error
}

func (r *ForumRepo) DeletePost(id uint) error {
	return r.db.Delete(&model.Post{}, id).Error
}

func (r *ForumRepo) DeleteReplies(postID uint) error {
	return r.db.Where("parent_id = ?", postID).Delete(&model.Post{}).Error
}

func (r *ForumRepo) CountPosts() (int64, error) {
	var count int64
	err := r.db.Model(&model.Post{}).Count(&count).Error
	return count, err
}
