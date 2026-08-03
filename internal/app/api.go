package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"backend_crm_piposmart/internal/activity"
	"backend_crm_piposmart/internal/catalog"
	"backend_crm_piposmart/internal/closing"
	"backend_crm_piposmart/internal/customer"
	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/importing"
	"backend_crm_piposmart/internal/kpi"
	"backend_crm_piposmart/internal/lead"
	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/database"
	"backend_crm_piposmart/internal/platform/httpserver"
	"backend_crm_piposmart/internal/platform/jobqueue"
	"backend_crm_piposmart/internal/reporting"
	"backend_crm_piposmart/internal/subscription"
	"backend_crm_piposmart/internal/target"
	"backend_crm_piposmart/internal/wallet"
)

func RunAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	connection, err := database.Open(ctx, cfg.Database, logger)
	if err != nil {
		return err
	}
	defer connection.Close()

	router := httpserver.NewRouter(cfg, logger, connection)
	server := &http.Server{
		Addr:              cfg.App.Address(),
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api server started",
			slog.String("address", cfg.App.Address()),
			slog.String("environment", cfg.App.Environment),
		)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("api shutdown requested")
	case serverError := <-serverErrors:
		if !errors.Is(serverError, http.ErrServerClosed) {
			return fmt.Errorf("api server: %w", serverError)
		}
		return nil
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("api server stopped")
	return nil
}

func RunWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	connection, err := database.Open(ctx, cfg.Database, logger)
	if err != nil {
		return err
	}
	defer connection.Close()

	ticker := time.NewTicker(cfg.Worker.PollInterval)
	defer ticker.Stop()

	jobRepository := jobqueue.NewRepository(connection.SQL)
	kpiRepository := kpi.NewRepository(connection.SQL)
	importingRepository := importing.NewRepository(connection.SQL)
	customerService := customer.NewService(customer.NewRepository(connection.SQL))
	leadService := lead.NewService(lead.NewRepository(connection.SQL))
	walletRepository := wallet.NewRepository(connection.SQL)
	walletService := wallet.NewService(walletRepository)
	subscriptionService := subscription.NewService(subscription.NewRepository(connection.SQL))
	catalogService := catalog.NewService(catalog.NewRepository(connection.SQL))
	targetService := target.NewService(target.NewRepository(connection.SQL))
	activityService := activity.NewService(activity.NewRepository(connection.SQL))
	closingService := closing.NewService(closing.NewRepository(connection.SQL))
	reportingRepository := reporting.NewRepository(connection.SQL)
	reportingService := reporting.NewService(reportingRepository, jobRepository, cfg.Storage)
	registry := jobqueue.Registry{
		kpi.JobTypeRecompute:      kpi.RecomputeHandler(kpiRepository),
		importing.JobTypeValidate: importing.ValidateHandler(importingRepository),
		importing.JobTypeCommit: importing.CommitHandler(
			importingRepository, customerService, leadService,
			walletService, subscriptionService, catalogService, targetService,
			activityService, closingService,
		),
		reporting.JobTypeGenerateExport: reporting.GenerateExportHandler(reportingRepository, reportingService),
	}

	logger.Info("worker started",
		slog.Duration("poll_interval", cfg.Worker.PollInterval),
		slog.Int("max_attempts", cfg.Worker.MaxAttempts),
		slog.Duration("stale_job_timeout", cfg.Worker.StaleJobTimeout),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return nil
		case <-ticker.C:
			pingContext, cancel := context.WithTimeout(ctx, cfg.Database.ConnectTimeout)
			err := connection.PingContext(pingContext)
			cancel()
			if err != nil {
				logger.Error("worker database heartbeat failed", slog.String("error", err.Error()))
				continue
			}
			logger.Debug("worker heartbeat", slog.String("status", "ready"))

			if reclaimed, err := jobRepository.ReclaimStale(ctx, cfg.Worker.StaleJobTimeout); err != nil {
				logger.Error("worker stale job reclaim failed", slog.String("error", err.Error()))
			} else if reclaimed > 0 {
				logger.Warn("worker reclaimed stale jobs", slog.Int64("count", reclaimed))
			}

			// Sprint 15a §2: a cheap idempotent bulk UPDATE, not worth a dedicated job type —
			// PENDING top-ups older than their 24h session window auto-EXPIRE every tick.
			if expired, err := walletRepository.ExpireStaleTopups(ctx); err != nil {
				logger.Error("worker top-up expiry failed", slog.String("error", err.Error()))
			} else if expired > 0 {
				logger.Info("worker expired stale top-ups", slog.Int64("count", expired))
			}

			// Drain the queue each tick rather than handling one job per tick, so a burst of
			// enqueued jobs (e.g. several KPI recompute requests) doesn't wait multiple
			// PollInterval cycles to clear.
			for {
				handled, dispatchErr := jobqueue.Dispatch(ctx, connection.SQL, jobRepository, registry)
				if dispatchErr != nil {
					logger.Error("worker job dispatch failed", slog.String("error", dispatchErr.Error()))
				}
				if !handled {
					break
				}
			}
		}
	}
}

func RunBootstrapAdmin(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	connection, err := database.Open(ctx, cfg.Database, logger)
	if err != nil {
		return err
	}
	defer connection.Close()

	repository := identity.NewRepository(connection.SQL)
	service := identity.NewService(repository, cfg)
	user, err := service.BootstrapAdmin(ctx)
	if err != nil {
		return err
	}

	logger.Info("bootstrap admin ready",
		slog.Int64("user_id", user.ID),
		slog.String("email", user.Email),
	)
	return nil
}
