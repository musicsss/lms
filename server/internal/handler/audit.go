package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/dci/data"
	"gorm.io/gorm"
)

type AuditHandler struct {
	db         *gorm.DB
	auditRepo   data.AuditLogRepo
}

func NewAuditHandler(db *gorm.DB, auditRepo data.AuditLogRepo) *AuditHandler {
	return &AuditHandler{db: db, auditRepo: auditRepo}
}

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	severity := c.Query("severity")
	action := c.Query("action")
	sort := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")
	userIDStr := c.Query("user_id")
	userID, _ := strconv.ParseUint(userIDStr, 10, 64)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := h.auditRepo.ListAll(h.db, page, pageSize, severity, action, sort, order, uint(userID))
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "audit: list failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AuditHandler) ListByUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := h.auditRepo.FindByUserID(h.db, uint(userID), page, pageSize)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "audit: list by user failed", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AuditHandler) Severities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"severities": []gin.H{
			{"value": "critical", "label": "严重"},
			{"value": "warning", "label": "警告"},
			{"value": "info", "label": "普通"},
			{"value": "debug", "label": "调试"},
		},
		"actions": []gin.H{
			{"value": "login_success", "label": "登录成功"},
			{"value": "login_failed", "label": "登录失败"},
			{"value": "login_blocked", "label": "登录被封禁"},
			{"value": "file_upload", "label": "文件上传"},
			{"value": "file_delete", "label": "文件删除"},
			{"value": "file_download", "label": "文件下载"},
			{"value": "video_comment", "label": "视频评论"},
			{"value": "video_like", "label": "视频点赞"},
			{"value": "video_collect", "label": "视频收藏"},
			{"value": "video_play", "label": "视频播放"},
			{"value": "danmaku_send", "label": "发送弹幕"},
			{"value": "danmaku_delete", "label": "删除弹幕"},
			{"value": "forum_post", "label": "论坛发帖"},
			{"value": "forum_reply", "label": "论坛回复"},
			{"value": "forum_like", "label": "论坛点赞"},
			{"value": "profile_update", "label": "资料更新"},
			{"value": "password_change", "label": "密码修改"},
			{"value": "avatar_upload", "label": "头像上传"},
			{"value": "admin_access", "label": "管理操作"},
		},
	})
}
