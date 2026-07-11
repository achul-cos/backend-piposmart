package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"github.com/gin-gonic/gin"
	"piposmart-backend/config"
	"piposmart-backend/models"
)

// Struct Input DTO penampung payload form Next.js saat Create
type CreateListMitraInput struct {
	Kategori             string      `json:"kategori" binding:"required"`
	PicNasabah           string      `json:"picNasabah"`
	BulanTerdaftar       string      `json:"bulanTerdaftar"`
	Tahun                interface{} `json:"tahun"`
	KodeOwner            string      `json:"kodeOwner"`
	Owner                string      `json:"owner" binding:"required"`
	Brand                string      `json:"brand" binding:"required"`
	KategoriSub          string      `json:"kategoriSub"`
	TotalAkuisisiReferal interface{} `json:"totalAkuisisiReferal"`
	TotalReferral        interface{} `json:"totalReferral"`
	Telp                 string      `json:"telp" binding:"required"`
	Rekening             string      `json:"rekening"`
	Wilayah              string      `json:"wilayah"`
	Alamat               string      `json:"alamat"`
}

// 🌟 Struct Input DTO khusus Update (Menghapus binding required agar fleksibel saat inline/partial edit)
type UpdateListMitraInput struct {
	Kategori             string      `json:"kategori"`
	PicNasabah           string      `json:"picNasabah"`
	BulanTerdaftar       string      `json:"bulanTerdaftar"`
	Tahun                interface{} `json:"tahun"`
	KodeOwner            string      `json:"kodeOwner"`
	Owner                string      `json:"owner"`
	Brand                string      `json:"brand"`
	KategoriSub          string      `json:"kategoriSub"`
	TotalAkuisisiReferal interface{} `json:"totalAkuisisiReferal"`
	TotalReferral        interface{} `json:"totalReferral"`
	Telp                 string      `json:"telp"`
	Rekening             string      `json:"rekening"`
	Wilayah              string      `json:"wilayah"`
	Alamat               string      `json:"alamat"`
}

// Helper internal konversi interface{} ke int secara aman
func parseMitraInterfaceToInt(val interface{}) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if v == "" {
			return 0
		}
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// GET /api/list-mitra
func GetListMitra(c *gin.Context) {
	var list []models.ListMitra
	if err := config.DB.Order("id desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengambil master list mitra"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// POST /api/list-mitra/create
func CreateListMitra(c *gin.Context) {
	var input CreateListMitraInput
	
	if err := c.ShouldBindJSON(&input); err != nil {
		fmt.Println("❌ ERROR BINDING JSON LIST MITRA:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi input gagal: " + err.Error()})
		return
	}

	valTahun := parseMitraInterfaceToInt(input.Tahun)
	if valTahun == 0 {
		valTahun = time.Now().Year()
	}

	mitraData := models.ListMitra{
		KategoriMitra:        input.Kategori,
		PicNasabah:           input.PicNasabah,
		BulanTerdaftar:       input.BulanTerdaftar,
		Tahun:                valTahun,
		KodeOwner:            input.KodeOwner,
		OwnerMitra:           input.Owner,
		NamaBrand:            input.Brand,
		KategoriMitraSub:     input.KategoriSub,
		TotalAkuisisiReferal: parseMitraInterfaceToInt(input.TotalAkuisisiReferal),
		TotalReferral:        parseMitraInterfaceToInt(input.TotalReferral),
		Telp:                 input.Telp,
		Rekening:             input.Rekening,
		Wilayah:              input.Wilayah,
		Alamat:               input.Alamat,
		CreatedAt:            time.Now(),
	}

	if err := config.DB.Create(&mitraData).Error; err != nil {
		fmt.Println("❌ GAGAL SIMPAN DATABASE LIST MITRA:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan data ke database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Sakti! Data Master Mitra Berhasil Disimpan Permanen!",
		"data":    mitraData,
	})
}

// ✏️ PUT /api/list-mitra/update/:id
func UpdateListMitra(c *gin.Context) {
	id := c.Param("id")
	var record models.ListMitra

	// 1. Periksa ketersediaan data partner di database
	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data master mitra tidak ditemukan!"})
		return
	}

	// 2. Bind payload request perubahan JSON dari frontend
	var input UpdateListMitraInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Struktur data perubahan tidak valid: " + err.Error()})
		return
	}

	valTahun := parseMitraInterfaceToInt(input.Tahun)
	if valTahun == 0 {
		valTahun = record.Tahun
	}

	// 3. Mapping fields updates menggunakan map interface{}
	updates := map[string]interface{}{
		"kategori_mitra":         input.Kategori,
		"pic_nasabah":            input.PicNasabah,
		"bulan_terdaftar":        input.BulanTerdaftar,
		"tahun":                  valTahun,
		"kode_owner":             input.KodeOwner,
		"owner_mitra":            input.Owner,
		"nama_brand":             input.Brand,
		"kategori_mitra_sub":     input.KategoriSub,
		"total_akuisisi_referal": parseMitraInterfaceToInt(input.TotalAkuisisiReferal),
		"total_referral":         parseMitraInterfaceToInt(input.TotalReferral),
		"telp":                   input.Telp,
		"rekening":               input.Rekening,
		"wilayah":                input.Wilayah,
		"alamat":                 input.Alamat,
	}

	if err := config.DB.Model(&record).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal merubah data master mitra di database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Data Master Mitra Berhasil Diperbarui!",
		"data":    record,
	})
}

// 🗑️ DELETE /api/list-mitra/delete/:id
func DeleteListMitra(c *gin.Context) {
	id := c.Param("id")
	var record models.ListMitra

	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data partner mitra tidak ditemukan atau sudah dihapus sebelumnya"})
		return
	}

	if err := config.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengeksekusi penghapusan dari SQL database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Data Master Mitra Berhasil Dihapus Secara Permanen dari Sistem!",
	})
}