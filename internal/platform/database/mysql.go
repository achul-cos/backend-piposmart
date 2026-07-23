package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"backend_crm_piposmart/internal/platform/config"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Connection struct {
	GORM *gorm.DB
	SQL  *sql.DB
}

func Open(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) (*Connection, error) {
	gormLogLevel := gormlogger.Warn
	if logger.Enabled(ctx, slog.LevelDebug) {
		gormLogLevel = gormlogger.Info
	}

	db, err := gorm.Open(gormmysql.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("buka koneksi MySQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("ambil koneksi SQL: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}

	logger.Info("database connected",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("database", cfg.Name),
	)

	return &Connection{GORM: db, SQL: sqlDB}, nil
}

func (c *Connection) PingContext(ctx context.Context) error {
	return c.SQL.PingContext(ctx)
}

func (c *Connection) Close() error {
	return c.SQL.Close()
}
