package models

import "time"

// Agent represents the ai_agent table.
type Agent struct {
	ID          int64     `json:"-" gorm:"primaryKey;autoIncrement;column:id"`
	UUID        string    `json:"id" gorm:"column:uuid;type:char(36);uniqueIndex;not null;default:''"`
	UserID      int64     `json:"user_id" gorm:"column:user_id;not null;default:0;index"`
	Title       string    `json:"title" gorm:"column:title;type:varchar(255);not null;default:''"`
	Description string    `json:"description" gorm:"column:intro;type:varchar(1000);not null;default:''"`
	Prompt      string    `json:"prompt" gorm:"column:prompt;type:text"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:mtime;autoUpdateTime"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:ctime;autoCreateTime"`
}

// TableName overrides the default table name.
func (Agent) TableName() string { return "ai_agent" }
