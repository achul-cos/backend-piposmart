package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"piposmart-backend/config"
	"piposmart-backend/models"
)

// Struct Input DTO penampung payload request dari Next.js
type CreateReportInput struct {
	Tanggal       string      `json:"tanggal" binding:"required"`
	Keterangan    string      `json:"keterangan" binding:"required"`
	Pic           string      `json:"pic"` // 🌟 FIX UTAMA: Menangkap payload key "pic" dari Next.js
	ResponCall    interface{} `json:"responCall"`
	ResponChat    interface{} `json:"responChat"`
	ResponMeeting interface{} `json:"responMeeting"`
	ResponVisit   interface{} `json:"responVisit"`
	NoResponCount interface{} `json:"noResponCount"`
}

// Helper untuk mengonversi tipe data interface{} ke int secara aman
func parseInterfaceToInt(val interface{}) int {
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

// GET /api/report
func GetReports(c *gin.Context) {
	var listReports []models.Report
	if err := config.DB.Order("tanggal desc, id desc").Find(&listReports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal memuat history report: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, listReports)
}

// POST /api/report/create
func CreateReport(c *gin.Context) {
	var input CreateReportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi input gagal: " + err.Error()})
		return
	}

	call := parseInterfaceToInt(input.ResponCall)
	chat := parseInterfaceToInt(input.ResponChat)
	meeting := parseInterfaceToInt(input.ResponMeeting)
	visit := parseInterfaceToInt(input.ResponVisit)
	noRespon := parseInterfaceToInt(input.NoResponCount)

	totalResponLokal := call + chat + meeting + visit
	grandTotalLokal := totalResponLokal + noRespon

	// Ambil nama PIC, pasang default jika kosong
	picFinal := input.Pic
	if picFinal == "" {
		picFinal = "Satria"
	}

	// Konversi penamaan nama bulan lokal Indonesia
	namaBulan := ""
	parsedTime, err := time.Parse("2006-01-02", input.Tanggal)
	if err == nil {
		months := map[string]string{
			"January": "Januari", "February": "Februari", "March": "Maret", "April": "April",
			"May": "Mei", "June": "Juni", "July": "Juli", "August": "Agustus",
			"September": "September", "October": "Oktober", "November": "November", "December": "Desember",
		}
		namaBulan = months[parsedTime.Format("January")]
	} else {
		namaBulan = time.Now().Format("January")
	}

	reportData := models.Report{
		Tanggal:       input.Tanggal, 
		Bulan:         namaBulan,     
		Keterangan:    input.Keterangan,
		Pic:           picFinal, // 🌟 FIX UTAMA: Menyimpan nama PIC asli (Lydia/Laura/Satria) ke database
		ResponCall:    call,     
		ResponChat:    chat,     
		ResponMeeting: meeting,  
		ResponVisit:   visit,    
		NoResponCount: noRespon, 
		TotalRespon:   totalResponLokal,
		TotalNoRespon: noRespon,
		GrandTotal:    grandTotalLokal, 
		CreatedAt:     time.Now(),
	}

	if err := config.DB.Create(&reportData).Error; err != nil {
		fmt.Println("🔥 ERROR SAVE DATABASE REPORT:", err.Error())
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "ganda") {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Gagal menyimpan! Batasan indeks tanggal ganda mendeteksi laporan kembar di sistem database.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan entri database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Log data laporan berhasil disimpan!",
		"data":    reportData,
	})
}

// ✏️ PUT /api/report/update/:id
func UpdateReport(c *gin.Context) {
	id := c.Param("id")
	var report models.Report

	if err := config.DB.First(&report, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data report log tidak ditemukan!"})
		return
	}

	var input CreateReportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi update gagal: " + err.Error()})
		return
	}

	call := parseInterfaceToInt(input.ResponCall)
	chat := parseInterfaceToInt(input.ResponChat)
	meeting := parseInterfaceToInt(input.ResponMeeting)
	visit := parseInterfaceToInt(input.ResponVisit)
	noRespon := parseInterfaceToInt(input.NoResponCount)

	totalResponLokal := call + chat + meeting + visit
	grandTotalLokal := totalResponLokal + noRespon

	namaBulan := ""
	parsedTime, err := time.Parse("2006-01-02", input.Tanggal)
	if err == nil {
		months := map[string]string{
			"January": "Januari", "February": "Februari", "March": "Maret", "April": "April",
			"May": "Mei", "June": "Juni", "July": "Juli", "August": "Agustus",
			"September": "September", "October": "Oktober", "November": "November", "December": "Desember",
		}
		namaBulan = months[parsedTime.Format("January")]
	} else {
		namaBulan = report.Bulan 
	}

	picFinal := input.Pic
	if picFinal == "" {
		picFinal = report.Pic
	}

	updates := map[string]interface{}{
		"tanggal":         input.Tanggal,
		"bulan":           namaBulan,
		"keterangan":      input.Keterangan,
		"pic":             picFinal, // 🌟 FIX UTAMA: Update field PIC di DB
		"respon_call":     call,
		"respon_chat":     chat,
		"respon_meeting":  meeting,
		"respon_visit":    visit,
		"no_respon_count": noRespon,
		"total_respon":    totalResponLokal,
		"total_no_respon": noRespon,
		"grand_total":     grandTotalLokal,
		"updated_at":      time.Now(),
	}

	if err := config.DB.Model(&report).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal memperbarui data database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Data performa laporan kerja harian berhasil diperbarui!",
		"data":    report,
	})
}

// 🗑️ DELETE /api/report/delete/:id
func DeleteReport(c *gin.Context) {
	id := c.Param("id")
	var report models.Report

	if err := config.DB.First(&report, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data report log tidak ditemukan atau sudah dihapus!"})
		return
	}

	if err := config.DB.Delete(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menghapus log report permanen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Data log report harian berhasil dihapus secara permanen dari sistem!",
	})
}