package config

import (
	"fmt"
	"log"
	"os"
	"gorm.io/driver/mysql"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
    // 1. Load file .env untuk mengambil password database
    err := godotenv.Load()
    if err != nil {
        log.Println("Peringatan: Gagal memuat file .env, menggunakan env system")
    }

    // 2. Ambil konfigurasi dari env
    host := os.Getenv("DB_HOST")
    user := os.Getenv("DB_USER")
    password := os.Getenv("DB_PASSWORD")
    dbName := os.Getenv("DB_NAME")
    port := os.Getenv("DB_PORT")

    // UBAH DI SINI: Format DSN diubah ke gaya MySQL
    // Format: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbName)
    
    // UBAH DI SINI: Hubungkan ke MySQL Laragon menggunakan driver mysql.Open
    database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Gagal terhubung ke database MySQL Laragon:", err)
    }

    DB = database
    fmt.Println("Berhasil terhubung ke database MySQL Laragon!")
}