// Package admin 提供管理后台相关的 DCI 上下文。
package admin

import (
	"errors"

	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

// ---- Stats ----

type StatsResult struct {
	UserCount  int64 `json:"user_count"`
	FileCount  int64 `json:"file_count"`
	FileSize   int64 `json:"file_size"`
	PostCount  int64 `json:"post_count"`
	BoardCount int64 `json:"board_count"`
}

type GetStatsContext struct {
	db        *gorm.DB
	userRepo  data.UserRepo
	fileRepo  data.FileRepo
	forumRepo data.ForumRepo

	result *StatsResult
}

func NewGetStatsContext(db *gorm.DB, userRepo data.UserRepo, fileRepo data.FileRepo, forumRepo data.ForumRepo) *GetStatsContext {
	return &GetStatsContext{db: db, userRepo: userRepo, fileRepo: fileRepo, forumRepo: forumRepo}
}

func (c *GetStatsContext) Execute() (*StatsResult, error) {
	userCount, err := c.userRepo.Count(c.db)
	if err != nil {
		return nil, err
	}
	fileCount, err := c.fileRepo.CountAll(c.db)
	if err != nil {
		return nil, err
	}
	fileSize, err := c.fileRepo.SumSize(c.db)
	if err != nil {
		return nil, err
	}
	postCount, err := c.forumRepo.CountPosts(c.db)
	if err != nil {
		return nil, err
	}
	boards, err := c.forumRepo.ListBoards(c.db)
	if err != nil {
		return nil, err
	}

	c.result = &StatsResult{
		UserCount:  userCount,
		FileCount:  fileCount,
		FileSize:   fileSize,
		PostCount:  postCount,
		BoardCount: int64(len(boards)),
	}
	return c.result, nil
}

// ---- Users ----

type ListUsersContext struct {
	db       *gorm.DB
	userRepo data.UserRepo

	Page     int
	PageSize int
	Search   string

	users []model.User
	total int64
}

func NewListUsersContext(db *gorm.DB, userRepo data.UserRepo, page, pageSize int, search string) *ListUsersContext {
	if page < 1 {
		page = 1
	}
	return &ListUsersContext{db: db, userRepo: userRepo, Page: page, PageSize: pageSize, Search: search}
}

func (c *ListUsersContext) Execute() ([]model.User, int64, error) {
	offset := (c.Page - 1) * c.PageSize
	users, total, err := c.userRepo.List(c.db, offset, c.PageSize, c.Search)
	if err != nil {
		return nil, 0, err
	}
	c.users = users
	c.total = total
	return users, total, nil
}

// ---- UpdateUserRoleContext ----

type UpdateUserRoleContext struct {
	db       *gorm.DB
	userRepo data.UserRepo

	UserID uint
	Role   string
}

func NewUpdateUserRoleContext(db *gorm.DB, userRepo data.UserRepo, userID uint, role string) *UpdateUserRoleContext {
	return &UpdateUserRoleContext{db: db, userRepo: userRepo, UserID: userID, Role: role}
}

func (c *UpdateUserRoleContext) Execute() error {
	if c.Role != model.RoleAdmin && c.Role != model.RoleUser {
		return errors.New("invalid role")
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return err
	}
	if err := c.userRepo.UpdateRole(u, c.UserID, c.Role); err != nil {
		u.Rollback()
		return err
	}
	return u.Commit()
}

// ---- DeleteUserContext ----

type DeleteUserContext struct {
	db       *gorm.DB
	userRepo data.UserRepo

	UserID uint
}

func NewDeleteUserContext(db *gorm.DB, userRepo data.UserRepo, userID uint) *DeleteUserContext {
	return &DeleteUserContext{db: db, userRepo: userRepo, UserID: userID}
}

func (c *DeleteUserContext) Execute() error {
	// 最后管理员检查（读操作，在事务外）
	user, err := c.userRepo.FindByID(c.db, c.UserID)
	if err != nil {
		return err
	}
	if user.Role == model.RoleAdmin {
		adminCount, err := c.userRepo.CountByRole(c.db, model.RoleAdmin)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return errors.New("cannot delete the last admin")
		}
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return err
	}
	if err := c.userRepo.Delete(u, c.UserID); err != nil {
		u.Rollback()
		return err
	}
	return u.Commit()
}

// ---- Files ----

type ListFilesContext struct {
	db       *gorm.DB
	fileRepo data.FileRepo

	Page     int
	PageSize int

	files []model.File
	total int64
}

func NewListFilesContext(db *gorm.DB, fileRepo data.FileRepo, page, pageSize int) *ListFilesContext {
	if page < 1 {
		page = 1
	}
	return &ListFilesContext{db: db, fileRepo: fileRepo, Page: page, PageSize: pageSize}
}

func (c *ListFilesContext) Execute() ([]model.File, int64, error) {
	offset := (c.Page - 1) * c.PageSize
	files, total, err := c.fileRepo.ListAll(c.db, offset, c.PageSize)
	if err != nil {
		return nil, 0, err
	}
	c.files = files
	c.total = total
	return files, total, nil
}

// ---- DeleteFileContext (admin) ----

type DeleteFileContext struct {
	db       *gorm.DB
	fileRepo data.FileRepo
	store    storage.Driver

	FileID uint

	toDelete []storageKey
}

type storageKey struct {
	key string
}

