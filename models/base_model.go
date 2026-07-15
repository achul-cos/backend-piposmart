package models

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
