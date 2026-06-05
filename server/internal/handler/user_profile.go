package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	authctx "github.com/lms/server/internal/dci/context/auth"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

type UserProfileHandler struct {
	db              *gorm.DB
	userRepo        data.UserRepo
	fileRepo        data.FileRepo
	forumRepo       data.ForumRepo
	videoSocialRepo data.VideoSocialRepo
	store           storage.Driver
	auditRepo       data.AuditLogRepo
}

func NewUserProfileHandler(db *gorm.DB, userRepo data.UserRepo, fileRepo data.FileRepo, forumRepo data.ForumRepo, videoSocialRepo data.VideoSocialRepo, store storage.Driver, auditRepo data.AuditLogRepo) *UserProfileHandler {
	return &UserProfileHandler{
		db:              db,
		userRepo:        userRepo,
		fileRepo:        fileRepo,
		forumRepo:       forumRepo,
		videoSocialRepo: videoSocialRepo,
		store:           store,
		auditRepo:       auditRepo,
	}
}

func (h *UserProfileHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	ctx := authctx.NewGetProfileContext(h.db, h.userRepo, userID)
	user, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "user: get profile failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserProfileHandler) GetUserProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := authctx.NewGetProfileContext(h.db, h.userRepo, uint(id))
	user, err := ctx.Execute()
	if err != nil {
		slog.WarnContext(c.Request.Context(), "user: get user profile not found", "user_id", id, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"nickname":   user.Nickname,
		"bio":        user.Bio,
		"avatar_url": user.AvatarURL,
		"created_at": user.CreatedAt,
	})
}

func (h *UserProfileHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	var input struct {
		Nickname  string `json:"nickname"`
		Bio       string `json:"bio"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := authctx.NewUpdateProfileContext(h.db, h.userRepo, userID, input.Nickname, input.Bio, input.Email, input.AvatarURL)
	user, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "user: update profile failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.InfoContext(c.Request.Context(), "user: profile updated", "user_id", userID)
	c.JSON(http.StatusOK, user)
}

func (h *UserProfileHandler) UpdatePassword(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	var input struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := authctx.NewUpdatePasswordContext(h.db, h.userRepo, userID, input.OldPassword, input.NewPassword)
	if err := ctx.Execute(); err != nil {
		if err == authctx.ErrWrongPassword {
			slog.WarnContext(c.Request.Context(), "user: old password wrong", "user_id", userID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		slog.ErrorContext(c.Request.Context(), "user: update password failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.InfoContext(c.Request.Context(), "user: password updated", "user_id", userID)
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

func (h *UserProfileHandler) GetUserFiles(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	fileType := c.DefaultQuery("type", "all")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	ctx := authctx.NewGetUserFilesContext(h.db, h.userRepo, userID, fileType, page, pageSize)
	files, total, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "user: list files failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "page": page, "page_size": pageSize, "files": files})
}

func (h *UserProfileHandler) GetUserPosts(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	ctx := authctx.NewGetUserPostsContext(h.db, h.userRepo, userID, page, pageSize)
	posts, total, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "user: list posts failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "page": page, "page_size": pageSize, "posts": posts})
}

func (h *UserProfileHandler) GetUserLikedVideos(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	ctx := authctx.NewGetUserLikedVideosContext(h.db, h.userRepo, userID, page, pageSize)
	files, total, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "user: list liked videos failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "page": page, "page_size": pageSize, "files": files})
}

// UploadAvatar handles avatar image upload
func (h *UserProfileHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)

	f, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar file is required"})
		return
	}

	if f.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar too large, max 5MB"})
		return
	}

	contentType := f.Header.Get("Content-Type")
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	case "image/jpeg":
		ext = ".jpg"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "only jpeg, png, webp, gif allowed"})
		return
	}

	file, err := f.Open()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "user: open avatar file failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	defer file.Close()

	storageKey := fmt.Sprintf("avatars/%d%s", userID, ext)

	if err := h.store.Put(c.Request.Context(), storageKey, file, f.Size); err != nil {
		slog.ErrorContext(c.Request.Context(), "user: save avatar failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar"})
		return
	}

	avatarURL := "/api/v1/files/download-by-key/" + storageKey

	ctx := authctx.NewUpdateProfileContext(h.db, h.userRepo, userID, "", "", "", avatarURL)
	if _, err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "user: update avatar url failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar"})
		return
	}

	slog.InfoContext(c.Request.Context(), "user: avatar uploaded", "user_id", userID, "url", avatarURL)
	c.JSON(http.StatusOK, gin.H{"avatar_url": avatarURL})
}

// DownloadByKey serves files by storage key (avatars etc)
func (h *UserProfileHandler) DownloadByKey(c *gin.Context) {
	key := c.Param("key")
	reader, err := h.store.Get(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer reader.Close()

	ctype := "application/octet-stream"
	if len(key) > 4 {
		switch key[len(key)-4:] {
		case ".jpg":
			ctype = "image/jpeg"
		case ".png":
			ctype = "image/png"
		case ".webp":
			ctype = "image/webp"
		case ".gif":
			ctype = "image/gif"
		}
	}
	c.Header("Content-Type", ctype)
	c.Header("Content-Disposition", "inline")
	c.Status(http.StatusOK)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				break
			}
		}
		if readErr != nil {
			break
		}
	}
}