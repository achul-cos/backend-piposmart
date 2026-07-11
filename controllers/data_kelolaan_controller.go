package controllers

import (
	"net/http"
	"piposmart-backend/config"
	"piposmart-backend/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

// Struct Input JSON untuk menampung data simpan baru dari Next.js
type CreateDataKelolaanInput struct {
	TanggalFu      string `json:"tanggalFu" binding:"required"`
	TanggalReal    string `json:"tanggalReal"`
	Bulan          string `json:"bulan" binding:"required"`
	StatusAkun     string `json:"statusAkun" binding:"required"`
	Pic            string `json:"pic"`
	KodeOwner      string `json:"kodeOwner"`
	NamaOwner      string `json:"namaOwner" binding:"required"`
	Brand          string `json:"brand" binding:"required"`
	Outlet         string `json:"outlet"`
	HpOwner        string `json:"hpOwner" binding:"required"`
	HpOutlet       string `json:"hpOutlet"`
	ExpiredDate    string `json:"expiredDate"`
	TotalTransaksi string `json:"totalTransaksi"`
	Score          string `json:"score"`
	CallStatus     string `json:"callStatus"`
	ChatStatus     string `json:"chatStatus"`
	Validitas      string `json:"validitas"`
	Remaks         string `json:"remaks"`
	Sumber         string `json:"sumber"`
	Noted          string `json:"noted"`
}

// Struct Input JSON khusus edit tanpa flag 'required' agar fleksibel
type UpdateDataKelolaanInput struct {
	TanggalFu      string `json:"tanggalFu"`
	TanggalReal    string `json:"tanggalReal"`
	Bulan          string `json:"bulan"`
	StatusAkun     string `json:"statusAkun"`
	Pic            string `json:"pic"`
	KodeOwner      string `json:"kodeOwner"`
	NamaOwner      string `json:"namaOwner"`
	Brand          string `json:"brand"`
	Outlet         string `json:"outlet"`
	HpOwner        string `json:"hpOwner"`
	HpOutlet       string `json:"hpOutlet"`
	ExpiredDate    string `json:"expiredDate"`
	TotalTransaksi string `json:"totalTransaksi"`
	Score          string `json:"score"`
	CallStatus     string `json:"callStatus"`
	ChatStatus     string `json:"chatStatus"`
	Validitas      string `json:"validitas"`
	Remaks         string `json:"remaks"`
	Sumber         string `json:"sumber"`
	Noted          string `json:"noted"`
}

// 🌟 STRUCT BARU: Data Transfer Object (DTO) Ringkas untuk Payload Tabel Utama Harian
type DataKelolaanSummary struct {
	ID             uint   `json:"id"`
	TanggalFu      string `json:"tanggalFu"`
	StatusAkun     string `json:"statusAkun"`
	NamaOwner      string `json:"namaOwner"`
	Brand          string `json:"brand"`
	Outlet         string `json:"outlet"`
	TotalTransaksi string `json:"totalTransaksi"`
	Pic            string `json:"pic"`
	Bulan          string `json:"bulan"`
}

// CreateDataKelolaan menyimpan record manual dari form UI Next.js
func CreateDataKelolaan(c *gin.Context) {
	var input CreateDataKelolaanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format input salah: " + err.Error()})
		return
	}

	dataBaru := models.DataKelolaan{
		TanggalFu:      input.TanggalFu,
		TanggalReal:    input.TanggalReal,
		Bulan:          input.Bulan,
		StatusAkun:     input.StatusAkun,
		Pic:            input.Pic,
		KodeOwner:      input.KodeOwner,
		NamaOwner:      input.NamaOwner,
		Brand:          input.Brand,
		Outlet:         input.Outlet,
		HpOwner:        input.HpOwner,
		HpOutlet:       input.HpOutlet,
		ExpiredDate:    input.ExpiredDate,
		TotalTransaksi: input.TotalTransaksi,
		Score:          input.Score,
		CallStatus:     input.CallStatus,
		ChatStatus:     input.ChatStatus,
		Validitas:      input.Validitas,
		Remaks:         input.Remaks,
		Sumber:         input.Sumber,
		Noted:          input.Noted,
		CreatedAt:      time.Now(),
	}

	if err := config.DB.Create(&dataBaru).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ke database: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Data Kelolaan berhasil disimpan!", "data": dataBaru})
}

// 🌟 OPTIMASI UTAMA: Mengambil list ringkas (Summary DTO) untuk menghemat bandwidth & mempercepat render FE
func GetDataKelolaan(c *gin.Context) {
	var summaryList []DataKelolaanSummary

	// GORM Select pintar mengekstrak field esensial saja dari baris data_kelolaans
	err := config.DB.Model(&models.DataKelolaan{}).
		Order("id desc").
		Select("id", "tanggal_fu", "status_akun", "nama_owner", "brand", "outlet", "total_transaksi", "pic", "bulan").
		Scan(&summaryList).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kelolaan ringkas: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, summaryList)
}

