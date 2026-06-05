package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	filectx "github.com/lms/server/internal/dci/context/file"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/middleware"

	"gorm.io/gorm"
)

type DanmakuHandler struct {
	db          *gorm.DB
	danmakuRepo data.DanmakuRepo
	fileRepo    data.FileRepo
}

func NewDanmakuHandler(db *gorm.DB, danmakuRepo data.DanmakuRepo, fileRepo data.FileRepo) *DanmakuHandler {
	return &DanmakuHandler{db: db, danmakuRepo: danmakuRepo, fileRepo: fileRepo}
}

// Send creates a danmaku (requires auth). Body: {content, time_sec, color, font_size, type}
func (h *DanmakuHandler) Send(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var input struct {
		Content  string  `json:"content" binding:"required"`
		TimeSec  float64 `json:"time_sec" binding:"required"`
		Color    string  `json:"color"`
		FontSize int     `json:"font_size"`
		Type     int     `json:"type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := filectx.NewSendDanmakuContext(h.db, h.danmakuRepo, h.fileRepo, uint(id), userID, input.Content, input.TimeSec, input.Color, input.FontSize, input.Type)
	dm, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "danmaku: send failed", "video_id", id, "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dm)
}

// List returns approved danmaku for a video (public).
func (h *DanmakuHandler) List(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := filectx.NewGetDanmakuContext(h.db, h.danmakuRepo, uint(id))
	danmaku, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "danmaku: list failed", "video_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"danmaku": danmaku})
}

// AdminList lists all danmaku with pagination (admin only).
func (h *DanmakuHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ctx := filectx.NewListDanmakuAdminContext(h.db, h.danmakuRepo, page, pageSize)
	danmaku, total, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "danmaku: admin list failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"danmaku":   danmaku,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminApprove approves a danmaku (admin only).
func (h *DanmakuHandler) AdminApprove(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := filectx.NewUpdateDanmakuStatusContext(h.db, h.danmakuRepo, uint(id), "approved")
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "danmaku: approve failed", "dm_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.InfoContext(c.Request.Context(), "danmaku: approved", "dm_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "approved"})
}

// AdminReject rejects a danmaku (admin only).
func (h *DanmakuHandler) AdminReject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := filectx.NewUpdateDanmakuStatusContext(h.db, h.danmakuRepo, uint(id), "rejected")
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "danmaku: reject failed", "dm_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.InfoContext(c.Request.Context(), "danmaku: rejected", "dm_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "rejected"})
}

// AdminDelete deletes a danmaku (admin only).
func (h *DanmakuHandler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := filectx.NewDeleteDanmakuContext(h.db, h.danmakuRepo, uint(id))
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "danmaku: delete failed", "dm_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.InfoContext(c.Request.Context(), "danmaku: deleted", "dm_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
