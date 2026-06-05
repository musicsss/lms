package data

import (
	"time"

	"github.com/lms/server/internal/model"
)

type AuditLogRepo interface {
	Create(db DB, log *model.AuditLog) error
	FindByUserID(db DB, userID uint, page, pageSize int) ([]model.AuditLog, int64, error)
	ListAll(db DB, page, pageSize int, severity, action, sort, order string, userID uint) ([]model.AuditLog, int64, error)
	DeleteBefore(db DB, before time.Time) error
}
