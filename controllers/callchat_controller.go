package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"piposmart-backend/config"
	"piposmart-backend/models"

	"github.com/gin-gonic/gin"
)

// Struct Input DTO penampung payload request dari Next.js
type CreateCallChatPayload struct {
	TanggalFu             string      `json:"tanggalFu" binding:"required"`
	Bulan                 string      `json:"bulan"`
	TanggalDibagikan      string      `json:"tanggalDibagikan"`
	StatusAkun            string      `json:"statusAkun"`
	StatusAcntAlternative string      `json:"statusAcnt"`
	Pic                   string      `json:"pic"` // 🌟 FIX UTAMA: Menangkap key "pic" yang dikirim oleh Next.js Lydia
	KodeBaris             interface{} `json:"kodeBaris"`
	KodeOwner             interface{} `json:"kodeOwner"`
	NamaOwner             string      `json:"namaOwner" binding:"required"`
	Brand                 string      `json:"brand"`
	Outlet                string      `json:"outlet"`
	HpOwner               string      `json:"hpOwner" binding:"required"`
}

// Helper untuk merubah interface{} dari JSON menjadi int secara aman
func parseCallChatToUint(val interface{}) uint {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return uint(v)
	case int:
		return uint(v)
	case string:
		if v == "" {
			return 0
		}
		if i, err := strconv.Atoi(v); err == nil {
			return uint(i)
		}
	}
	return 0
}

// isDuplicateError mendeteksi error unique constraint dari berbagai driver DB
func isDuplicateError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "ganda")
}

// GET /api/callchat
func GetCallChat(c *gin.Context) {
	var list []models.CallChat
	if err := config.DB.Order("id desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengambil log callchat"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// POST /api/callchat/create
func CreateCallChat(c *gin.Context) {
	var payload CreateCallChatPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		fmt.Println("🔥 ERROR BINDING JSON CALLCHAT:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Validasi struktur input gagal: " + err.Error()})
		return
	}

	statusFinal := payload.StatusAkun
	if statusFinal == "" && payload.StatusAcntAlternative != "" {
		statusFinal = payload.StatusAcntAlternative
	}
	if statusFinal == "" {
		statusFinal = "Akun Baru"
	}

	// Fallback aman seandainya payload PIC dari frontend Next.js bernilai kosong kosong
	picFinal := payload.Pic
	if picFinal == "" {
		picFinal = "Satria"
	}

	kBaris := parseCallChatToUint(payload.KodeBaris)
	kOwner := parseCallChatToUint(payload.KodeOwner)

	callChatData := models.CallChat{
		TanggalFu:   payload.TanggalFu,
		Bulan:       payload.Bulan,
		TanggalOrig: payload.TanggalDibagikan,
		StatusAkun:  statusFinal,
		Pic:         picFinal, // 🌟 FIX UTAMA: Menyimpan data nama PIC (Lydia/Laura/Satria) ke database
		KodeBaris:   int(kBaris),
		KodeOwner:   int(kOwner),
		NamaOwner:   payload.NamaOwner,
		Brand:       payload.Brand,
		Outlet:      payload.Outlet,
		HpOwner:     payload.HpOwner,
		CreatedAt:   time.Now(),
	}

	if err := config.DB.Create(&callChatData).Error; err != nil {
		fmt.Println("🔥 ERROR SAVE DATABASE GORM CALLCHAT:", err.Error())

		if isDuplicateError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Gagal Menyimpan: Nomor HP Owner ini sudah terdaftar dalam sistem log tracking.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan log ke database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Sakti! Log riwayat monitoring tele-marketing berhasil disimpan!",
		"data":    callChatData,
	})
}

// PUT /api/callchat/update/:id
func UpdateCallChat(c *gin.Context) {
	id := c.Param("id")
	var existingData models.CallChat

	if err := config.DB.First(&existingData, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data log Call & Chat tidak ditemukan"})
		return
	}

	var payload CreateCallChatPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Struktur edit data tidak valid: " + err.Error()})
		return
	}

	statusFinal := payload.StatusAkun
	if statusFinal == "" && payload.StatusAcntAlternative != "" {
		statusFinal = payload.StatusAcntAlternative
	}
	if statusFinal == "" {
		statusFinal = "Akun Baru"
	}

	picFinal := payload.Pic
	if picFinal == "" {
		picFinal = "Satria"
	}

	kBaris := parseCallChatToUint(payload.KodeBaris)
	kOwner := parseCallChatToUint(payload.KodeOwner)

	existingData.TanggalFu = payload.TanggalFu
	existingData.Bulan = payload.Bulan
	existingData.TanggalOrig = payload.TanggalDibagikan
	existingData.StatusAkun = statusFinal
	existingData.Pic = picFinal // 🌟 FIX UTAMA: Memperbarui data nama PIC saat proses edit/update log
	existingData.KodeBaris = int(kBaris)
	existingData.KodeOwner = int(kOwner)
	existingData.NamaOwner = payload.NamaOwner
	existingData.Brand = payload.Brand
	existingData.Outlet = payload.Outlet
	existingData.HpOwner = payload.HpOwner

	if err := config.DB.Save(&existingData).Error; err != nil {
		if isDuplicateError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Gagal Memperbarui: Nomor HP Owner ini bentrok dengan data log lainnya.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal memperbarui database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Log Call & Chat berhasil diperbarui secara sukses!",
		"data":    existingData,
	})
}

// DELETE /api/callchat/delete/:id
func DeleteCallChat(c *gin.Context) {
	id := c.Param("id")
	var data models.CallChat

	if err := config.DB.First(&data, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data log tidak ditemukan atau telah dihapus sebelumnya"})
		return
	}

	if err := config.DB.Delete(&data).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menghapus data log dari database: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Data log Call & Chat berhasil dihapus dari sistem secara permanen",
	})
}