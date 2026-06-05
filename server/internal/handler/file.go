package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	auditctx "github.com/lms/server/internal/dci/context/audit"
	filectx "github.com/lms/server/internal/dci/context/file"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/presence"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/runtimecfg"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

type FileHandler struct {
	db          *gorm.DB
	fileRepo    data.FileRepo
	shareRepo   data.ShareRepo
	videoRepo   data.VideoSocialRepo
	store       storage.Driver
	rtEngine    *runtimecfg.Engine
	presenceHub *presence.Hub
	auditRepo   data.AuditLogRepo
}

func NewFileHandler(db *gorm.DB, fileRepo data.FileRepo, shareRepo data.ShareRepo, videoRepo data.VideoSocialRepo, store storage.Driver, rtEngine *runtimecfg.Engine, presenceHub *presence.Hub, auditRepo data.AuditLogRepo) *FileHandler {
	return &FileHandler{db: db, fileRepo: fileRepo, shareRepo: shareRepo, videoRepo: videoRepo, store: store, rtEngine: rtEngine, presenceHub: presenceHub, auditRepo: auditRepo}
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

	// Generate thumbnail asynchronously for video files
	if file.IsVideo {
		go h.generateThumbnail(file)
	}
}

// generateThumbnail extracts a frame from the video using ffmpeg
func (h *FileHandler) generateThumbnail(file *model.File) {
	thumbKey := strings.TrimSuffix(file.StorageKey, filepath.Ext(file.StorageKey)) + ".jpg"
	inputPath := h.store.(*storage.LocalDriver).ResolvePath(file.StorageKey)
	outputPath := h.store.(*storage.LocalDriver).ResolvePath(thumbKey)

	cmd := exec.Command("ffmpeg",
		"-y",
		"-ss", "00:00:01",
		"-i", inputPath,
		"-vframes", "1",
		"-q:v", "5",
		"-vf", "scale=480:-1",
		outputPath,
	)
	if err := cmd.Run(); err != nil {
		slog.Warn("thumbnail: ffmpeg failed", "file_id", file.ID, "err", err)
		return
	}

	// Update thumb_key in DB
	if err := h.db.Model(file).Update("thumb_key", thumbKey).Error; err != nil {
		slog.Warn("thumbnail: db update failed", "file_id", file.ID, "err", err)
	}
	slog.Info("thumbnail: generated", "file_id", file.ID, "thumb_key", thumbKey)
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
		slog.ErrorContext(c.Request.Context(), "file: download failed", "file_id", id, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "inline; filename=\""+file.Name+"\"")
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	c.Header("Accept-Ranges", "bytes")

	rangeHeader := c.GetHeader("Range")
	if rangeHeader == "" {
		c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
		c.Status(http.StatusOK)
		io.Copy(c.Writer, reader)
		return
	}

	var start, end int64
	parsed := false
	if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err == nil {
		parsed = true
	} else if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err == nil {
		end = file.Size - 1
		parsed = true
	}

	if !parsed || start < 0 || start >= file.Size {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", file.Size))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	if end >= file.Size {
		end = file.Size - 1
	}

	length := end - start + 1
	rangeReader, err := h.store.Range(nil, file.StorageKey, start, length)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "file: range read failed", "file_id", id, "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rangeReader.Close()

	c.Header("Content-Length", strconv.FormatInt(length, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.Size))
	c.Status(http.StatusPartialContent)
	io.Copy(c.Writer, rangeReader)
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

	userID := c.GetUint(middleware.CtxKeyUserID)
	auditctx.NewRecordContext(h.db, h.auditRepo, userID, model.ActionFileDelete, "file", uint(id), "", c.ClientIP(), true).Execute()
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

// Play serves video files for direct streaming via <video> tag.
// This endpoint is deliberately public (no auth) because browsers do not
// send Authorization headers on <video> <source> requests.
// Access control is enforced by only allowing is_video=true files.
// Play serves video files for direct streaming via <video> tag.
// This endpoint is deliberately public (no auth) because browsers do not
// send Authorization headers on <video> <source> requests.
// Access control is enforced by only allowing is_video=true files.
func (h *FileHandler) Play(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "video: invalid id param", "id_raw", c.Param("id"), "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	file, err := h.fileRepo.FindByID(h.db, uint(id))
	if err != nil {
		slog.WarnContext(c.Request.Context(), "video: file not found", "file_id", id, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	if !file.IsVideo {
		slog.WarnContext(c.Request.Context(), "video: not a video file", "file_id", id, "mime", file.MimeType)
		c.JSON(http.StatusForbidden, gin.H{"error": "not a video file"})
		return
	}

	// Increment view count asynchronously
	go func() {
		if err := h.db.Model(&model.File{}).Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
			slog.Warn("video: failed to increment view", "file_id", id, "err", err)
		}
	}()

	// Check range header
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Disposition", "inline; filename=\""+file.Name+"\"")

	rangeHeader := c.GetHeader("Range")
	if rangeHeader == "" {
		fullReader, err := h.store.Get(nil, file.StorageKey)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "video: failed to open storage", "file_id", id, "storage_key", file.StorageKey, "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}
		defer fullReader.Close()
		c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
		c.Status(http.StatusOK)
		io.Copy(c.Writer, fullReader)
		return
	}

	// Parse Range: bytes=start-end or bytes=start-
	var start, end int64
	parsed := false
	if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err == nil {
		parsed = true
	} else if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err == nil {
		end = file.Size - 1
		parsed = true
	}

	if !parsed || start < 0 || start >= file.Size {
		slog.WarnContext(c.Request.Context(), "video: invalid range", "file_id", id, "range", rangeHeader, "file_size", file.Size)
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", file.Size))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	if end >= file.Size {
		end = file.Size - 1
	}

	length := end - start + 1
	rangeReader, err := h.store.Range(nil, file.StorageKey, start, length)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "video: range read failed", "file_id", id, "storage_key", file.StorageKey, "offset", start, "length", length, "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rangeReader.Close()

	c.Header("Content-Length", strconv.FormatInt(length, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.Size))
	c.Status(http.StatusPartialContent)
	io.Copy(c.Writer, rangeReader)
}
// RandomVideos returns random video files for the homepage feed.

// VideoInfo returns video metadata including uploader info for the player page.
func (h *FileHandler) VideoInfo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "video-info: invalid id", "id_raw", c.Param("id"), "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	file, err := h.fileRepo.FindByID(h.db, uint(id))
	if err != nil {
		slog.WarnContext(c.Request.Context(), "video-info: not found", "file_id", id, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	if !file.IsVideo {
		slog.WarnContext(c.Request.Context(), "video-info: not a video", "file_id", id)
		c.JSON(http.StatusForbidden, gin.H{"error": "not a video file"})
		return
	}

	// Load user association
	h.db.Model(file).Association("User").Find(&file.User)
	// Load like and comment counts
	likeCount, _ := h.videoRepo.CountVideoLikes(h.db, file.ID)
	commentCount, _ := h.videoRepo.CountVideoComments(h.db, file.ID)
	file.LikeCount = likeCount
	file.CommentCount = commentCount

	slog.InfoContext(c.Request.Context(), "video-info: served", "file_id", id, "likes", likeCount, "comments", commentCount)
	c.JSON(http.StatusOK, file)
}
// Heartbeat records the current user as actively watching a video.
func (h *FileHandler) Heartbeat(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	userID := c.GetUint(middleware.CtxKeyUserID)
	if h.presenceHub != nil {
		h.presenceHub.Heartbeat(uint(id), userID)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Watchers returns the number of active watchers for a video.
func (h *FileHandler) Watchers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var count int64
	if h.presenceHub != nil {
		count = h.presenceHub.Count(uint(id), 15*time.Second)
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// Thumbnail serves a video thumbnail image.
func (h *FileHandler) Thumbnail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	file, err := h.fileRepo.FindByID(h.db, uint(id))
	if err != nil || file.ThumbKey == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "thumbnail not found"})
		return
	}

	reader, err := h.store.Get(c.Request.Context(), file.ThumbKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "thumbnail not available"})
		return
	}
	defer reader.Close()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=86400")
	io.Copy(c.Writer, reader)
}

func (h *FileHandler) RandomVideos(c *gin.Context) {
	ctx := filectx.NewRandomVideosContext(h.db, h.fileRepo, 20)
	videos, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "videos: random query failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"videos": videos})
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
