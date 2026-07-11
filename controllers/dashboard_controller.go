package controllers

import (
	"net/http"
	"piposmart-backend/config"
	"piposmart-backend/models"

	"github.com/gin-gonic/gin"
)

// GET /api/dashboard/stats
func GetDashboardStats(c *gin.Context) {
	var totalKelolaan int64

	// 1. Hitung total baris di tabel data_kelolaans (menggantikan mitra & nasabah lama)
	if err := config.DB.Model(&models.DataKelolaan{}).Count(&totalKelolaan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data kelolaan"})
		return
	}

	// 2. Kembalikan data dalam bentuk JSON terstruktur ke Frontend Next.js kamu
	// Note: total_nasabah tetap kita kirim (diisi nilai totalKelolaan atau 0) agar dashboard frontend tidak error/blank jika masih membaca key tersebut.
	c.JSON(http.StatusOK, gin.H{
		"total_mitra":   totalKelolaan,
		"total_nasabah": totalKelolaan, // Diarahkan ke totalKelolaan karena nasabah sudah menyatu ke sini
	})
}