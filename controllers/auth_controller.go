package controllers

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"piposmart-backend/config"
	"piposmart-backend/models"
)

// Gunakan kata kunci rahasia yang sama untuk mengunci token
var jwtKey = []byte("PIPOSMART_SUPER_SECRET_KEY")

type LoginPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Input tidak valid"})
		return
	}

	var user models.User
	// 🔍 Cari user berdasarkan email/username yang dikirim frontend
	if err := config.DB.Where("username = ?", payload.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Email atau Password internal salah"})
		return
	}

	// 🔒 Validasi Password
	if payload.Password != user.Password { 
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Email atau Password internal salah"})
		return
	}

	// 🎫 Generate JWT Token (Masa aktif 24 Jam)
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"username": user.Username,
		"nama":     user.Nama,
		"role":     user.Role,
		"exp":      expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal membuat token akses"})
		return
	}

	// 🚀 Kirim respons sukses ke Next.js
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"token":  tokenString,
		"user": gin.H{
			"nama": user.Nama,
			"role": user.Role,
			"pic":  user.Nama, // Menjamin kecocokan state otomatisasi Next.js
		},
	})
}