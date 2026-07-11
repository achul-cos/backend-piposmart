package models

import "time"

type CallChat struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    TanggalFu   string    `gorm:"type:varchar(10);column:tanggal_fu" json:"tanggalFu"`
    Bulan       string    `gorm:"type:varchar(30);column:bulan" json:"bulan"`
    TanggalOrig string    `gorm:"type:varchar(10);column:tanggal_dibagikan" json:"tanggalDibagikan"`
    StatusAkun  string    `gorm:"type:varchar(50);column:status_akun" json:"statusAkun"`
    
    // FIX UTAMA: Menambahkan kolom pic agar dikenali oleh controller Go
    Pic         string    `gorm:"type:varchar(50);column:pic" json:"pic"`

    KodeBaris   int       `gorm:"type:int;column:kode_baris" json:"kodeBaris"`
    KodeOwner   int       `gorm:"type:int;column:kode_owner" json:"kodeOwner"`
    NamaOwner   string    `gorm:"type:varchar(100);column:nama_owner" json:"namaOwner"`
    Brand       string    `gorm:"type:varchar(100);column:brand" json:"brand"`
    Outlet      string    `gorm:"type:varchar(100);column:outlet" json:"outlet"`
    
    // 🌟 SELESAI: Tag ';unique' telah dihapus agar nomor handphone bisa didaftarkan ulang untuk cabang/brand berbeda
    HpOwner     string    `gorm:"type:varchar(30);column:hp_owner" json:"hpOwner"`
    
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}