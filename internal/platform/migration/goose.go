package migration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"

	"backend_crm_piposmart/internal/platform/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
)

func Run(ctx context.Context, cfg config.Config, command string, output io.Writer) error {
	if !isSupportedCommand(command) {
		return fmt.Errorf("perintah migration %q tidak didukung; gunakan up, down, reset, status, atau version", command)
	}

	db, err := sql.Open("mysql", cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("buka database migration: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database migration: %w", err)
	}
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("set dialect Goose: %w", err)
	}
	goose.SetLogger(log.New(output, "", log.LstdFlags))

	switch command {
	case "up":
		err = goose.UpContext(ctx, db, cfg.Migration.Directory)
	case "down":
		err = goose.DownContext(ctx, db, cfg.Migration.Directory)
	case "reset":
		err = goose.ResetContext(ctx, db, cfg.Migration.Directory)
	case "status":
		err = goose.StatusContext(ctx, db, cfg.Migration.Directory)
	case "version":
		err = goose.VersionContext(ctx, db, cfg.Migration.Directory)
	}
	if err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}
	return nil
}

func isSupportedCommand(command string) bool {
	switch command {
	case "up", "down", "reset", "status", "version":
		return true
	default:
		return false
	}
}
