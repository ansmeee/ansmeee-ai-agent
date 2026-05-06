package models

import "time"

// ModelConfig represents the ai_model_config table.
type ModelConfig struct {
	ID        int64     `json:"-" gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `json:"user_id" gorm:"column:user_id;not null"`
	Model     string    `json:"model" gorm:"column:model;type:varchar(255);not null;default:''"`
	BaseURL   string    `json:"base_url" gorm:"column:base_url;type:varchar(255);not null;default:''"`
	Token     string    `json:"token" gorm:"column:token;type:varchar(255);not null;default:''"`
	CreatedAt time.Time `json:"created_at" gorm:"column:ctime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:mtime;autoUpdateTime"`
}

// TableName overrides the default table name.
func (ModelConfig) TableName() string { return "ai_model_config" }
