package handler

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/service/file"
)

type FileHandler struct {
	svc *file.Service
}

func NewFileHandler(svc *file.Service) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) List(c *gin.Context) {
	userID := c.GetUint("userID")

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

	files, err := h.svc.List(userID, parentID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "file: list failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

type mkdirInput struct {
	Name     string `json:"name" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

func (h *FileHandler) Mkdir(c *gin.Context) {
	userID := c.GetUint("userID")
	var input mkdirInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dir, err := h.svc.CreateDir(userID, input.ParentID, input.Name)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "file: mkdir failed", "user_id", userID, "name", input.Name, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "file: directory created", "user_id", userID, "dir_id", dir.ID, "name", input.Name)
	c.JSON(http.StatusCreated, dir)
}

func (h *FileHandler) Upload(c *gin.Context) {
	userID := c.GetUint("userID")

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

	file, err := h.svc.Upload(userID, parentID, f)
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

	file, reader, err := h.svc.Download(uint(id))
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

	if err := h.svc.Delete(uint(id)); err != nil {
		slog.ErrorContext(c.Request.Context(), "file: delete failed", "file_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "file: deleted", "file_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

type shareInput struct {
	Password    string `json:"password"`
	ExpireHours int    `json:"expire_hours"`
}

func (h *FileHandler) Share(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var input shareInput
	c.ShouldBindJSON(&input)

	share, err := h.svc.CreateShare(uint(id), input.Password, input.ExpireHours)
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
	share, err := h.svc.GetShare(token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}

	c.JSON(http.StatusOK, share)
}
