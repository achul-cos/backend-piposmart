package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"piposmart-backend/config"
	"piposmart-backend/models"
)

// ==========================================================
// DTO RINGKAS UNTUK TABEL UTAMA (LIST) — jangan tambah field
// baru di sini kecuali memang mau ikut tampil di tabel compact.
// Semua field lain tetap tersimpan penuh di database, cuma
// tidak dikirim di response list ini.
// ==========================================================
type PenjualanSummary struct {
	ID             uint      `json:"id"`
	Tanggal        time.Time `json:"tanggal"`
	Status         string    `json:"status"`
	NamaOwner      string    `json:"nama_owner"`
	NamaOutlet     string    `json:"nama_outlet"`
	NamaBrand      string    `json:"nama_brand"`
	TotalPenjualan float64   `json:"total_penjualan"`
	PicTeam        string    `json:"pic_team"`
	TargetPaket    string    `json:"target_paket"` // 🌟 FIX UTAMA: Loloskan target_paket agar Chart Donut Dashboard bisa baca kuantitas!
}

// GET /api/pipo/penjualan
// Dipakai tabel utama -> HANYA kirim field ringkas (summary)
func GetPenjualan(c *gin.Context) {
	var listPenjualan []models.Penjualan

	if err := config.DB.Order("tanggal DESC").Find(&listPenjualan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	summaries := make([]PenjualanSummary, 0, len(listPenjualan))
	for _, p := range listPenjualan {
		summaries = append(summaries, PenjualanSummary{
			ID:             p.ID,
			Tanggal:        p.Tanggal,
			Status:         p.Status,
			NamaOwner:      p.NamaOwner,
			NamaOutlet:     p.NamaOutlet,
			NamaBrand:      p.NamaBrand,
			TotalPenjualan: p.TotalPenjualan,
			PicTeam:        p.PicTeam,
			TargetPaket:    p.TargetPaket, // 🌟 Petakan field paket ke FE summary list
		})
	}

	c.JSON(http.StatusOK, summaries)
}

// GET /api/pipo/penjualan/detail/:id
// Dipakai saat baris tabel diklik (modal rincian / edit) -> kirim SEMUA field
func GetPenjualanDetail(c *gin.Context) {
	id := c.Param("id")
	var penjualan models.Penjualan

	if err := config.DB.First(&penjualan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, penjualan)
}

// POST /api/pipo/penjualan
func CreatePenjualan(c *gin.Context) {
	type CreateInput struct {
		TanggalInput     string  `json:"tanggalInput"`
		SumberData       string  `json:"sumberData"`
		PicNasabah       string  `json:"picNasabah"`
		KodeOwner        string  `json:"kodeOwner"`
		NamaOwner        string  `json:"namaOwner"`
		Brand            string  `json:"brand"`
		NamaOutlet       string  `json:"namaOutlet"`
		HpOwner          string  `json:"hpOwner"`
		Status           string  `json:"status"`
		Membership       string  `json:"membership"`
		TargetPaket      string  `json:"targetPaket"`
		TargetNominal    float64 `json:"targetNominal"`
		NominalAktual    float64 `json:"nominalAktual"`
		TanggalTraining  string  `json:"tanggalTraining"`
		StatusTraining   string  `json:"statusTraining"`
		TanggalRealisasi string  `json:"tanggalRealisasi"`
		BuktiTransfer    string  `json:"buktiTransfer"`
	}

	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Payload JSON tidak valid: " + err.Error()})
		return
	}

	waktuTransaksi := time.Now()
	if input.TanggalInput != "" {
		if t, err := time.Parse("2006-01-02", input.TanggalInput); err == nil {
			waktuTransaksi = t
		}
	}

	var tTraining *time.Time
	if input.TanggalTraining != "" {
		if t, err := time.Parse("2006-01-02", input.TanggalTraining); err == nil {
			tTraining = &t
		}
	}

	var tRealisasi *time.Time
	if input.TanggalRealisasi != "" {
		if t, err := time.Parse("2006-01-02", input.TanggalRealisasi); err == nil {
			tRealisasi = &t
		}
	}

	newPenjualan := models.Penjualan{
		Tanggal:          waktuTransaksi,
		PicTeam:          input.PicNasabah,
		KodeOwner:        input.KodeOwner,
		NamaOwner:        input.NamaOwner,
		NamaBrand:        input.Brand,
		NamaOutlet:       input.NamaOutlet,
		HpOwner:          input.HpOwner,
		SumberData:       input.SumberData,
		Status:           input.Status,
		Membership:       input.Membership,
		TargetPaket:      input.TargetPaket,
		TargetNominal:    input.TargetNominal,
		PaketAktual:      input.TargetPaket,
		TotalPenjualan:   input.NominalAktual,
		TanggalTraining:  tTraining,
		StatusTraining:   input.StatusTraining,
		TanggalRealisasi: tRealisasi,
		BuktiTransfer:    input.BuktiTransfer,
	}

	if err := config.DB.Create(&newPenjualan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan ke database SQL: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Data pipeline CRM berhasil disimpan!", "data": newPenjualan})
}

// PUT /api/pipo/penjualan/:id
func UpdatePenjualan(c *gin.Context) {
	id := c.Param("id")
	var penjualan models.Penjualan

	if err := config.DB.First(&penjualan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data tidak ditemukan"})
		return
	}

	type UpdateInput struct {
		TanggalInput     *string  `json:"tanggalInput"`
		SumberData       *string  `json:"sumberData"`
		PicNasabah       *string  `json:"picNasabah"`
		KodeOwner        *string  `json:"kodeOwner"`
		NamaOwner        *string  `json:"namaOwner"`
		Brand            *string  `json:"brand"`
		NamaOutlet       *string  `json:"namaOutlet"`
		HpOwner          *string  `json:"hpOwner"`
		Status           *string  `json:"status"`
		Membership       *string  `json:"membership"`
		TargetPaket      *string  `json:"targetPaket"`
		TargetNominal    *float64 `json:"targetNominal"`
		NominalAktual    *float64 `json:"nominalAktual"`
		TanggalTraining  *string  `json:"tanggalTraining"`
		StatusTraining   *string  `json:"statusTraining"`
		TanggalRealisasi *string  `json:"tanggalRealisasi"`
		BuktiTransfer    *string  `json:"buktiTransfer"`
	}

	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	updates := map[string]interface{}{}

	if input.TanggalInput != nil && *input.TanggalInput != "" {
		if t, err := time.Parse("2006-01-02", *input.TanggalInput); err == nil {
			updates["tanggal"] = t
		}
	}
	if input.PicNasabah != nil { updates["pic_team"] = *input.PicNasabah }
	if input.KodeOwner != nil { updates["kode_owner"] = *input.KodeOwner }
	if input.NamaOwner != nil { updates["nama_owner"] = *input.NamaOwner }
	if input.Brand != nil { updates["nama_brand"] = *input.Brand }
	if input.NamaOutlet != nil { updates["nama_outlet"] = *input.NamaOutlet }
	if input.HpOwner != nil { updates["hp_owner"] = *input.HpOwner }
	if input.SumberData != nil { updates["sumber_data"] = *input.SumberData }
	if input.Status != nil { updates["status"] = *input.Status }
	if input.Membership != nil { updates["membership"] = *input.Membership }
	if input.TargetPaket != nil { updates["target_paket"] = *input.TargetPaket }
	if input.TargetNominal != nil { updates["target_nominal"] = *input.TargetNominal }
	if input.NominalAktual != nil { updates["total_penjualan"] = *input.NominalAktual }
	if input.StatusTraining != nil { updates["status_training"] = *input.StatusTraining }
	if input.BuktiTransfer != nil { updates["bukti_transfer"] = *input.BuktiTransfer }

	if input.TanggalTraining != nil {
		if *input.TanggalTraining != "" {
			if t, err := time.Parse("2006-01-02", *input.TanggalTraining); err == nil {
				updates["tanggal_training"] = &t
			}
		} else {
			updates["tanggal_training"] = nil
		}
	}
	if input.TanggalRealisasi != nil {
		if *input.TanggalRealisasi != "" {
			if t, err := time.Parse("2006-01-02", *input.TanggalRealisasi); err == nil {
				updates["tanggal_realisasi"] = &t
			}
		} else {
			updates["tanggal_realisasi"] = nil
		}
	}

	if err := config.DB.Model(&penjualan).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Record pipeline CRM berhasil diperbarui", "data": penjualan})
}

// DELETE /api/pipo/penjualan/:id
func DeletePenjualan(c *gin.Context) {
	id := c.Param("id")
	var penjualan models.Penjualan

	if err := config.DB.First(&penjualan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data tidak ditemukan"})
		return
	}

	if err := config.DB.Delete(&penjualan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menghapus data dari database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Record pipeline crm berhasil dihapus secara permanen"})
}

// ==========================================================
// IMPORT EXCEL (BULK) — dipakai frontend saat import
// ==========================================================
type bulkPenjualanInput struct {
	TanggalInput     string  `json:"tanggalInput"`
	SumberData       string  `json:"sumberData"`
	PicNasabah       string  `json:"picNasabah"`
	KodeOwner        string  `json:"kodeOwner"`
	NamaOwner        string  `json:"namaOwner"`
	Brand            string  `json:"brand"`
	NamaOutlet       string  `json:"namaOutlet"`
	HpOwner          string  `json:"hpOwner"`
	Status           string  `json:"status"`
	Membership       string  `json:"membership"`
	TargetPaket      string  `json:"targetPaket"`
	TargetNominal    float64 `json:"targetNominal"`
	NominalAktual    float64 `json:"nominalAktual"`
	TanggalTraining  string  `json:"tanggalTraining"`
	StatusTraining   string  `json:"statusTraining"`
	TanggalRealisasi string  `json:"tanggalRealisasi"`
	BuktiTransfer    string  `json:"buktiTransfer"`
}

// POST /api/pipo/penjualan/bulk
func BulkCreatePenjualan(c *gin.Context) {
	var inputs []bulkPenjualanInput
	if err := c.ShouldBindJSON(&inputs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Payload JSON tidak valid: " + err.Error()})
		return
	}

	parseTanggal := func(s string, fallback time.Time) time.Time {
		if s == "" { return fallback }
		if t, err := time.Parse("2006-01-02", s); err == nil { return t }
		return fallback
	}
	parseTanggalPtr := func(s string) *time.Time {
		if s == "" { return nil }
		if t, err := time.Parse("2006-01-02", s); err == nil { return &t }
		return nil
	}

	batch := make([]models.Penjualan, 0, len(inputs))
	for _, in := range inputs {
		batch = append(batch, models.Penjualan{
			Tanggal:          parseTanggal(in.TanggalInput, time.Now()),
			PicTeam:          in.PicNasabah,
			KodeOwner:        in.KodeOwner,
			NamaOwner:        in.NamaOwner,
			NamaBrand:        in.Brand,
			NamaOutlet:       in.NamaOutlet,
			HpOwner:          in.HpOwner,
			SumberData:       in.SumberData,
			Status:           in.Status,
			Membership:       in.Membership,
			TargetPaket:      in.TargetPaket,
			TargetNominal:    in.TargetNominal,
			PaketAktual:      in.TargetPaket,
			TotalPenjualan:   in.NominalAktual,
			TanggalTraining:  parseTanggalPtr(in.TanggalTraining),
			StatusTraining:   in.StatusTraining,
			TanggalRealisasi: parseTanggalPtr(in.TanggalRealisasi),
			BuktiTransfer:    in.BuktiTransfer,
		})
	}

	if len(batch) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Tidak ada data valid untuk diimport"})
		return
	}

	if err := config.DB.CreateInBatches(&batch, 100).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan data import: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Import Excel berhasil disimpan ke database", "count": len(batch)})
}

// ==========================================================
// HAPUS SEMUA DATA (dengan proteksi double confirm di FE)
// ==========================================================
func DeleteAllPenjualan(c *gin.Context) {
	if err := config.DB.Where("1 = 1").Delete(&models.Penjualan{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengosongkan database"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}