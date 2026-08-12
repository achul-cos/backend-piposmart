//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/factory"
	"backend_crm_piposmart/internal/platform/seeder"
	"backend_crm_piposmart/pkg/mysql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	db, err := mysql.New(context.Background(), cfg.Database)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	// get admin id
	var adminID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM users WHERE email = 'admin@piposmart.id' LIMIT 1").Scan(&adminID)
	if err != nil {
		log.Fatalf("get admin id: %v", err)
	}

	fake := factory.New(1, time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC))
	fmt.Println("Seeding subscriptions from Excel...")
	if err := seeder.SeedSubscriptionsFromExcel(ctx, tx, fake, adminID, "", nil); err != nil {
		log.Fatalf("seed error: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit tx: %v", err)
	}
	fmt.Println("Done seeding subscriptions!")
}
