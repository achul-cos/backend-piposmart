package main

import (
	"context"
	"database/sql"
	"log"

	"backend_crm_piposmart/internal/platform/seeder"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:@tcp(localhost:3306)/crm_piposmart?parseTime=true&loc=Local"
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to begin tx: %v", err)
	}

	// Determine admin ID for assignment
	var adminID int64
	err = tx.QueryRowContext(context.Background(), "SELECT u.id FROM users u JOIN roles r ON u.role_id = r.id WHERE r.name = 'ADMIN' LIMIT 1").Scan(&adminID)
	if err != nil {
		log.Printf("Warning: no admin found: %v", err)
		adminID = 1
	}

	err = seeder.SeedMitraFromExcel(context.Background(), tx, adminID, nil)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Failed to seed mitra: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit tx: %v", err)
	}

	log.Println("Successfully synced mitra data from excel!")
}
