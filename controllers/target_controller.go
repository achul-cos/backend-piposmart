package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"piposmart-backend/config"
	"piposmart-backend/models"
)

type TargetRekapResponse struct {
	ID                 uint    `json:"id"`
	PicTeam            string  `json:"pic_team"`
	PaketLangganan     string  `json:"paket_langganan"`
	TargetNasabah      int     `json:"target_nasabah"`
	RealisasiNasabah   int     `json:"realisasi_nasabah"`
	TargetPenjualan    float64 `json:"target_penjualan"`
	RealisasiPenjualan float64 `json:"realisasi_penjualan"`
	SelisihPenjualan   float64 `json:"selisih_penjualan"`
	PeriodeBulan       string  `json:"periode_bulan"`
	TanggalAwal        string  `json:"tanggal_awal"`
	TanggalAkhir       string  `json:"tanggal_akhir"`
}

// GET /api/pipo/target
// 🟢 FIXED: Mendukung filter rentang tanggal (Date Range) custom untuk target dan realisasi harian
func GetTargets(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Jika filter tanggal kosong, fallback ke hari ini secara otomatis
	waktuSekarang := time.Now().Format("2006-01-02")
	if startDate == "" {
		startDate = waktuSekarang
	}
	if endDate == "" {
		endDate = waktuSekarang
	}

	// Mengambil parameter bulan "YYYY-MM" diambil dari tanggal awal filter
	bulanFilter := startDate[0:7]

	type KomponenTim struct {
		PicTeam   string
		NamaPaket string
	}
	var listTim []KomponenTim

	namaTabelPenjualan := config.DB.NamingStrategy.TableName("Penjualan")

	// Ambil list sales PIC yang aktif bertransaksi dalam rentang tanggal tersebut
	config.DB.Table(namaTabelPenjualan).
		Select("pic_team, nama_paket").
		Where("SUBSTR(tanggal, 1, 10) BETWEEN ? AND ? AND pic_team != ? AND pic_team != ?", startDate, endDate, "system account", "System Account").
		Group("pic_team, nama_paket").
		Scan(&listTim)

	var responseData []TargetRekapResponse

	defaultTargetNasabah := 11
	defaultTargetPenjualan := 14362000.0

	if len(listTim) == 0 {
		listTim = []KomponenTim{
			{PicTeam: "Satria", NamaPaket: "Pro"},
			{PicTeam: "Lidya", NamaPaket: "Business"},
			{PicTeam: "Laura", NamaPaket: "Basic"},
		}
	}

	for idx, tim := range listTim {
		var realNasabah int64 = 0
		var realOmset float64 = 0.0

		// 🟢 REALISASI: Dihitung secara presisi menggunakan BETWEEN (Rentang Tanggal)
		config.DB.Table(namaTabelPenjualan).
			Where("pic_team = ? AND nama_paket = ? AND SUBSTR(tanggal, 1, 10) BETWEEN ? AND ?", tim.PicTeam, tim.NamaPaket, startDate, endDate).
			Count(&realNasabah)

		config.DB.Table(namaTabelPenjualan).
			Where("pic_team = ? AND nama_paket = ? AND SUBSTR(tanggal, 1, 10) BETWEEN ? AND ?", tim.PicTeam, tim.NamaPaket, startDate, endDate).
			Select("COALESCE(SUM(total_penjualan), 0.0)").
			Row().Scan(&realOmset)

		// Mencari data kustom target bulanan sales aktif
		var kustomTarget models.Target
		err := config.DB.Where("SUBSTR(periode_bulan, 1, 7) = ? AND LOWER(TRIM(pic_team)) = LOWER(TRIM(?))", bulanFilter, tim.PicTeam).First(&kustomTarget).Error
		
		targetNasabahFinal := defaultTargetNasabah
		targetPenjualanFinal := defaultTargetPenjualan

		if err == nil {
			targetNasabahFinal = kustomTarget.TargetNasabah
			targetPenjualanFinal = kustomTarget.TargetPenjualan
		}

		selisih := realOmset - targetPenjualanFinal

		responseData = append(responseData, TargetRekapResponse{
			ID:                 uint(idx + 1),
			PicTeam:            tim.PicTeam,
			PaketLangganan:     tim.NamaPaket,
			TargetNasabah:      targetNasabahFinal,
			RealisasiNasabah:   int(realNasabah),
			TargetPenjualan:    targetPenjualanFinal,
			RealisasiPenjualan: realOmset,
			SelisihPenjualan:   selisih,
			PeriodeBulan:       bulanFilter,
			TanggalAwal:        startDate,
			TanggalAkhir:       endDate,
		})
	}

	c.JSON(http.StatusOK, responseData)
}

// POST /api/pipo/target/update
func UpdateTarget(c *gin.Context) {
	type UpdateTargetInput struct {
		PeriodeBulan    string   `json:"periode_bulan" binding:"required"`
		PicTeam         string   `json:"pic_team" binding:"required"`
		PaketLangganan  string   `json:"paket_langganan" binding:"required"`
		TargetNasabah   *int     `json:"target_nasabah"`
		TargetPenjualan *float64 `json:"target_penjualan"`
	}

	var input UpdateTargetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	var target models.Target
	err := config.DB.Where("SUBSTR(periode_bulan, 1, 7) = ? AND LOWER(TRIM(pic_team)) = LOWER(TRIM(?))", input.PeriodeBulan[0:7], input.PicTeam).First(&target).Error

	if err != nil {
		newTarget := models.Target{
			PeriodeBulan:   input.PeriodeBulan,
			PicTeam:        input.PicTeam,
			PaketLangganan: input.PaketLangganan,
		}
		if input.TargetNasabah != nil {
			newTarget.TargetNasabah = *input.TargetNasabah
		} else {
			newTarget.TargetNasabah = 11
		}
		if input.TargetPenjualan != nil {
			newTarget.TargetPenjualan = *input.TargetPenjualan
		} else {
			newTarget.TargetPenjualan = 14362000.0
		}

		if err := config.DB.Create(&newTarget).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Target baru berhasil dimasukkan", "data": newTarget})
		return
	}

	updates := make(map[string]interface{})
	if input.TargetNasabah != nil {
		updates["target_nasabah"] = *input.TargetNasabah
	}
	if input.TargetPenjualan != nil {
		updates["target_penjualan"] = *input.TargetPenjualan
	}

	if err := config.DB.Model(&models.Target{}).Where("id = ?", target.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Tabel Target berhasil diperbarui"})
}