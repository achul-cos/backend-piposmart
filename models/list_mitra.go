package models

import "time"

type ListMitra struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	KategoriMitra        string    `gorm:"type:varchar(100);column:kategori_mitra" json:"kategori"`
	
	// 🌟 FIX MODEL: Menyediakan field mapping penampung string PIC ke database SQL
	PicNasabah           string    `gorm:"type:varchar(50);column:pic_nasabah" json:"picNasabah"`
	
	BulanTerdaftar       string    `gorm:"type:varchar(30);column:bulan_terdaftar" json:"bulanTerdaftar"`
	Tahun                int       `gorm:"type:int;column:tahun" json:"tahun"`
	KodeOwner            string    `gorm:"type:varchar(50);column:kode_owner" json:"kodeOwner"`
	OwnerMitra           string    `gorm:"type:varchar(150);column:owner_mitra" json:"owner"`
	NamaBrand            string    `gorm:"type:varchar(150);column:nama_brand" json:"brand"`
	KategoriMitraSub     string    `gorm:"type:varchar(100);column:kategori_mitra_sub" json:"kategoriSub"`
	TotalAkuisisiReferal int       `gorm:"type:int;column:total_akuisisi_referal" json:"totalAkuisisiReferal"`
	TotalReferral        int       `gorm:"type:int;column:total_referral" json:"totalReferral"`
	Telp                 string    `gorm:"type:varchar(50);column:telp" json:"telp"`
	Rekening             string    `gorm:"type:varchar(100);column:rekening" json:"rekening"`
	Wilayah              string    `gorm:"type:varchar(100);column:wilayah" json:"wilayah"`
	Alamat               string    `gorm:"type:text;column:alamat" json:"alamat"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}