// GetDetailDataKelolaan menarik data finance biner lengkap berdasarkan ID baris tabel
func GetDetailDataKelolaan(c *gin.Context) {
	id := c.Param("id")
	var dataKelolaan models.DataKelolaan

	if err := config.DB.First(&dataKelolaan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Detail data tidak ditemukan!"})
		return
	}
	c.JSON(http.StatusOK, dataKelolaan)
}

// UpdateDataKelolaan memperbarui record data kelolaan mitra via form modal edit
func UpdateDataKelolaan(c *gin.Context) {
	id := c.Param("id")
	var dataKelolaan models.DataKelolaan

	if err := config.DB.First(&dataKelolaan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Kelolaan tidak ditemukan!"})
		return
	}

	var input UpdateDataKelolaanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format input edit salah: " + err.Error()})
		return
	}

	updatedFields := models.DataKelolaan{
		TanggalFu:      input.TanggalFu,
		TanggalReal:    input.TanggalReal,
		Bulan:          input.Bulan,
		StatusAkun:     input.StatusAkun,
		Pic:            input.Pic,
		KodeOwner:      input.KodeOwner,
		NamaOwner:      input.NamaOwner,
		Brand:          input.Brand,
		Outlet:         input.Outlet,
		HpOwner:        input.HpOwner,
		HpOutlet:       input.HpOutlet,
		ExpiredDate:    input.ExpiredDate,
		TotalTransaksi: input.TotalTransaksi,
		Score:          input.Score,
		CallStatus:     input.CallStatus,
		ChatStatus:     input.ChatStatus,
		Validitas:      input.Validitas,
		Remaks:         input.Remaks,
		Sumber:         input.Sumber,
		Noted:          input.Noted,
	}

	if err := config.DB.Model(&dataKelolaan).Updates(updatedFields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data Kelolaan berhasil diperbarui!", "data": dataKelolaan})
}

// DeleteDataKelolaan menghapus record kelolaan tunggal (Single Row Hard Delete)
func DeleteDataKelolaan(c *gin.Context) {
	id := c.Param("id")
	var dataKelolaan models.DataKelolaan

	if err := config.DB.First(&dataKelolaan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Kelolaan tidak ditemukan!"})
		return
	}

	if err := config.DB.Delete(&dataKelolaan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data Kelolaan berhasil dihapus!"})
}

// DeleteAllDataKelolaan membersihkan total tabel kelolaan (Truncate Command)
func DeleteAllDataKelolaan(c *gin.Context) {
	if err := config.DB.Exec("DELETE FROM data_kelolaans").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengosongkan tabel: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Seluruh basis data CRM berhasil dibersihkan!"})
}

// BulkCreateDataKelolaan memproses mass-upload Excel per 1000 baris dengan fitur Upsert anti-duplikat
func BulkCreateDataKelolaan(c *gin.Context) {
	var inputs []CreateDataKelolaanInput
	if err := c.ShouldBindJSON(&inputs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data bulk Excel salah: " + err.Error()})
		return
	}

	var dataBaruList []models.DataKelolaan
	waktuSekarang := time.Now()

	for _, input := range inputs {
		dataBaru := models.DataKelolaan{
			TanggalFu:      input.TanggalFu,
			TanggalReal:    input.TanggalReal,
			Bulan:          input.Bulan,
			StatusAkun:     input.StatusAkun,
			Pic:            input.Pic,
			KodeOwner:      input.KodeOwner,
			NamaOwner:      input.NamaOwner,
			Brand:          input.Brand,
			Outlet:         input.Outlet,
			HpOwner:        input.HpOwner,
			HpOutlet:       input.HpOutlet,
			ExpiredDate:    input.ExpiredDate,
			TotalTransaksi: input.TotalTransaksi,
			Score:          input.Score,
			CallStatus:     input.CallStatus,
			ChatStatus:     input.ChatStatus,
			Validitas:      input.Validitas,
			Remaks:         input.Remaks,
			Sumber:         input.Sumber,
			Noted:          input.Noted,
			CreatedAt:      waktuSekarang,
		}
		dataBaruList = append(dataBaruList, dataBaru)
	}

	if len(dataBaruList) > 0 {
		if err := config.DB.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&dataBaruList, 1000).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data massal: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Bulk Data Kelolaan dari Excel berhasil disimpan tanpa duplikasi!",
		"count":   len(dataBaruList),
	})
}

// ExportDataKelolaan bypass placeholder
func ExportDataKelolaan(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Fitur export menyusul"})
}
