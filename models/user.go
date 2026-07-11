package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"type:varchar(50);unique;column:username" json:"username"` // Diisi email seperti satria@piposmart.com
	Password  string    `gorm:"type:varchar(255);column:password" json:"-"`              // Password (misal: admin123)
	Nama      string    `gorm:"type:varchar(100);column:nama" json:"nama"`               // Nama asli (Satria, Lydia, Laura)
	Role      string    `gorm:"type:varchar(20);column:role" json:"role"`                 // "Admin" atau "Sales"
	CreatedAt time.Time `json:"createdAt"`
}