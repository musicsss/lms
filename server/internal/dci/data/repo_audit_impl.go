package data

import (
	"time"

	"github.com/lms/server/internal/model"
	"gorm.io/gorm"
)

type auditLogRepoImpl struct {
	db *gorm.DB
}

func NewAuditLogRepo(db *gorm.DB) AuditLogRepo {
	return &auditLogRepoImpl{db: db}
}

func (r *auditLogRepoImpl) dbFrom(d DB) *gorm.DB {
	if u, ok := d.(*gorm.DB); ok {
		return u
	}
	return d.(*gorm.DB)
}

func (r *auditLogRepoImpl) Create(d DB, log *model.AuditLog) error {
	return r.dbFrom(d).Create(log).Error
}

func (r *auditLogRepoImpl) FindByUserID(d DB, userID uint, page, pageSize int) ([]model.AuditLog, int64, error) {
	db := r.dbFrom(d)
	var logs []model.AuditLog
	var total int64

	query := db.Model(&model.AuditLog{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepoImpl) ListAll(d DB, page, pageSize int, severity, action, sort, order string, userID uint) ([]model.AuditLog, int64, error) {
	db := r.dbFrom(d)
	var logs []model.AuditLog
	var total int64

	query := db.Model(&model.AuditLog{})
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Valid sort columns (whitelist to prevent SQL injection)
	validSorts := map[string]string{
		"created_at": "created_at",
		"action":     "action",
		"severity":   "severity",
		"user_id":    "user_id",
		"ip":         "ip",
		"success":    "success",
	}
	sortCol := "created_at"
	if col, ok := validSorts[sort]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if order == "asc" {
		sortDir = "ASC"
	}

	offset := (page - 1) * pageSize
	err := query.Order(sortCol + " " + sortDir).Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepoImpl) DeleteBefore(d DB, before time.Time) error {
	return r.dbFrom(d).Where("created_at < ?", before).Delete(&model.AuditLog{}).Error
}
