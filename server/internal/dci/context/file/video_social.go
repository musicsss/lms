package file

import (
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type CommentsContext struct {
	db        *gorm.DB
	videoRepo data.VideoSocialRepo
	VideoID   uint
	result []model.VideoComment
}

func NewCommentsContext(db *gorm.DB, videoRepo data.VideoSocialRepo, videoID uint) *CommentsContext {
	return &CommentsContext{db: db, videoRepo: videoRepo, VideoID: videoID}
}

func (c *CommentsContext) Execute() ([]model.VideoComment, error) {
	comments, err := c.videoRepo.FindCommentsByVideo(c.db, c.VideoID)
	if err != nil { return nil, err }
	c.result = comments
	return comments, nil
}

type CreateCommentContext struct {
	db        *gorm.DB
	videoRepo data.VideoSocialRepo
	VideoID   uint
	UserID    uint
	ParentID  *uint
	Content   string
	result *model.VideoComment
}

func NewCreateCommentContext(db *gorm.DB, videoRepo data.VideoSocialRepo, videoID, userID uint, parentID *uint, content string) *CreateCommentContext {
	return &CreateCommentContext{db: db, videoRepo: videoRepo, VideoID: videoID, UserID: userID, ParentID: parentID, Content: content}
}

func (c *CreateCommentContext) Execute() (*model.VideoComment, error) {
	comment := &model.VideoComment{VideoID: c.VideoID, UserID: c.UserID, ParentID: c.ParentID, Content: c.Content}
	if err := c.videoRepo.CreateComment(c.db, comment); err != nil { return nil, err }
	if err := c.videoRepo.IncrementCommentCount(c.db, c.VideoID); err != nil { return nil, err }
	c.result = comment
	return comment, nil
}

type ToggleVideoLikeContext struct {
	db        *gorm.DB
	videoRepo data.VideoSocialRepo
	VideoID   uint
	UserID    uint
	result bool
}

func NewToggleVideoLikeContext(db *gorm.DB, videoRepo data.VideoSocialRepo, videoID, userID uint) *ToggleVideoLikeContext {
	return &ToggleVideoLikeContext{db: db, videoRepo: videoRepo, VideoID: videoID, UserID: userID}
}

func (c *ToggleVideoLikeContext) Execute() (bool, error) {
	existing, err := c.videoRepo.FindVideoLike(c.db, c.VideoID, c.UserID)
	if err == nil && existing != nil {
		if err := c.videoRepo.DeleteVideoLike(c.db, c.VideoID, c.UserID); err != nil { return false, err }
		if err := c.videoRepo.DecrementLikeCount(c.db, c.VideoID); err != nil { return false, err }
		c.result = false
		return false, nil
	}
	like := &model.VideoLike{VideoID: c.VideoID, UserID: c.UserID}
	if err := c.videoRepo.CreateVideoLike(c.db, like); err != nil { return false, err }
	if err := c.videoRepo.IncrementLikeCount(c.db, c.VideoID); err != nil { return false, err }
	c.result = true
	return true, nil
}

type GetLikeStatusContext struct {
	db        *gorm.DB
	videoRepo data.VideoSocialRepo
	VideoID   uint
	UserID    uint
	result bool
}

func NewGetLikeStatusContext(db *gorm.DB, videoRepo data.VideoSocialRepo, videoID, userID uint) *GetLikeStatusContext {
	return &GetLikeStatusContext{db: db, videoRepo: videoRepo, VideoID: videoID, UserID: userID}
}

func (c *GetLikeStatusContext) Execute() (bool, error) {
	like, err := c.videoRepo.FindVideoLike(c.db, c.VideoID, c.UserID)
	if err != nil { return false, nil }
	c.result = true
	return like != nil, nil
}
