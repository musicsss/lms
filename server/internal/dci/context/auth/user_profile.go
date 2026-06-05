package auth

import (
	"errors"

	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---- 公共错误 ----

var (
	ErrWrongPassword = errors.New("wrong password")
)

// ---- GetProfileContext ----

type GetProfileContext struct {
	db       *gorm.DB
	userRepo data.UserRepo
	UserID   uint

	result *model.User
}

func NewGetProfileContext(db *gorm.DB, userRepo data.UserRepo, userID uint) *GetProfileContext {
	return &GetProfileContext{db: db, userRepo: userRepo, UserID: userID}
}

func (c *GetProfileContext) Execute() (*model.User, error) {
	user, err := c.userRepo.FindByID(c.db, c.UserID)
	if err != nil {
		return nil, err
	}
	c.result = user
	return user, nil
}

// ---- UpdateProfileContext ----

type UpdateProfileContext struct {
	db        *gorm.DB
	userRepo  data.UserRepo
	UserID    uint
	Nickname  string
	Bio       string
	Email     string
	AvatarURL string

	result *model.User
}

func NewUpdateProfileContext(db *gorm.DB, userRepo data.UserRepo, userID uint, nickname, bio, email, avatarURL string) *UpdateProfileContext {
	return &UpdateProfileContext{
		db:        db,
		userRepo:  userRepo,
		UserID:    userID,
		Nickname:  nickname,
		Bio:       bio,
		Email:     email,
		AvatarURL: avatarURL,
	}
}

func (c *UpdateProfileContext) Execute() (*model.User, error) {
	updates := map[string]interface{}{
		"nickname":   c.Nickname,
		"bio":        c.Bio,
		"email":      c.Email,
		"avatar_url": c.AvatarURL,
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}

	if err := c.userRepo.UpdateProfile(u, c.UserID, updates); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	user, err := c.userRepo.FindByID(c.db, c.UserID)
	if err != nil {
		return nil, err
	}
	c.result = user
	return user, nil
}

// ---- UpdatePasswordContext ----

type UpdatePasswordContext struct {
	db          *gorm.DB
	userRepo    data.UserRepo
	UserID      uint
	OldPassword string
	NewPassword string
}

func NewUpdatePasswordContext(db *gorm.DB, userRepo data.UserRepo, userID uint, oldPassword, newPassword string) *UpdatePasswordContext {
	return &UpdatePasswordContext{
		db:          db,
		userRepo:    userRepo,
		UserID:      userID,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}
}

func (c *UpdatePasswordContext) Execute() error {
	user, err := c.userRepo.FindByID(c.db, c.UserID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(c.OldPassword)); err != nil {
		return ErrWrongPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(c.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return err
	}

	if err := c.userRepo.UpdatePassword(u, c.UserID, string(hash)); err != nil {
		u.Rollback()
		return err
	}

	return u.Commit()
}

// ---- GetUserFilesContext ----

type GetUserFilesContext struct {
	db       *gorm.DB
	userRepo data.UserRepo
	UserID   uint
	FileType string
	Page     int
	PageSize int

	files []model.File
	total int64
}

func NewGetUserFilesContext(db *gorm.DB, userRepo data.UserRepo, userID uint, fileType string, page, pageSize int) *GetUserFilesContext {
	return &GetUserFilesContext{
		db:       db,
		userRepo: userRepo,
		UserID:   userID,
		FileType: fileType,
		Page:     page,
		PageSize: pageSize,
	}
}

func (c *GetUserFilesContext) Execute() ([]model.File, int64, error) {
	files, total, err := c.userRepo.FindFilesByUser(c.db, c.UserID, c.FileType, c.Page, c.PageSize)
	if err != nil {
		return nil, 0, err
	}
	c.files = files
	c.total = total
	return files, total, nil
}

// ---- GetUserPostsContext ----

type GetUserPostsContext struct {
	db       *gorm.DB
	userRepo data.UserRepo
	UserID   uint
	Page     int
	PageSize int

	posts []model.Post
	total int64
}

func NewGetUserPostsContext(db *gorm.DB, userRepo data.UserRepo, userID uint, page, pageSize int) *GetUserPostsContext {
	return &GetUserPostsContext{
		db:       db,
		userRepo: userRepo,
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	}
}

func (c *GetUserPostsContext) Execute() ([]model.Post, int64, error) {
	posts, total, err := c.userRepo.FindPostsByUser(c.db, c.UserID, c.Page, c.PageSize)
	if err != nil {
		return nil, 0, err
	}
	c.posts = posts
	c.total = total
	return posts, total, nil
}

// ---- GetUserLikedVideosContext ----

type GetUserLikedVideosContext struct {
	db       *gorm.DB
	userRepo data.UserRepo
	UserID   uint
	Page     int
	PageSize int

	files []model.File
	total int64
}

func NewGetUserLikedVideosContext(db *gorm.DB, userRepo data.UserRepo, userID uint, page, pageSize int) *GetUserLikedVideosContext {
	return &GetUserLikedVideosContext{
		db:       db,
		userRepo: userRepo,
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	}
}

func (c *GetUserLikedVideosContext) Execute() ([]model.File, int64, error) {
	files, total, err := c.userRepo.FindLikedVideos(c.db, c.UserID, c.Page, c.PageSize)
	if err != nil {
		return nil, 0, err
	}
	c.files = files
	c.total = total
	return files, total, nil
}