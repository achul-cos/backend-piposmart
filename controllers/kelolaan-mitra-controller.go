package controllers

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"piposmart-backend/config"
	"piposmart-backend/models"
)

// KelolaanMitraInput DTO (Data Transfer Object)
// 🌟 UPDATED: Field referral dibersihkan agar backend tidak menuntut mandatory input lagi
type KelolaanMitraInput struct {
	TanggalInput    string  `json:"tanggalInput" binding:"required"`
	NamaMitra       string  `json:"namaMitra" binding:"required"`
	PicNasabah      string  `json:"picNasabah" binding:"required"`
	BrandUtama      string  `json:"brandUtama" binding:"required"`
	KodeOwnerUtama  string  `json:"kodeOwnerUtama"`
	KategoriMitra   string  `json:"kategoriMitra"`
	PaketLangganan  string  `json:"paketLangganan"`
	StatusLangganan string  `json:"statusLangganan"`
	BuktiFu         string  `json:"buktiFu"`
	StatusInput     string  `json:"statusInput"`
	NominalKomisi   float64 `json:"nominal_komisi"`
	StatusKomisi    string  `json:"status_komisi"`
}

// GET /api/kelolaan-mitra
func GetKelolaanMitra(c *gin.Context) {
	var list []models.KelolaanMitra
	if err := config.DB.Order("tanggal_input desc, id desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengambil data kelolaan mitra"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// POST /api/kelolaan-mitra/create
func CreateKelolaanMitra(c *gin.Context) {
	var input KelolaanMitraInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi input gagal: " + err.Error()})
		return
	}

	stInput := input.StatusInput
	if stInput == "" {
		stInput = "Done"
	}

	// Buat object models baru (Field referral dikosongkan/diabaikan secara aman)
	dataObj := models.KelolaanMitra{
		TanggalInput:    input.TanggalInput,
		NamaMitra:       input.NamaMitra,
		PicNasabah:      input.PicNasabah,
		BrandUtama:      input.BrandUtama,
		KodeOwnerUtama:  input.KodeOwnerUtama,
		KategoriMitra:   input.KategoriMitra,
		PaketLangganan:  input.PaketLangganan,
		StatusLangganan: input.StatusLangganan,
		BuktiFu:         input.BuktiFu,
		StatusInput:     stInput,
		NominalKomisi:   input.NominalKomisi,
		StatusKomisi:    input.StatusKomisi,
		CreatedAt:       time.Now(),
	}

	if err := config.DB.Create(&dataObj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan data ke database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Data Kelolaan Mitra berhasil disimpan!"})
}

// PUT /api/kelolaan-mitra/update/:id
func UpdateKelolaanMitra(c *gin.Context) {
	id := c.Param("id")
	var record models.KelolaanMitra

	// Cari record lama di DB
	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data kelolaan mitra tidak ditemukan"})
		return
	}

	var input KelolaanMitraInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi update gagal: " + err.Error()})
		return
	}

	// Update field menggunakan map GORM
	updates := map[string]interface{}{
		"tanggal_input":    input.TanggalInput,
		"nama_mitra":       input.NamaMitra,
		"pic_nasabah":      input.PicNasabah,
		"brand_utama":      input.BrandUtama,
		"kode_owner_utama": input.KodeOwnerUtama,
		"kategori_mitra":   input.KategoriMitra,
		"paket_langganan":  input.PaketLangganan,
		"status_langganan": input.StatusLangganan,
		"bukti_fu":         input.BuktiFu,
		"status_input":     input.StatusInput,
		"nominal_komisi":   input.NominalKomisi,
		"status_komisi":    input.StatusKomisi,
		"updated_at":       time.Now(),
	}

	if err := config.DB.Model(&record).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal memperbarui data: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Data Kelolaan Mitra berhasil diperbarui!", "data": record})
}

// DELETE /api/kelolaan-mitra/delete/:id
func DeleteKelolaanMitra(c *gin.Context) {
	id := c.Param("id")
	var record models.KelolaanMitra

	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data tidak ditemukan"})
		return
	}

	if err := config.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menghapus data dari database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Data Kelolaan Mitra berhasil dihapus permanen!"})
}

// GET /api/kelolaan-mitra/export
func ExportKelolaanMitra(c *gin.Context) {
	var list []models.KelolaanMitra
	if err := config.DB.Order("tanggal_input desc, id desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal ekspor data"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": list})
}