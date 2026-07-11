package models

import "time"

type Report struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Tanggal       string    `gorm:"type:varchar(10);column:tanggal" json:"tanggal"`
	Bulan         string    `gorm:"type:varchar(30);column:bulan" json:"bulan"`
	Keterangan    string    `gorm:"type:text;column:keterangan" json:"keterangan"`
	
	// 🌟 FIX UTAMA: Tambahkan field Pic ini ke dalam struct model Report
	Pic           string    `gorm:"type:varchar(50);column:pic" json:"pic"`

	ResponCall    int       `gorm:"type:int;column:respon_call" json:"responCall"`
	ResponChat    int       `gorm:"type:int;column:respon_chat" json:"responChat"`
	ResponMeeting int       `gorm:"type:int;column:respon_meeting" json:"responMeeting"`
	ResponVisit   int       `gorm:"type:int;column:respon_visit" json:"responVisit"`
	NoResponCount int       `gorm:"type:int;column:no_respon_count" json:"noResponCount"`
	TotalRespon   int       `gorm:"type:int;column:total_respon" json:"totalRespon"`
	TotalNoRespon int       `gorm:"type:int;column:total_no_respon" json:"totalNoRespon"`
	GrandTotal    int       `gorm:"type:int;column:grand_total" json:"grandTotal"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}