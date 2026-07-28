package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"backend_crm_piposmart/internal/app"
	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/logging"
	"backend_crm_piposmart/internal/platform/migration"
	"backend_crm_piposmart/internal/platform/seeder"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}

	command := args[0]
	if command == "version" {
		fmt.Printf("crm %s (commit=%s, built=%s)\n", version, commit, buildTime)
		return nil
	}
	if command == "help" || command == "--help" || command == "-h" {
		printUsage()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger, err := logging.New(cfg.Log)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch command {
	case "api":
		return app.RunAPI(ctx, cfg, logger)
	case "worker":
		return app.RunWorker(ctx, cfg, logger)
	case "migrate":
		if len(args) < 2 {
			return errors.New("perintah migrate memerlukan action: up, down, reset, status, atau version")
		}
		return migration.Run(ctx, cfg, args[1], os.Stdout)
	case "seed":
		if len(args) < 2 {
			return errors.New("perintah seed memerlukan mode: master atau demo")
		}
		return seeder.Run(ctx, cfg, args[1:], os.Stdout, logger)
	case "bootstrap-admin":
		return app.RunBootstrapAdmin(ctx, cfg, logger)
	case "setup":
		return runSetup(ctx, cfg, logger)
	default:
		return fmt.Errorf("command %q tidak dikenali\n\n%s", command, usage)
	}
}

func runSetup(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	logger.Info("setup started")

	if setupBoolEnv("SETUP_RUN_MIGRATE", true) {
		logger.Info("setup step", slog.String("action", "migrate up"))
		if err := migration.Run(ctx, cfg, "up", os.Stdout); err != nil {
			return fmt.Errorf("setup migrate up: %w", err)
		}
	}

	if setupBoolEnv("SETUP_RUN_SEED_MASTER", true) {
		logger.Info("setup step", slog.String("action", "seed master"))
		if err := seeder.Run(ctx, cfg, []string{"master"}, os.Stdout, logger); err != nil {
			return fmt.Errorf("setup seed master: %w", err)
		}
	}

	if setupBoolEnv("SETUP_RUN_BOOTSTRAP_ADMIN", true) {
		logger.Info("setup step", slog.String("action", "bootstrap-admin"))
		if err := app.RunBootstrapAdmin(ctx, cfg, logger); err != nil {
			return fmt.Errorf("setup bootstrap-admin: %w", err)
		}
	}

	if setupBoolEnv("SETUP_RUN_DEMO_SEED", true) {
		seedArgs := buildSetupDemoSeedArgs()
		logger.Info("setup step", slog.String("action", "seed demo"), slog.Any("args", seedArgs))
		if err := seeder.Run(ctx, cfg, seedArgs, os.Stdout, logger); err != nil {
			return fmt.Errorf("setup seed demo: %w", err)
		}
	}

	logger.Info("setup completed")
	return nil
}

func buildSetupDemoSeedArgs() []string {
	today := time.Now().UTC().Format("2006-01-02")
	preset := setupStringEnv("SETUP_DEMO_PRESET", "minimal")
	args := []string{
		"demo",
		"--preset=" + preset,
		"--seed=" + setupStringEnv("SETUP_DEMO_SEED", "20260723"),
		"--from=" + setupStringEnv("SETUP_DEMO_FROM", today),
		"--to=" + setupStringEnv("SETUP_DEMO_TO", today),
	}
	if strings.EqualFold(preset, "large") {
		args = append(args,
			"--scale="+setupStringEnv("SETUP_DEMO_SCALE", "1"),
			"--variation="+setupStringEnv("SETUP_DEMO_VARIATION", "0.5"),
		)
	}
	return args
}

func setupBoolEnv(name string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func setupStringEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

const usage = `CRM Piposmart backend

Usage:
  crm api
  crm worker
  crm setup
  crm migrate <up|down|reset|status|version>
  crm seed master
  crm seed demo --preset=minimal --seed=20260723 --from=2026-07-01 --to=2026-07-01
  crm bootstrap-admin
  crm version
  crm help

setup akan menjalankan migrate up, seed master, bootstrap-admin, dan seed demo sesuai environment.`

func printUsage() {
	fmt.Println(usage)
}

func usageError() error {
	return errors.New(usage)
}
