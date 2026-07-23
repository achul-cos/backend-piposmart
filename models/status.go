package models

import "time"

type CustomerStatus struct {
	BaseModel
	CustomerID        uint64     `json:"customer_id" gorm:"uniqueIndex;constraint:OnDelete:CASCADE;"`
	CallStatus        *string    `json:"call_status" gorm:"type:varchar(255)"`
	ChatStatus        *string    `json:"chat_status" gorm:"type:varchar(255)"`
	TanggalFu         *time.Time `json:"tanggal_fu"`
	TotalFu           *int       `json:"total_fu" gorm:"default:0"`
	Noted             *string    `json:"noted"`
	Remarks           *string    `json:"remarks"`
	Score             *int       `json:"score" gorm:"default:0"`
	StatusAkun        *string    `json:"status_akun" gorm:"type:varchar(255)"`
	FinalisasiClosing *string    `json:"finalisasi_closing" gorm:"type:varchar(255)"`
	Nominal           *float64   `json:"nominal" gorm:"default:0"`
}
