package migration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"

	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/seeder"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
)

func Run(ctx context.Context, cfg config.Config, command string, output io.Writer) (err error) {
	if !isSupportedCommand(command) {
		return fmt.Errorf("perintah migration %q tidak didukung; gunakan up, down, reset, clear, status, atau version", command)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("goose panic: %v", recovered)
		}
	}()

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
	case "clear":
		err = clearAndSeedMaster(ctx, db, cfg, output)
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
	case "up", "down", "reset", "clear", "status", "version":
		return true
	default:
		return false
	}
}

type clearTableInfo struct {
	Name             string
	HasAutoIncrement bool
}

func clearAndSeedMaster(ctx context.Context, db *sql.DB, cfg config.Config, output io.Writer) error {
	tables, err := listClearableTables(ctx, db, cfg.Database.Name)
	if err != nil {
		return fmt.Errorf("siapkan daftar tabel clear: %w", err)
	}

	fmt.Fprintln(output, "PERINGATAN: migrate clear akan menghapus seluruh data non-master dari database aktif.")
	fmt.Fprintln(output, "Tabel migrasi Goose akan dipertahankan, lalu seed master dijalankan ulang.")

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("ambil koneksi clear: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("nonaktifkan foreign key checks: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SET FOREIGN_KEY_CHECKS = 1")
	}()

	for _, table := range tables {
		if err := clearTable(ctx, conn, table); err != nil {
			return err
		}
	}

	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return fmt.Errorf("aktifkan kembali foreign key checks: %w", err)
	}

	fmt.Fprintf(output, "clear database selesai: %d tabel data dibersihkan. Menjalankan seed master...\n", len(tables))
	if err := seeder.Run(ctx, cfg, []string{seeder.ModeMaster}, output, slog.Default()); err != nil {
		return fmt.Errorf("seed master setelah clear: %w", err)
	}

	fmt.Fprintln(output, "migrate clear selesai: database kini hanya menyisakan data seed master.")
	fmt.Fprintln(output, "Catatan: seluruh akun Admin/Supervisor/Sales ikut terhapus. Jalankan bootstrap-admin bila ingin login kembali.")
	return nil
}

func listClearableTables(ctx context.Context, db *sql.DB, schema string) ([]clearTableInfo, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, CASE WHEN auto_increment IS NULL THEN 0 ELSE 1 END AS has_auto_increment
		FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'
		ORDER BY table_name ASC`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]clearTableInfo, 0)
	for rows.Next() {
		var item clearTableInfo
		var hasAutoIncrement int
		if err := rows.Scan(&item.Name, &hasAutoIncrement); err != nil {
			return nil, err
		}
		if isClearPreservedTable(item.Name) {
			continue
		}
		item.HasAutoIncrement = hasAutoIncrement == 1
		tables = append(tables, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func clearTable(ctx context.Context, conn *sql.Conn, table clearTableInfo) error {
	quoted := quoteIdentifier(table.Name)

	if _, err := conn.ExecContext(ctx, "TRUNCATE TABLE "+quoted); err == nil {
		return nil
	}

	if _, err := conn.ExecContext(ctx, "DELETE FROM "+quoted); err != nil {
		return fmt.Errorf("hapus data tabel %s: %w", table.Name, err)
	}

	if table.HasAutoIncrement {
		if _, err := conn.ExecContext(ctx, "ALTER TABLE "+quoted+" AUTO_INCREMENT = 1"); err != nil {
			return fmt.Errorf("reset auto increment tabel %s: %w", table.Name, err)
		}
	}

	return nil
}

func isClearPreservedTable(table string) bool {
	return len(table) >= len("goose_") && table[:len("goose_")] == "goose_"
}

func quoteIdentifier(name string) string {
	return "`" + sqlEscapeIdentifier(name) + "`"
}

func sqlEscapeIdentifier(name string) string {
	result := make([]rune, 0, len(name))
	for _, r := range name {
		if r == '`' {
			result = append(result, '`', '`')
			continue
		}
		result = append(result, r)
	}
	return string(result)
}
