package model

import (
	"time"
)

const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
	SeverityDebug    = "debug"
)

const (
	ActionLoginSuccess    = "login_success"
	ActionLoginFailed     = "login_failed"
	ActionLoginBlocked    = "login_blocked"
	ActionLogout          = "logout"
	ActionRegister        = "register"
	ActionFileUpload      = "file_upload"
	ActionFileDelete      = "file_delete"
	ActionFileDownload    = "file_download"
	ActionVideoComment    = "video_comment"
	ActionVideoLike       = "video_like"
	ActionVideoCollect    = "video_collect"
	ActionVideoPlay       = "video_play"
	ActionDanmakuSend     = "danmaku_send"
	ActionDanmakuDelete   = "danmaku_delete"
	ActionForumPost       = "forum_post"
	ActionForumReply      = "forum_reply"
	ActionForumLike       = "forum_like"
	ActionProfileUpdate   = "profile_update"
	ActionPasswordChange  = "password_change"
	ActionAvatarUpload    = "avatar_upload"
	ActionAdminAccess     = "admin_access"
	ActionAdminDeleteUser = "admin_delete_user"
	ActionAdminDeleteFile = "admin_delete_file"
	ActionAdminConfig     = "admin_config"
)

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index" json:"user_id"`
	Username   string    `gorm:"size:64" json:"username"`
	Action     string    `gorm:"size:64;index;not null" json:"action"`
	Severity   string    `gorm:"size:16;index;not null;default:info" json:"severity"`
	Resource   string    `gorm:"size:128" json:"resource"`
	ResourceID uint      `gorm:"index" json:"resource_id"`
	Detail     string    `gorm:"size:512" json:"detail"`
	IP         string    `gorm:"size:64" json:"ip"`
	Success    bool      `gorm:"not null;default:true" json:"success"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

var ActionSeverityMap = map[string]string{
	ActionLoginSuccess:    SeverityInfo,
	ActionLoginFailed:     SeverityWarning,
	ActionLoginBlocked:    SeverityCritical,
	ActionLogout:          SeverityDebug,
	ActionRegister:        SeverityInfo,
	ActionFileUpload:      SeverityInfo,
	ActionFileDelete:      SeverityWarning,
	ActionFileDownload:    SeverityDebug,
	ActionVideoComment:    SeverityInfo,
	ActionVideoLike:       SeverityDebug,
	ActionVideoCollect:    SeverityDebug,
	ActionVideoPlay:       SeverityDebug,
	ActionDanmakuSend:     SeverityInfo,
	ActionDanmakuDelete:   SeverityWarning,
	ActionForumPost:       SeverityInfo,
	ActionForumReply:      SeverityInfo,
	ActionForumLike:       SeverityDebug,
	ActionProfileUpdate:   SeverityInfo,
	ActionPasswordChange:  SeverityWarning,
	ActionAvatarUpload:    SeverityInfo,
	ActionAdminAccess:     SeverityWarning,
	ActionAdminDeleteUser: SeverityCritical,
	ActionAdminDeleteFile: SeverityCritical,
	ActionAdminConfig:     SeverityCritical,
}
