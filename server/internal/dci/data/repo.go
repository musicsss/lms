package data

import (
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
)

type UserRepo interface {
	Create(u *tx.Unit, user *model.User) error
	FindByUsername(db DB, username string) (*model.User, error)
	FindByID(db DB, id uint) (*model.User, error)
	List(db DB, offset, limit int, search string) ([]model.User, int64, error)
	UpdateRole(u *tx.Unit, id uint, role string) error
	Delete(u *tx.Unit, id uint) error
	Count(db DB) (int64, error)
	CountByRole(db DB, role string) (int64, error)
	UpdateProfile(u *tx.Unit, id uint, updates map[string]interface{}) error
	UpdatePassword(u *tx.Unit, id uint, hash string) error
	FindFilesByUser(db DB, userID uint, fileType string, page, pageSize int) ([]model.File, int64, error)
	FindPostsByUser(db DB, userID uint, page, pageSize int) ([]model.Post, int64, error)
	FindLikedVideos(db DB, userID uint, page, pageSize int) ([]model.File, int64, error)
}

type FileRepo interface {
	Create(u *tx.Unit, file *model.File) error
	FindByID(db DB, id uint) (*model.File, error)
	FindByParent(db DB, userID uint, parentID *uint) ([]model.File, error)
	Delete(u *tx.Unit, id uint) error
	UpdateVideoStatus(u *tx.Unit, id uint, status string) error
	FindChildren(db DB, parentID uint) ([]model.File, error)
	ListAll(db DB, offset, limit int) ([]model.File, int64, error)
	CountAll(db DB) (int64, error)
	SumSize(db DB) (int64, error)
	RandomVideos(db DB, limit int) ([]model.File, error)
}

type ForumRepo interface {
	ListBoards(db DB) ([]model.Board, error)
	FindBoardByID(db DB, id uint) (*model.Board, error)
	FindBoardBySlug(db DB, slug string) (*model.Board, error)
	CreatePost(u *tx.Unit, post *model.Post) error
	FindPostByID(db DB, id uint) (*model.Post, error)
	ListPosts(db DB, boardID uint, page, pageSize int) ([]model.Post, int64, error)
	IncrementViewCount(u *tx.Unit, postID uint) error
	FindLike(db DB, postID, userID uint) (*model.PostLike, error)
	CreateLike(u *tx.Unit, like *model.PostLike) error
	DeleteLike(u *tx.Unit, postID, userID uint) error
	CountLikes(db DB, postID uint) (int64, error)
	CountReplies(db DB, postID uint) (int64, error)
	CreateBoard(u *tx.Unit, board *model.Board) error
	UpdateBoard(u *tx.Unit, board *model.Board) error
	DeleteBoard(u *tx.Unit, id uint) error
	DeletePost(u *tx.Unit, id uint) error
	DeleteReplies(u *tx.Unit, postID uint) error
	CountPosts(db DB) (int64, error)
}

type ShareRepo interface {
	Create(u *tx.Unit, share *model.FileShare) error
	FindByToken(db DB, token string) (*model.FileShare, error)
}

type VideoSocialRepo interface {
	CreateComment(db DB, comment *model.VideoComment) error
	FindCommentsByVideo(db DB, videoID uint) ([]model.VideoComment, error)
	FindVideoLike(db DB, videoID, userID uint) (*model.VideoLike, error)
	CreateVideoLike(db DB, like *model.VideoLike) error
	DeleteVideoLike(db DB, videoID, userID uint) error
	CountVideoLikes(db DB, videoID uint) (int64, error)
	CountVideoComments(db DB, videoID uint) (int64, error)
	IncrementCommentCount(db DB, videoID uint) error
	DecrementCommentCount(db DB, videoID uint) error
	IncrementLikeCount(db DB, videoID uint) error
	DecrementLikeCount(db DB, videoID uint) error
}
type DanmakuRepo interface {
	CreateDanmaku(db DB, d *model.Danmaku) error
	FindDanmakuByVideo(db DB, videoID uint, status string) ([]model.Danmaku, error)
	FindDanmakuByID(db DB, id uint) (*model.Danmaku, error)
	ListDanmakuAdmin(db DB, page, pageSize int) ([]model.Danmaku, int64, error)
	UpdateDanmakuStatus(db DB, id uint, status string) error
	DeleteDanmaku(db DB, id uint) error
}

type DB interface {
}
