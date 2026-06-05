package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	auditctx "github.com/lms/server/internal/dci/context/audit"
	filectx "github.com/lms/server/internal/dci/context/file"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type VideoSocialHandler struct {
	db        *gorm.DB
	videoRepo data.VideoSocialRepo
	fileRepo  data.FileRepo
	auditRepo data.AuditLogRepo
}

func NewVideoSocialHandler(db *gorm.DB, videoRepo data.VideoSocialRepo, fileRepo data.FileRepo, auditRepo data.AuditLogRepo) *VideoSocialHandler {
	return &VideoSocialHandler{db: db, videoRepo: videoRepo, fileRepo: fileRepo, auditRepo: auditRepo}
}

func (h *VideoSocialHandler) GetComments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	ctx := filectx.NewCommentsContext(h.db, h.videoRepo, uint(id))
	comments, err := ctx.Execute()
	if err != nil { slog.ErrorContext(c.Request.Context(), "video: list comments failed", "video_id", id, "err", err); c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (h *VideoSocialHandler) CreateComment(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	var input struct {
		Content  string `json:"content" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	ctx := filectx.NewCreateCommentContext(h.db, h.videoRepo, uint(id), userID, input.ParentID, input.Content)
	comment, err := ctx.Execute()
	if err != nil { slog.ErrorContext(c.Request.Context(), "video: create comment failed", "video_id", id, "user_id", userID, "err", err); c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	auditctx.NewRecordContext(h.db, h.auditRepo, userID, model.ActionVideoComment, "video", uint(id), "", c.ClientIP(), true).Execute()
	c.JSON(http.StatusCreated, comment)
}

func (h *VideoSocialHandler) ToggleLike(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	ctx := filectx.NewToggleVideoLikeContext(h.db, h.videoRepo, uint(id), userID)
	liked, err := ctx.Execute()
	if err != nil { slog.ErrorContext(c.Request.Context(), "video: toggle like failed", "video_id", id, "user_id", userID, "err", err); c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	auditctx.NewRecordContext(h.db, h.auditRepo, userID, model.ActionVideoLike, "video", uint(id), "", c.ClientIP(), true).Execute()
	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

func (h *VideoSocialHandler) GetLikeStatus(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	ctx := filectx.NewGetLikeStatusContext(h.db, h.videoRepo, uint(id), userID)
	liked, err := ctx.Execute()
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	auditctx.NewRecordContext(h.db, h.auditRepo, userID, model.ActionVideoLike, "video", uint(id), "", c.ClientIP(), true).Execute()
	c.JSON(http.StatusOK, gin.H{"liked": liked})
}
