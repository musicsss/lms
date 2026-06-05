package runtimecfg

import (
	"encoding/json"
	"fmt"
	"strings"

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
	err := s.db.Where("target = ? AND kind = ?", target, KindSet).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(&RuntimeConfig{
			Target:    target,
			Kind:      KindSet,
			AttrsJSON: string(raw),
		}).Error
	}
	if err != nil {
		return err
	}
	return s.db.Model(&existing).Update("attrs_json", string(raw)).Error
}

// dedupKey computes a semantic uniqueness key from attrs for a given target.
// Returns "" if no dedup is needed for this target.
//
// For LoginFail, dedup keys match loginprotect.Range constants ("ALL_USER", "SINGLE_USER:<val>", "IP:<val>").
func dedupKey(target string, attrs map[string]string) string {
	switch target {
	case TargetLoginFail:
		r := attrs[FieldRange]
		if r == "" {
			return ""
		}
		if idx := strings.Index(r, ":"); idx >= 0 {
			base := r[:idx]
			val := r[idx+1:]
			if (base == "SINGLE_USER" || base == "IP") && val != "" {
				return base + ":" + val
			}
			return r
		}
		return "ALL_USER"
	case TargetCORS:
		o := attrs[FieldOrigin]
		if o == "" {
			return ""
		}
		return "ORIGIN:" + o
	default:
		return ""
	}
}

// CreateAdd inserts a new ADD row. Returns error if a semantically duplicate row exists.
func (s *Store) CreateAdd(target string, attrs map[string]string) (uint, error) {
	raw, _ := json.Marshal(attrs)

	// semantic uniqueness check
	key := dedupKey(target, attrs)
	if key != "" {
		rows, _ := s.ListByTarget(target)
		for _, row := range rows {
			if row.Kind != KindAdd {
				continue
			}
			var existingAttrs map[string]string
			json.Unmarshal([]byte(row.AttrsJSON), &existingAttrs)
			ek := dedupKey(target, existingAttrs)
			if ek != "" && ek == key {
				return 0, fmt.Errorf("duplicate scope: a config with the same scope already exists (ID=%d)", row.ID)
			}
		}
	}

	r := &RuntimeConfig{
		Target:    target,
		Kind:      KindAdd,
		AttrsJSON: string(raw),
	}
	if err := s.db.Create(r).Error; err != nil {
		return 0, err
	}
	return r.ID, nil
}

// UpdateAdd updates an existing ADD row by ID. Checks semantic uniqueness against other rows.
func (s *Store) UpdateAdd(target string, id uint, attrs map[string]string) error {
	raw, _ := json.Marshal(attrs)

	// semantic uniqueness check (exclude self)
	key := dedupKey(target, attrs)
	if key != "" {
		rows, _ := s.ListByTarget(target)
		for _, row := range rows {
			if row.Kind != KindAdd || row.ID == id {
				continue
			}
			var existingAttrs map[string]string
			json.Unmarshal([]byte(row.AttrsJSON), &existingAttrs)
			ek := dedupKey(target, existingAttrs)
			if ek != "" && ek == key {
				return fmt.Errorf("duplicate scope: a config with the same scope already exists (ID=%d)", row.ID)
			}
		}
	}

	return s.db.Model(&RuntimeConfig{}).Where("id = ? AND kind = ?", id, KindAdd).
		Update("attrs_json", string(raw)).Error
}

// DeleteAdd deletes an ADD row by ID.
func (s *Store) DeleteAdd(id uint) error {
	return s.db.Where("id = ? AND kind = ?", id, KindAdd).Delete(&RuntimeConfig{}).Error
}

// HasSet returns true if a SET row exists for the given target.
func (s *Store) HasSet(target string) bool {
	var count int64
	s.db.Model(&RuntimeConfig{}).Where("target = ? AND kind = ?", target, KindSet).Count(&count)
	return count > 0
}
