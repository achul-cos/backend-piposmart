package models

type Target struct {
	ID                 uint    `gorm:"primaryKey" json:"id"`
	PicTeam            string  `gorm:"column:pic_team" json:"pic_team"`
	PaketLangganan     string  `gorm:"column:paket_langganan" json:"paket_langganan"`
	TargetNasabah      int     `gorm:"column:target_nasabah" json:"target_nasabah"`
	RealisasiNasabah   int     `gorm:"column:realisasi_nasabah" json:"realisasi_nasabah"`
	TargetPenjualan    float64 `gorm:"column:target_penjualan" json:"target_penjualan"`
	RealisasiPenjualan float64 `gorm:"column:realisasi_penjualan" json:"realisasi_penjualan"`
	SelisihPenjualan   float64 `gorm:"column:selisih_penjualan" json:"selisih_penjualan"`
	PeriodeBulan       string  `gorm:"column:periode_bulan" json:"periode_bulan"`
}