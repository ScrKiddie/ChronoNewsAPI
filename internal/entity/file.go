package entity

import (
	"time"
)

type File struct {
	ID             int32 `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Name           string  `gorm:"column:name;type:varchar(255);index"`
	Status         string  `gorm:"column:status;type:file_status;default:'pending';index"`
	FailedAttempts int     `gorm:"column:failed_attempts;default:0"`
	LastError      *string `gorm:"column:last_error;type:varchar(255)"`
	UsedByPostID   *int32  `gorm:"column:used_by_post_id;index"`
	Post           *Post   `gorm:"foreignKey:UsedByPostID;constraint:OnDelete:SET NULL"`
}

func (File) TableName() string {
	return "file" // Singular (konsisten dengan 'user')
}

type DeadLetterQueue struct {
	ID           int32 `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	FileID       int32  `gorm:"column:file_id"`
	File         File   `gorm:"foreignKey:FileID"`
	ErrorMessage string `gorm:"column:error_message;type:varchar(255)"`
}

func (DeadLetterQueue) TableName() string {
	return "dead_letter_queue"
}
