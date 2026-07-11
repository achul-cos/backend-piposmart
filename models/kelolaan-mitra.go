package models

import "time"

type KelolaanMitra struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TanggalInput    string    `gorm:"type:varchar(10);column:tanggal_input" json:"tanggalInput"`
	NamaMitra       string    `gorm:"type:varchar(150);column:nama_mitra" json:"namaMitra"`
	
	// 🌟 SINKRONISASI DATABASE: Kolom mapping string penampung PIC penelusuran crm
	PicNasabah      string    `gorm:"type:varchar(50);column:pic_nasabah" json:"picNasabah"`
	
	BrandUtama      string    `gorm:"type:varchar(150);column:brand_utama" json:"brandUtama"`
	KodeOwnerUtama  string    `gorm:"type:varchar(50);column:kode_owner_utama" json:"kodeOwnerUtama"`
	KategoriMitra   string    `gorm:"type:varchar(100);column:kategori_mitra" json:"kategoriMitra"`
	PaketLangganan  string    `gorm:"type:varchar(100);column:paket_langganan" json:"paketLangganan"`
	StatusLangganan string    `gorm:"type:varchar(50);column:status_langganan" json:"statusLangganan"`
	BuktiFu         string    `gorm:"type:text;column:bukti_fu" json:"buktiFu"`
	StatusInput     string    `gorm:"type:varchar(30);column:status_input" json:"statusInput"`
	NominalKomisi   float64   `gorm:"type:numeric;column:nominal_komisi" json:"nominal_komisi"`
	StatusKomisi    string    `gorm:"type:varchar(30);column:status_komisi" json:"status_komisi"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}