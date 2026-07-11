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

// Struct Input DTO khusus Create (POST)
type CreatePotensiInput struct {
	TanggalInput     string      `json:"tanggalInput" binding:"required"`
	SumberData       string      `json:"sumberData"`
	PicNasabah       string      `json:"picNasabah" binding:"required"`
	KodeOwner        string      `json:"kodeOwner" binding:"required"`
	NamaOwner        string      `json:"namaOwner" binding:"required"`
	Brand            string      `json:"brand" binding:"required"`
	HpOwner          string      `json:"hpOwner" binding:"required"`
	Status           string      `json:"status" binding:"required"`
	Membership       string      `json:"membership"`
	TargetPaket      string      `json:"targetPaket"`
	TargetNominal    interface{} `json:"targetNominal"`
	PaketAktual      string      `json:"paketAktual"`
	NominalAktual    interface{} `json:"nominalAktual"`
	TanggalTraining  string      `json:"tanggalTraining"`
	StatusTraining   string      `json:"statusTraining"`
	TimelineTraining string      `json:"timelineTraining"`
	RatingPlaystore  string      `json:"ratingPlaystore"`
	TanggalRealisasi string      `json:"tanggalRealisasi"`
	BuktiTransfer    string      `json:"buktiTransfer"`
}

// Struct DTO khusus Update (Mencegah error binding required saat edit data)
type UpdatePotensiInput struct {
	TanggalInput     string      `json:"tanggalInput"`
	SumberData       string      `json:"sumberData"`
	PicNasabah       string      `json:"picNasabah"`
	KodeOwner        string      `json:"kodeOwner"`
	NamaOwner        string      `json:"namaOwner"`
	Brand            string      `json:"brand"`
	HpOwner          string      `json:"hpOwner"`
	Status           string      `json:"status"`
	Membership       string      `json:"membership"`
	TargetPaket      string      `json:"targetPaket"`
	TargetNominal    interface{} `json:"targetNominal"`
	PaketAktual      string      `json:"paketAktual"`
	NominalAktual    interface{} `json:"nominalAktual"`
	TanggalTraining  string      `json:"tanggalTraining"`
	StatusTraining   string      `json:"statusTraining"`
	TimelineTraining string      `json:"timelineTraining"`
	RatingPlaystore  string      `json:"ratingPlaystore"`
	TanggalRealisasi string      `json:"tanggalRealisasi"`
	BuktiTransfer    string      `json:"buktiTransfer"`
}

// Helper pengonversi angka interface{} ke float64 secara aman
func parseInterfaceToFloat(val interface{}) float64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if v == "" {
			return 0
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

// GET /api/potensi
func GetPotensi(c *gin.Context) {
	var listPotensi []models.Potensi
	if err := config.DB.Order("tanggal_input desc, id desc").Find(&listPotensi).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengambil data pipeline"})
		return
	}
	c.JSON(http.StatusOK, listPotensi)
}

// POST /api/potensi/create
func CreatePotensi(c *gin.Context) {
	var input CreatePotensiInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fmt.Println("❌ GAGAL BINDING JSON POTENSI:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi struktur input gagal: " + err.Error()})
		return
	}

	tNominal := parseInterfaceToFloat(input.TargetNominal)
	nAktual := parseInterfaceToFloat(input.NominalAktual)

	// Pemetaan bersih langsung ke Model Database GORM Potensi
	potensiObj := models.Potensi{
		TanggalInput:     input.TanggalInput,
		SumberData:       input.SumberData,
		PicNasabah:       input.PicNasabah,
		KodeOwner:        input.KodeOwner,
		NamaOwner:        input.NamaOwner,
		Brand:            input.Brand,
		HpOwner:          input.HpOwner,
		Status:           input.Status,
		Membership:       input.Membership,
		TargetPaket:      input.TargetPaket,
		TargetNominal:    tNominal,
		PaketAktual:      input.PaketAktual,
		NominalAktual:    nAktual,
		TanggalTraining:  input.TanggalTraining,
		StatusTraining:   input.StatusTraining,
		TimelineTraining: input.TimelineTraining,
		RatingPlaystore:  input.RatingPlaystore,
		TanggalRealisasi: input.TanggalRealisasi,
		BuktiTransfer:    input.BuktiTransfer,
		CreatedAt:        time.Now(),
	}

	if err := config.DB.Create(&potensiObj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan ke database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Sakti! Data Prospek Pipeline Berhasil Disimpan!", "data": potensiObj})
}

// ✏️ PUT /api/potensi/update/:id
func UpdatePotensi(c *gin.Context) {
	id := c.Param("id")
	var record models.Potensi

	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data potensi pipeline tidak ditemukan!"})
		return
	}

	var input UpdatePotensiInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi struktur data edit gagal: " + err.Error()})
		return
	}

	tNominal := parseInterfaceToFloat(input.TargetNominal)
	nAktual := parseInterfaceToFloat(input.NominalAktual)

	updates := map[string]interface{}{
		"tanggal_input":     input.TanggalInput,
		"sumber_data":       input.SumberData,
		"pic_nasabah":       input.PicNasabah,
		"kode_owner":        input.KodeOwner,
		"nama_owner":        input.NamaOwner,
		"brand":             input.Brand,
		"hp_owner":          input.HpOwner,
		"status":            input.Status,
		"membership":        input.Membership,
		"target_paket":      input.TargetPaket,
		"target_nominal":    tNominal,
		"paket_aktual":      input.PaketAktual,
		"nominal_aktual":    nAktual,
		"tanggal_training":  input.TanggalTraining,
		"status_training":   input.StatusTraining,
		"timeline_training": input.TimelineTraining,
		"rating_playstore":  input.RatingPlaystore,
		"tanggal_realisasi": input.TanggalRealisasi,
		"bukti_transfer":    input.BuktiTransfer,
		"updated_at":        time.Now(),
	}

	if err := config.DB.Model(&record).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal memperbarui database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Data Potensi & Demo berhasil diperbarui!", "data": record})
}

// 🗑️ DELETE /api/potensi/delete/:id
func DeletePotensi(c *gin.Context) {
	id := c.Param("id")
	var record models.Potensi

	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data tidak ditemukan atau sudah dihapus!"})
		return
	}

	if err := config.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menghapus data dari database SQL: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Data Potensi & Demo berhasil dihapus secara permanen!"})
}