package database

import (
	"fmt"
	"log"

	"backend_crm_piposmart/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	// Alamat koneksi ke mysql server
	dsn := "root:@tcp(localhost:3306)/crm_piposmart?charset=utf8mb4&parseTime=True&loc=Local"

	// melakukan koneksi ke database mysql menggunakan gorm,
	// dan menghasilkan objek database dan error (jika ada)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	// Jika ada error, maka hentikan koneksi dan berikan pesan error
	if err != nil {
		fmt.Println("Error Connected To Database")
		return err
	}

	// Selanjutnya, definisikan objek database agar diakses global
	DB = db

	if err := DB.AutoMigrate(
		&models.Sales{},
		&models.Customer{},
		&models.CustomerStatus{},
		&models.CallHistory{},
		&models.TrainingHistory{},
		&models.PurchaseHistory{},
	); err != nil {
		log.Fatal("Error saat Migrasi Tabel Database: ", err)
	}

	fmt.Println("Succesful Connectod to database")

	return nil
}
