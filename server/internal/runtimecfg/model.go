package runtimecfg

import "time"

// RuntimeConfig stores a dynamic configuration entry in the database.
// SET configs have exactly one row per Target (Kind="set").
// ADD configs can have multiple rows per Target (Kind="add"), each with a unique ID.
type RuntimeConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Target    string    `gorm:"index;size:64;not null" json:"target"`
	Kind      string    `gorm:"size:8;not null" json:"kind"`
	AttrsJSON string    `gorm:"type:text;not null" json:"attrs_json"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RuntimeConfig) TableName() string {
	return "runtime_configs"
}
