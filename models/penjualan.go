package models

import (
	"time"
)

type Penjualan struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Tanggal          time.Time  `json:"tanggal"`            // Menggantikan tanggalInput operasional
	PicTeam          string     `gorm:"type:varchar(100)" json:"pic_team"` // Maps ke picNasabah
	KodeOwner        string     `gorm:"type:varchar(50)" json:"kode_owner"`
	NamaOwner        string     `gorm:"type:varchar(100)" json:"nama_owner"`
	NamaBrand        string     `gorm:"type:varchar(100)" json:"nama_brand"` // Maps ke brand
	NamaOutlet       string     `gorm:"type:varchar(100)" json:"nama_outlet"` // Maps ke namaOutlet
	HpOwner          string     `gorm:"type:varchar(50)" json:"hp_owner"`
	SumberData       string     `gorm:"type:varchar(150);default:'Kelolaan'" json:"sumber_data"`
	Status           string     `gorm:"type:varchar(50);default:'Closing'" json:"status"` // Closing, Potensi, Negoisasi, dll
	Membership       string     `gorm:"type:varchar(50);default:'Aktivasi'" json:"membership"`
	TargetPaket      string     `gorm:"type:varchar(100)" json:"target_paket"`
	TargetNominal    float64    `json:"target_nominal"`
	PaketAktual      string     `gorm:"type:varchar(100)" json:"paket_aktual"`
	TotalPenjualan   float64    `json:"total_penjualan"`    // Nominal Aktual Realisasi deal lapangan
	TanggalTraining  *time.Time `json:"tanggal_training"`
	StatusTraining   string     `gorm:"type:varchar(50);default:'Belum Training'" json:"status_training"`
	TanggalRealisasi *time.Time `json:"tanggal_realisasi"`
	BuktiTransfer    string     `gorm:"type:text" json:"bukti_transfer"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// 🌟 MEMAKSA TABEL TERPISAH YANG BERSIH AGAR TIDAK BENTROK DENGAN KOLOM LAMA YANG RUSAK
func (Penjualan) TableName() string {
	return "pipo_penjualan"
}