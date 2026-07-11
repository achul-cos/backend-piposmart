package models

import "time"

type SopActivity struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Tipe       string    `gorm:"type:varchar(50);not null" json:"tipe"`   // "potensi", "tidak_potensi", "call", "chat", "reason"
	Label      string    `gorm:"type:varchar(100);not null" json:"label"` // Nama status / Judul Aktivitas
	Keterangan string    `gorm:"type:text" json:"keterangan"`             // Deskripsi atau rincian teks SOP
	Urutan     int       `gorm:"type:int;default:0" json:"urutan"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}