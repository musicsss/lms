package audit

import (
	"log/slog"

	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

// RecordContext 记录一条审计日志（同步写入，极轻量）。
type RecordContext struct {
	db         *gorm.DB
	auditRepo  data.AuditLogRepo
	UserID     uint
	Action     string
	Resource   string
	ResourceID uint
	Detail     string
	IP         string
	Success    bool
}

func NewRecordContext(db *gorm.DB, auditRepo data.AuditLogRepo, userID uint, action, resource string, resourceID uint, detail, ip string, success bool) *RecordContext {
	return &RecordContext{
		db:         db,
		auditRepo:  auditRepo,
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     detail,
		IP:         ip,
		Success:    success,
	}
}

// Execute 同步记录审计日志。
func (c *RecordContext) Execute() {
	severity, ok := model.ActionSeverityMap[c.Action]
	if !ok {
		severity = model.SeverityInfo
	}

	username := ""
	if c.UserID != 0 {
		var u model.User
		if err := c.db.First(&u, c.UserID).Error; err == nil {
			username = u.Username
		}
	}

	entry := &model.AuditLog{
		UserID:     c.UserID,
		Username:   username,
		Action:     c.Action,
		Severity:   severity,
		Resource:   c.Resource,
		ResourceID: c.ResourceID,
		Detail:     c.Detail,
		IP:         c.IP,
		Success:    c.Success,
	}

	if err := c.auditRepo.Create(c.db, entry); err != nil {
		slog.Warn("audit: write failed", "action", c.Action, "err", err)
	}
}
