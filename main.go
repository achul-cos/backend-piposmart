package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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
			return errors.New("perintah migrate memerlukan action: up, down, status, atau version")
		}
		return migration.Run(ctx, cfg, args[1], os.Stdout)
	case "seed":
		if len(args) < 2 {
			return errors.New("perintah seed memerlukan mode: master atau demo")
		}
		return seeder.Run(ctx, cfg, args[1:], os.Stdout, logger)
	case "bootstrap-admin":
		return app.RunBootstrapAdmin(ctx, cfg, logger)
	default:
		return fmt.Errorf("command %q tidak dikenali\n\n%s", command, usage)
	}
}

const usage = `CRM Piposmart backend

Usage:
  crm api
  crm worker
  crm migrate <up|down|status|version>
  crm seed master
  crm seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
  crm bootstrap-admin
  crm version
  crm help

Jalankan bootstrap-admin setelah migration dan seed master.`

func printUsage() {
	fmt.Println(usage)
}

func usageError() error {
	return errors.New(usage)
}
