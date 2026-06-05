// Package file 提供文件管理相关的 DCI 上下文。
package file

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/runtimecfg"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

// ---- 辅助函数 ----

func isVideoMime(mime string) bool {
	prefixes := []string{"video/", "application/vnd.apple.mpegurl"}
	for _, p := range prefixes {
		if len(mime) >= len(p) && mime[:len(p)] == p {
			return true
		}
	}
	return false
}

func videoStatus(isVideo bool) string {
	if isVideo {
		return model.VideoStatusPending
	}
	return model.VideoStatusNone
}

// ---- UploadContext ----

// UploadContext 处理文件上传的 DCI 上下文。
// 使用 Saga 补偿：存储写入成功后注册删除补偿，DB 写入失败时自动清理存储。
type UploadContext struct {
	db        *gorm.DB
	fileRepo  data.FileRepo
	store     storage.Driver
	rtEngine  *runtimecfg.Engine

	UserID   uint
	ParentID *uint
	Header   *multipart.FileHeader

	storageKey string
	result     *model.File
}

func NewUploadContext(db *gorm.DB, fileRepo data.FileRepo, store storage.Driver, rtEngine *runtimecfg.Engine, userID uint, parentID *uint, header *multipart.FileHeader) *UploadContext {
	return &UploadContext{
		db:       db,
		fileRepo: fileRepo,
		store:    store,
		rtEngine: rtEngine,
		UserID:   userID,
		ParentID: parentID,
		Header:   header,
	}
}

func (c *UploadContext) maxUploadSizeMB() int {
	if c.rtEngine != nil {
		if v := c.rtEngine.GetSet(runtimecfg.TargetFileUpl); v != nil {
			if mb, err := strconv.Atoi(v[runtimecfg.FieldMaxSize]); err == nil && mb > 0 {
				return mb
			}
		}
	}
	return 2048
}

// Execute 执行上传交互：
// (1) 校验大小 → (2) 写存储 (注册删除补偿) → (3) 事务内创建 DB 记录 → (4) Commit。
// 任何步骤失败触发 Rollback → 补偿删除存储文件。
func (c *UploadContext) Execute() (*model.File, error) {
	// Step 1: 校验大小
	maxMB := c.maxUploadSizeMB()
	maxBytes := int64(maxMB) * 1024 * 1024
	if c.Header.Size > maxBytes {
		return nil, fmt.Errorf("file size exceeds limit of %d MB", maxMB)
	}

	// Step 2: 写入存储
	src, err := c.Header.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	ext := filepath.Ext(c.Header.Filename)
	c.storageKey = fmt.Sprintf("%d/%s%s", c.UserID, uuid.New().String(), ext)

	if err := c.store.Put(nil, c.storageKey, src, c.Header.Size); err != nil {
		return nil, fmt.Errorf("store file: %w", err)
	}

	// Step 3: 事务内创建 DB 记录
	u := tx.NewUnit(c.db)
	u.Defer("delete-uploaded-storage", func() error {
		return c.store.Delete(nil, c.storageKey)
	})

	if err := u.Begin(); err != nil {
		u.Rollback()
		return nil, err
	}

	mimeType := c.Header.Header.Get("Content-Type")
	isVid := isVideoMime(mimeType)

	file := &model.File{
		UserID:      c.UserID,
		ParentID:    c.ParentID,
		Name:        c.Header.Filename,
		Size:        c.Header.Size,
		MimeType:    mimeType,
		StorageKey:  c.storageKey,
		IsVideo:     isVid,
		VideoStatus: videoStatus(isVid),
	}

	if err := c.fileRepo.Create(u, file); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	c.result = file
	return file, nil
}

// ---- ListContext ----

type ListContext struct {
	db       *gorm.DB
	fileRepo data.FileRepo

	UserID   uint
	ParentID *uint

	result []model.File
}

func NewListContext(db *gorm.DB, fileRepo data.FileRepo, userID uint, parentID *uint) *ListContext {
	return &ListContext{db: db, fileRepo: fileRepo, UserID: userID, ParentID: parentID}
}

// Execute 只读列出文件，无事务。
func (c *ListContext) Execute() ([]model.File, error) {
	files, err := c.fileRepo.FindByParent(c.db, c.UserID, c.ParentID)
	if err != nil {
		return nil, err
	}
	c.result = files
	return files, nil
}

// ---- MkdirContext ----

type MkdirContext struct {
	db       *gorm.DB
	fileRepo data.FileRepo

	UserID   uint
	ParentID *uint
	Name     string

	result *model.File
}

func NewMkdirContext(db *gorm.DB, fileRepo data.FileRepo, userID uint, parentID *uint, name string) *MkdirContext {
	return &MkdirContext{db: db, fileRepo: fileRepo, UserID: userID, ParentID: parentID, Name: name}
}