func NewDeleteFileContext(db *gorm.DB, fileRepo data.FileRepo, store storage.Driver, fileID uint) *DeleteFileContext {
	return &DeleteFileContext{db: db, fileRepo: fileRepo, store: store, FileID: fileID}
}

// Execute 递归删除文件/目录，注册存储删除补偿。
func (c *DeleteFileContext) Execute() error {
	file, err := c.fileRepo.FindByID(c.db, c.FileID)
	if err != nil {
		return err
	}

	if err := c.collect(file); err != nil {
		return err
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return err
	}

	for _, sk := range c.toDelete {
		key := sk.key
		u.Defer("delete-storage-"+key, func() error {
			return c.store.Delete(nil, key)
		})
	}

	if err := c.deleteRecursive(u, file); err != nil {
		u.Rollback()
		return err
	}

	return u.Commit()
}

func (c *DeleteFileContext) collect(f *model.File) error {
	if !f.IsDir {
		c.toDelete = append(c.toDelete, storageKey{key: f.StorageKey})
	}
	children, err := c.fileRepo.FindChildren(c.db, f.ID)
	if err != nil {
		return err
	}
	for _, child := range children {
		if child.IsDir {
			if err := c.collect(&child); err != nil {
				return err
			}
		} else {
			c.toDelete = append(c.toDelete, storageKey{key: child.StorageKey})
		}
	}
	return nil
}

func (c *DeleteFileContext) deleteRecursive(u *tx.Unit, f *model.File) error {
	children, err := c.fileRepo.FindChildren(c.db, f.ID)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := c.deleteRecursive(u, &child); err != nil {
			return err
		}
	}
	return c.fileRepo.Delete(u, f.ID)
}

// ---- Boards ----

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

// ---- CreateBoardContext ----

type CreateBoardContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	Name        string
	Slug        string
	Description string
	SortOrder   int

	result *model.Board
}

func NewCreateBoardContext(db *gorm.DB, forumRepo data.ForumRepo, name, slug, description string, sortOrder int) *CreateBoardContext {
	return &CreateBoardContext{db: db, forumRepo: forumRepo, Name: name, Slug: slug, Description: description, SortOrder: sortOrder}
}

func (c *CreateBoardContext) Execute() (*model.Board, error) {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}

	board := &model.Board{
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		SortOrder:   c.SortOrder,
	}
	if err := c.forumRepo.CreateBoard(u, board); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	c.result = board
	return board, nil
}

// ---- UpdateBoardContext ----

type UpdateBoardContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	BoardID     uint
	Name        string
	Slug        string
	Description string
	SortOrder   int

	result *model.Board
}

func NewUpdateBoardContext(db *gorm.DB, forumRepo data.ForumRepo, boardID uint, name, slug, description string, sortOrder int) *UpdateBoardContext {
	return &UpdateBoardContext{
		db:          db,
		forumRepo:   forumRepo,
		BoardID:     boardID,
		Name:        name,
		Slug:        slug,
		Description: description,
		SortOrder:   sortOrder,
	}
}

func (c *UpdateBoardContext) Execute() (*model.Board, error) {
	board, err := c.forumRepo.FindBoardByID(c.db, c.BoardID)
	if err != nil {
		return nil, err
	}

	if c.Name != "" {
		board.Name = c.Name
	}
	if c.Slug != "" {
		board.Slug = c.Slug
	}
	if c.Description != "" {
		board.Description = c.Description
	}
	board.SortOrder = c.SortOrder

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}
	if err := c.forumRepo.UpdateBoard(u, board); err != nil {
		u.Rollback()
		return nil, err
	}
	if err := u.Commit(); err != nil {
		return nil, err
	}

	c.result = board
	return board, nil
}

// ---- DeleteBoardContext ----

type DeleteBoardContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	BoardID uint
}

func NewDeleteBoardContext(db *gorm.DB, forumRepo data.ForumRepo, boardID uint) *DeleteBoardContext {
	return &DeleteBoardContext{db: db, forumRepo: forumRepo, BoardID: boardID}
}

func (c *DeleteBoardContext) Execute() error {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return err
	}
	if err := c.forumRepo.DeleteBoard(u, c.BoardID); err != nil {
		u.Rollback()
		return err
	}
	return u.Commit()
}

// ---- DeletePostContext (admin) ----

type DeletePostContext struct {
	db        *gorm.DB
	forumRepo data.ForumRepo

	PostID uint
}

func NewDeletePostContext(db *gorm.DB, forumRepo data.ForumRepo, postID uint) *DeletePostContext {
	return &DeletePostContext{db: db, forumRepo: forumRepo, PostID: postID}
}

func (c *DeletePostContext) Execute() error {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return err
	}
	if err := c.forumRepo.DeleteReplies(u, c.PostID); err != nil {
		u.Rollback()
		return err
	}
	if err := c.forumRepo.DeletePost(u, c.PostID); err != nil {
		u.Rollback()
		return err
	}
	return u.Commit()
}

// ---- ListPostsContext (admin) ----

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
	if page < 1 {
		page = 1
	}
	return &ListPostsContext{db: db, forumRepo: forumRepo, BoardID: boardID, Page: page, PageSize: pageSize}
}

func (c *ListPostsContext) Execute() ([]model.Post, int64, error) {
	posts, total, err := c.forumRepo.ListPosts(c.db, c.BoardID, c.Page, c.PageSize)
	if err != nil {
		return nil, 0, err
	}
	c.posts = posts
	c.total = total
	return posts, total, nil
}
