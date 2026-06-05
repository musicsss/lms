package handler

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	filectx "github.com/lms/server/internal/dci/context/file"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/runtimecfg"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

type FileHandler struct {
	db        *gorm.DB
	fileRepo  data.FileRepo
	shareRepo data.ShareRepo
	store     storage.Driver
	rtEngine  *runtimecfg.Engine
}

func NewFileHandler(db *gorm.DB, fileRepo data.FileRepo, shareRepo data.ShareRepo, store storage.Driver, rtEngine *runtimecfg.Engine) *FileHandler {
	return &FileHandler{db: db, fileRepo: fileRepo, shareRepo: shareRepo, store: store, rtEngine: rtEngine}
}

func (h *FileHandler) List(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)

	var parentID *uint
	if pidStr := c.Query("parent_id"); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		pidUint := uint(pid)
		parentID = &pidUint
	}

	ctx := filectx.NewListContext(h.db, h.fileRepo, userID, parentID)
	files, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "file: list failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *FileHandler) Mkdir(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	var input struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := filectx.NewMkdirContext(h.db, h.fileRepo, userID, input.ParentID, input.Name)
	dir, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "file: mkdir failed", "user_id", userID, "name", input.Name, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "file: directory created", "user_id", userID, "dir_id", dir.ID, "name", input.Name)
	c.JSON(http.StatusCreated, dir)
}

func (h *FileHandler) Upload(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	var parentID *uint
	if pidStr := c.PostForm("parent_id"); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		pidUint := uint(pid)
		parentID = &pidUint
	}

	ctx := filectx.NewUploadContext(h.db, h.fileRepo, h.store, h.rtEngine, userID, parentID, f)
	file, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "file: upload failed", "user_id", userID, "filename", f.Filename, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "file: uploaded", "user_id", userID, "file_id", file.ID, "name", file.Name, "size", file.Size)
	c.JSON(http.StatusCreated, file)
}

func (h *FileHandler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := filectx.NewDownloadContext(h.db, h.fileRepo, h.store, uint(id))
	file, reader, err := ctx.Execute()
	if err != nil {
		slog.WarnContext(c.Request.Context(), "file: download not found", "file_id", id, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, reader)
}

func (h *FileHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := filectx.NewDeleteContext(h.db, h.fileRepo, h.store, uint(id))
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "file: delete failed", "file_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "file: deleted", "file_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *FileHandler) Share(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var input struct {
		Password    string `json:"password"`
		ExpireHours int    `json:"expire_hours"`
	}
	c.ShouldBindJSON(&input)

	ctx := filectx.NewShareContext(h.db, h.fileRepo, h.shareRepo, uint(id), input.Password, input.ExpireHours)
	share, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "file: share failed", "file_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "file: shared", "share_token", share.Token, "file_id", id)
	c.JSON(http.StatusCreated, share)
}

func (h *FileHandler) GetShare(c *gin.Context) {
	token := c.Param("token")
	ctx := filectx.NewGetShareContext(h.db, h.shareRepo, token)
	share, err := ctx.Execute()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}

	c.JSON(http.StatusOK, share)
}
