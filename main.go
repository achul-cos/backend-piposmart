package main

import (
	"piposmart-backend/config"
	"piposmart-backend/controllers"
	"piposmart-backend/models"

	"github.com/gin-gonic/gin"
)

// 🌟 SINKRONISASI AKUN TIM: Menambahkan akun resmi privat Fenya ke database seeder
func SeedInisialisasiUser() {
	var count int64
	// Proteksi pengecekan agar tidak menyuntikkan data ganda ke SQL
	config.DB.Model(&models.User{}).Count(&count)

	if count == 0 {
		akunTim := []models.User{
			{
				Username: "satria@piposmart.com",
				Password: "admin123",
				Nama:     "Satria Ramadhan",
				Role:     "Admin",
			},
			{
				Username: "lydia@piposmart.com",
				Password: "sales123",
				Nama:     "Lydia Marpaung",
				Role:     "Sales",
			},
			{
				Username: "laura@piposmart.com",
				Password: "sales456",
				Nama:     "Laura",
				Role:     "Sales",
			},
			{
				// 🚀 BARU: Akun resmi untuk Fenya
				Username: "fenya@piposmart.com",
				Password: "sales789", // Bisa disesuaikan nanti
				Nama:     "Fenya",
				Role:     "Sales",
			},
		}
		// Kirim data akun ke database SQL via GORM
		config.DB.Create(&akunTim)
	}
}

func main() {
	// 1. Inisialisasi Koneksi Database Utama
	config.ConnectDatabase()

	// 🛠️ AUTOMIGRATE SINKRON: Mendaftarkan model ke SQL database
	config.DB.AutoMigrate(
		&models.User{}, // 🚀 Membuat tabel 'users' otomatis
		&models.Penjualan{},
		&models.Target{},
		&models.DataKelolaan{},
		&models.KelolaanMitra{},
		&models.CallChat{},
		&models.Report{},
		&models.ListMitra{},
		&models.SopActivity{},
	)

	// Jalankan penyuntikan data akun tim setelah tabel terbuat
	SeedInisialisasiUser()

	r := gin.Default()

	// 🛡️ AMANKAN CORS POLICY GLOBAL
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ─── 🛒 GROUP 1: API MODUL PIPO ──────────────
	pipo := r.Group("/api/pipo")
	{
		pipo.GET("/penjualan", controllers.GetPenjualan)
		pipo.POST("/penjualan", controllers.CreatePenjualan)
		pipo.PUT("/penjualan/:id", controllers.UpdatePenjualan)
		pipo.DELETE("/penjualan/:id", controllers.DeletePenjualan)

		// 🛠️ BARU: rincian lengkap & import Excel massal
		pipo.GET("/penjualan/detail/:id", controllers.GetPenjualanDetail)
		pipo.POST("/penjualan/bulk", controllers.BulkCreatePenjualan)
		// ⚠️ Sengaja pakai POST (bukan DELETE) supaya tidak bentrok dengan
		// wildcard "DELETE /penjualan/:id" di radix tree Gin
		pipo.POST("/penjualan/delete-all", controllers.DeleteAllPenjualan)

		pipo.GET("/target", controllers.GetTargets)
		pipo.POST("/target/update", controllers.UpdateTarget)
	}

	// ─── 🤝 GROUP 2: API UTAMA CRM SIDEBAR ────────────
	api := r.Group("/api")
	{
		// 🌟 🔑 ROUTE LOGIN: Endpoint utama untuk Next.js login
		api.POST("/login", controllers.Login)

		// 📊 Modul Data Kelolaan
		api.GET("/kelolaan", controllers.GetDataKelolaan)
		api.POST("/kelolaan/create", controllers.CreateDataKelolaan)
		api.PUT("/kelolaan/update/:id", controllers.UpdateDataKelolaan)
		api.DELETE("/kelolaan/delete/:id", controllers.DeleteDataKelolaan)
		api.GET("/kelolaan/export", controllers.ExportDataKelolaan)
		// 🛠️ BARU DISISIPKAN DI SINI: Jalur penerima mass-upload Excel ke database MySQL
		api.POST("/kelolaan/bulk", controllers.BulkCreateDataKelolaan)

		// 🏢 Modul Kelolaan Mitra
		api.GET("/kelolaan-mitra", controllers.GetKelolaanMitra)
		api.POST("/kelolaan-mitra/create", controllers.CreateKelolaanMitra)
		api.PUT("/kelolaan-mitra/update/:id", controllers.UpdateKelolaanMitra)
		api.DELETE("/kelolaan-mitra/delete/:id", controllers.DeleteKelolaanMitra)
		api.GET("/kelolaan-mitra/export", controllers.ExportKelolaanMitra)

		// 📞 Modul CallChat
		api.GET("/callchat", controllers.GetCallChat)
		api.POST("/callchat/create", controllers.CreateCallChat)
		api.PUT("/callchat/update/:id", controllers.UpdateCallChat)
		api.DELETE("/callchat/delete/:id", controllers.DeleteCallChat)

		// 📝 Modul Report
		api.GET("/report", controllers.GetReports)
		api.POST("/report/create", controllers.CreateReport)
		api.PUT("/report/update/:id", controllers.UpdateReport)
		api.DELETE("/report/delete/:id", controllers.DeleteReport)

		// 👥 Modul Master List Mitra
		api.GET("/list-mitra", controllers.GetListMitra)
		api.POST("/list-mitra/create", controllers.CreateListMitra)
		api.PUT("/list-mitra/update/:id", controllers.UpdateListMitra)
		api.DELETE("/list-mitra/delete/:id", controllers.DeleteListMitra)

		// 📋 Modul SOP Activity
		api.GET("/sop", controllers.GetSop)
		api.POST("/sop/create", controllers.CreateSop)
		api.PUT("/sop/update/:id", controllers.UpdateSop)
		api.DELETE("/sop/delete/:id", func(c *gin.Context) {
			id := c.Param("id")
			var activity models.SopActivity
			if err := config.DB.First(&activity, id).Error; err != nil {
				c.JSON(404, gin.H{"message": "Data SOP tidak ditemukan"})
				return
			}
			if err := config.DB.Delete(&activity).Error; err != nil {
				c.JSON(500, gin.H{"message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Data SOP berhasil dihapus secara permanen"})
		})

		// 🔄 Legacy / Fallback Routes
		api.GET("/callchat-legacy", controllers.GetDataKelolaan)
		api.GET("/mitra", controllers.GetDataKelolaan)
		api.GET("/list-mitra/export", controllers.GetDataKelolaan)
	}

	r.Run(":8080")
}