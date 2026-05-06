package models

import "time"

// User represents the user_info table.
type User struct {
	ID        int64     `json:"-" gorm:"primaryKey;autoIncrement;column:id"`
	UUID      string    `json:"id" gorm:"column:uuid;type:char(36);uniqueIndex;not null;default:''"`
	Email     string    `json:"email" gorm:"column:email;type:varchar(255);uniqueIndex;not null;default:''"`
	Password  string    `json:"-" gorm:"column:password;type:varchar(1000);not null;default:''"`
	Status    int8      `json:"status" gorm:"column:status;type:tinyint;not null;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"column:ctime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:mtime;autoUpdateTime"`
}

// TableName overrides the default table name.
func (User) TableName() string { return "user_info" }
