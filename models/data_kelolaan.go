package models

import "time"

// DataKelolaan adalah struct tunggal GORM untuk memetakan tabel data_kelolaans di MySQL Laragon
type DataKelolaan struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TanggalFu      string    `json:"tanggalFu"`
	TanggalReal    string    `json:"tanggalReal"` // Create Date Project rill dari Excel
	Bulan          string    `json:"bulan"`
	StatusAkun     string    `json:"statusAkun"`
	Pic            string    `gorm:"column:pic" json:"pic"`
	KodeOwner      string    `json:"kodeOwner"`
	NamaOwner      string    `json:"namaOwner"`
	Brand          string    `json:"brand"` // Berfungsi sebagai identifier brand utama
	Outlet         string    `json:"outlet"`
	HpOwner        string    `json:"hpOwner"`
	HpOutlet       string    `json:"hpOutlet"`
	ExpiredDate    string    `json:"expiredDate"`
	TotalTransaksi string    `json:"totalTransaksi"`
	Score          string    `json:"score"`
	CallStatus     string    `json:"callStatus"`
	ChatStatus     string    `json:"chatStatus"`
	Validitas      string    `json:"validitas"`
	Remaks         string    `json:"remaks"`
	Sumber         string    `json:"sumber"`
	Noted          string    `json:"noted"`
	CreatedAt      time.Time `json:"createdAt"`
}
