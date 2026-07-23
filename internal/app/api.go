package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/database"
	"backend_crm_piposmart/internal/platform/httpserver"
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

	logger.Info("worker started",
		slog.Duration("poll_interval", cfg.Worker.PollInterval),
		slog.Int("max_attempts", cfg.Worker.MaxAttempts),
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
		}
	}
}
