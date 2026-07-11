package controllers

import (
	"net/http"
	"piposmart-backend/config"
	"piposmart-backend/models"

	"github.com/gin-gonic/gin"
)

// GET /api/sop
func GetSop(c *gin.Context) {
	var activities []models.SopActivity
	err := config.DB.Order("urutan asc, id asc").Find(&activities).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat database SOP: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, activities)
}

// POST /api/sop/create
func CreateSop(c *gin.Context) {
	var input models.SopActivity
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format data input tidak valid: " + err.Error()})
		return
	}
	
	// Murni append/insert row baru ke database pusat tanpa mengganggu data lama
	if err := config.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan data SOP baru: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, input)
}

// PUT /api/sop/update/:id
func UpdateSop(c *gin.Context) {
	id := c.Param("id")
	var input models.SopActivity
	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format data edit tidak valid: " + err.Error()})
		return
	}
	
	var activity models.SopActivity
	if err := config.DB.First(&activity, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Data SOP tidak ditemukan"})
		return
	}
	
	// Hanya mengupdate kolom yang diubah secara fleksibel
	config.DB.Model(&activity).Updates(input)
	// Jika framework GORM dikonfigurasi berbeda pada project Anda, gunakan syntax di bawah ini:
	// config.DB.Model(&activity).Updates(input)
	
	c.JSON(http.StatusOK, activity)
}