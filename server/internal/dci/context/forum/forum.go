// Package forum 提供论坛相关的 DCI 上下文。
package forum

import (
	"errors"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

// ---- CreatePostContext ----

// CreatePostContext 处理发帖的 DCI 上下文。
type CreatePostContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	BoardID uint
	UserID  uint
	Title   string
	Content string

	result *model.Post
}

func NewCreatePostContext(db *gorm.DB, forumRepo data.ForumRepo, boardID, userID uint, title, content string) *CreatePostContext {
	return &CreatePostContext{db: db, forumRepo: forumRepo, BoardID: boardID, UserID: userID, Title: title, Content: content}
}

func (c *CreatePostContext) Execute() (*model.Post, error) {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}

	post := &model.Post{
		BoardID: c.BoardID,
		UserID:  c.UserID,
		Title:   c.Title,
		Content: c.Content,
	}
	if err := c.forumRepo.CreatePost(u, post); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	c.result = post
	return post, nil
}

// ---- ListBoardsContext ----

type ListBoardsContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	result []model.Board
}

func NewListBoardsContext(db *gorm.DB, forumRepo data.ForumRepo) *ListBoardsContext {
	return &ListBoardsContext{db: db, forumRepo: forumRepo}
}

func (c *ListBoardsContext) Execute() ([]model.Board, error) {
	boards, err := c.forumRepo.ListBoards(c.db)
	if err != nil {
		return nil, err
	}
	c.result = boards
	return boards, nil
}

// ---- ListPostsContext ----

type ListPostsContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	BoardID  uint
	Page     int
	PageSize int

	posts []model.Post
	total int64
}

func NewListPostsContext(db *gorm.DB, forumRepo data.ForumRepo, boardID uint, page, pageSize int) *ListPostsContext {
	return &ListPostsContext{db: db, forumRepo: forumRepo, BoardID: boardID, Page: page, PageSize: pageSize}
}

func (c *ListPostsContext) Execute() ([]model.Post, int64, error) {
	posts, total, err := c.forumRepo.ListPosts(c.db, c.BoardID, c.Page, c.PageSize)
	if err != nil {
		return nil, 0, err
	}

	// 填充 like_count
	for i := range posts {
		count, _ := c.forumRepo.CountLikes(c.db, posts[i].ID)
		posts[i].LikeCount = int(count)
	}

	c.posts = posts
	c.total = total
	return posts, total, nil
}

// ---- GetPostContext ----

type GetPostContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	PostID uint

	result *model.Post
}

func NewGetPostContext(db *gorm.DB, forumRepo data.ForumRepo, postID uint) *GetPostContext {
	return &GetPostContext{db: db, forumRepo: forumRepo, PostID: postID}
}

func (c *GetPostContext) Execute() (*model.Post, error) {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}

	if err := c.forumRepo.IncrementViewCount(u, c.PostID); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	post, err := c.forumRepo.FindPostByID(c.db, c.PostID)
	if err != nil {
		return nil, err
	}

	// 填充 like_count
	for i := range post.Replies {
		count, _ := c.forumRepo.CountLikes(c.db, post.Replies[i].ID)
		post.Replies[i].LikeCount = int(count)
	}
	likeCount, _ := c.forumRepo.CountLikes(c.db, post.ID)
	post.LikeCount = int(likeCount)

	c.result = post
	return post, nil
}

// ---- ReplyContext ----

type ReplyContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	PostID  uint
	ReplyTo uint
	UserID  uint
	Content string

	result *model.Post
}

func NewReplyContext(db *gorm.DB, forumRepo data.ForumRepo, postID, replyTo, userID uint, content string) *ReplyContext {
	return &ReplyContext{db: db, forumRepo: forumRepo, PostID: postID, ReplyTo: replyTo, UserID: userID, Content: content}
}

func (c *ReplyContext) Execute() (*model.Post, error) {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}

	reply := &model.Post{
		BoardID:  0, // 回复不直接关联板块
		UserID:   c.UserID,
		Content:  c.Content,
		ParentID: &c.PostID,
	}
	if err := c.forumRepo.CreatePost(u, reply); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	c.result = reply
	return reply, nil
}

// ---- ToggleLikeContext ----

type ToggleLikeContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	PostID uint
	UserID uint

	liked bool
}

func NewToggleLikeContext(db *gorm.DB, forumRepo data.ForumRepo, postID, userID uint) *ToggleLikeContext {
	return &ToggleLikeContext{db: db, forumRepo: forumRepo, PostID: postID, UserID: userID}
}

// Execute 切换点赞状态：(1) 查是否已赞 → (2) 事务内创建或删除 → (3) Commit。返回当前是否已赞。
func (c *ToggleLikeContext) Execute() (bool, error) {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return false, err
	}

	_, err := c.forumRepo.FindLike(c.db, c.PostID, c.UserID)
	liked := err == nil // 找到了 = 已赞
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		u.Rollback()
		return false, err
	}

	if liked {
		if err := c.forumRepo.DeleteLike(u, c.PostID, c.UserID); err != nil {
			u.Rollback()
			return false, err
		}
		c.liked = false
	} else {
		newLike := &model.PostLike{PostID: c.PostID, UserID: c.UserID}
		if err := c.forumRepo.CreateLike(u, newLike); err != nil {
			u.Rollback()
			return false, err
		}
		c.liked = true
	}

	if err := u.Commit(); err != nil {
		return false, err
	}

	return c.liked, nil
}