func (c *MkdirContext) Execute() (*model.File, error) {
	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}

	dir := &model.File{
		UserID:   c.UserID,
		ParentID: c.ParentID,
		Name:     c.Name,
		IsDir:    true,
	}
	if err := c.fileRepo.Create(u, dir); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	c.result = dir
	return dir, nil
}

// ---- DownloadContext ----

type DownloadContext struct {
	db       *gorm.DB
	fileRepo data.FileRepo
	store    storage.Driver

	FileID uint

	file   *model.File
	reader io.ReadCloser
}

func NewDownloadContext(db *gorm.DB, fileRepo data.FileRepo, store storage.Driver, fileID uint) *DownloadContext {
	return &DownloadContext{db: db, fileRepo: fileRepo, store: store, FileID: fileID}
}

// Execute 只读下载，返回文件元数据 + 内容流。
func (c *DownloadContext) Execute() (*model.File, io.ReadCloser, error) {
	file, err := c.fileRepo.FindByID(c.db, c.FileID)
	if err != nil {
		return nil, nil, err
	}

	reader, err := c.store.Get(nil, file.StorageKey)
	if err != nil {
		return nil, nil, err
	}

	c.file = file
	c.reader = reader
	return file, reader, nil
}

// ---- DeleteContext ----

// DeleteContext 处理文件/目录删除的 DCI 上下文。
// 递归收集子孙节点，事务内批量删除 DB 记录 + 注册存储删除补偿。
type DeleteContext struct {
	db       *gorm.DB
	fileRepo data.FileRepo
	store    storage.Driver

	FileID uint

	toDelete []storageKey // 待删除的存储 key 列表
}

type storageKey struct {
	key string
}

func NewDeleteContext(db *gorm.DB, fileRepo data.FileRepo, store storage.Driver, fileID uint) *DeleteContext {
	return &DeleteContext{db: db, fileRepo: fileRepo, store: store, FileID: fileID}
}

// Execute 执行删除交互：
// (1) 加载文件 → (2) 递归收集所有子孙的存储 key → (3) 事务内删除 DB 记录（注册存储删除补偿）→ (4) Commit。
func (c *DeleteContext) Execute() error {
	file, err := c.fileRepo.FindByID(c.db, c.FileID)
	if err != nil {
		return err
	}

	// 递归收集待删除的文件（非目录文件有存储 key）
	if err := c.collect(file); err != nil {
		return err
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return err
	}

	// 注册所有存储删除补偿
	for _, sk := range c.toDelete {
		key := sk.key
		u.Defer("delete-storage-"+key, func() error {
			return c.store.Delete(nil, key)
		})
	}

	// 递归删除 DB 记录
	if err := c.deleteRecursive(u, file); err != nil {
		u.Rollback()
		return err
	}

	return u.Commit()
}

// collect 递归收集需要删除的存储 key。
func (c *DeleteContext) collect(f *model.File) error {
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

// deleteRecursive 自底向上递归删除 DB 记录。
func (c *DeleteContext) deleteRecursive(u *tx.Unit, f *model.File) error {
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

// ---- ShareContext ----

type ShareContext struct {
	db        *gorm.DB
	fileRepo  data.FileRepo
	shareRepo data.ShareRepo

	FileID      uint
	Password    string
	ExpireHours int

	result *model.FileShare
}

func NewShareContext(db *gorm.DB, fileRepo data.FileRepo, shareRepo data.ShareRepo, fileID uint, password string, expireHours int) *ShareContext {
	return &ShareContext{
		db:          db,
		fileRepo:    fileRepo,
		shareRepo:   shareRepo,
		FileID:      fileID,
		Password:    password,
		ExpireHours: expireHours,
	}
}

func (c *ShareContext) Execute() (*model.FileShare, error) {
	_, err := c.fileRepo.FindByID(c.db, c.FileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("file not found")
		}
		return nil, err
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, err
	}

	share := &model.FileShare{
		FileID:   c.FileID,
		Token:    uuid.New().String(),
		Password: c.Password,
	}
	if err := c.shareRepo.Create(u, share); err != nil {
		u.Rollback()
		return nil, err
	}

	if err := u.Commit(); err != nil {
		return nil, err
	}

	c.result = share
	return share, nil
}

// ---- GetShareContext ----

type GetShareContext struct {
	db        *gorm.DB
	shareRepo data.ShareRepo

	Token string

	result *model.FileShare
}

func NewGetShareContext(db *gorm.DB, shareRepo data.ShareRepo, token string) *GetShareContext {
	return &GetShareContext{db: db, shareRepo: shareRepo, Token: token}
}

func (c *GetShareContext) Execute() (*model.FileShare, error) {
	share, err := c.shareRepo.FindByToken(c.db, c.Token)
	if err != nil {
		return nil, err
	}
	c.result = share
	return share, nil
}
