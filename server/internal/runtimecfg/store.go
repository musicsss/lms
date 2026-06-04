package runtimecfg

import (
	"encoding/json"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) LoadAll() ([]RuntimeConfig, error) {
	var rows []RuntimeConfig
	err := s.db.Order("id ASC").Find(&rows).Error
	return rows, err
}

func (s *Store) ListByTarget(target string) ([]RuntimeConfig, error) {
	var rows []RuntimeConfig
	err := s.db.Where("target = ?", target).Order("id ASC").Find(&rows).Error
	return rows, err
}

// UpsertSet creates or updates the single SET row for a target.
func (s *Store) UpsertSet(target string, attrs map[string]string) error {
	raw, _ := json.Marshal(attrs)

	var existing RuntimeConfig
	err := s.db.Where("target = ? AND kind = ?", target, "set").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(&RuntimeConfig{
			Target:    target,
			Kind:      "set",
			AttrsJSON: string(raw),
		}).Error
	}
	if err != nil {
		return err
	}
	return s.db.Model(&existing).Update("attrs_json", string(raw)).Error
}

// CreateAdd inserts a new ADD row.
func (s *Store) CreateAdd(target string, attrs map[string]string) (uint, error) {
	raw, _ := json.Marshal(attrs)
	r := &RuntimeConfig{
		Target:    target,
		Kind:      "add",
		AttrsJSON: string(raw),
	}
	if err := s.db.Create(r).Error; err != nil {
		return 0, err
	}
	return r.ID, nil
}

// UpdateAdd updates an existing ADD row by ID.
func (s *Store) UpdateAdd(id uint, attrs map[string]string) error {
	raw, _ := json.Marshal(attrs)
	return s.db.Model(&RuntimeConfig{}).Where("id = ? AND kind = ?", id, "add").
		Update("attrs_json", string(raw)).Error
}

// DeleteAdd deletes an ADD row by ID.
func (s *Store) DeleteAdd(id uint) error {
	return s.db.Where("id = ? AND kind = ?", id, "add").Delete(&RuntimeConfig{}).Error
}

// HasSet returns true if a SET row exists for the given target.
func (s *Store) HasSet(target string) bool {
	var count int64
	s.db.Model(&RuntimeConfig{}).Where("target = ? AND kind = ?", target, "set").Count(&count)
	return count > 0
}
