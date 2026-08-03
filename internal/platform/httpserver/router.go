package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"backend_crm_piposmart/internal/activity"
	"backend_crm_piposmart/internal/analytics"
	"backend_crm_piposmart/internal/catalog"
	"backend_crm_piposmart/internal/closing"
	"backend_crm_piposmart/internal/customer"
	"backend_crm_piposmart/internal/discussion"
	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/importing"
	"backend_crm_piposmart/internal/kpi"
	"backend_crm_piposmart/internal/lead"
	"backend_crm_piposmart/internal/partner"
	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/httpx"
	"backend_crm_piposmart/internal/platform/jobqueue"
	"backend_crm_piposmart/internal/reporting"
	"backend_crm_piposmart/internal/subscription"
	"backend_crm_piposmart/internal/target"
	"backend_crm_piposmart/internal/transfer"
	"backend_crm_piposmart/internal/wallet"

	"database/sql"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Connection interface {
	PingContext(context.Context) error
	SQLDB() *sql.DB
}

func NewRouter(cfg config.Config, logger *slog.Logger, connection Connection) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(
		requestIDMiddleware(),
		accessLogMiddleware(logger),
		recoveryMiddleware(logger),
		corsMiddleware(cfg.CORS),
	)

	router.GET("/openapi.yaml", serveOpenAPI)
	router.GET(
		"/swagger/*any",
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/openapi.yaml")),
	)

	router.GET("/health/live", func(c *gin.Context) {
		httpx.Success(c, http.StatusOK, gin.H{
			"status":  "alive",
			"service": cfg.App.Name,
		})
	})

	router.GET("/health/ready", func(c *gin.Context) {
		checkContext, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := connection.PingContext(checkContext); err != nil {
			logger.WarnContext(c.Request.Context(), "readiness check failed",
				slog.String("request_id", httpx.RequestID(c)),
				slog.String("dependency", "mysql"),
				slog.String("error", err.Error()),
			)
			httpx.Error(
				c,
				http.StatusServiceUnavailable,
				"SERVICE_NOT_READY",
				"Service belum siap menerima request",
				gin.H{"mysql": "unavailable"},
			)
			return
		}

		httpx.Success(c, http.StatusOK, gin.H{
			"status": "ready",
			"mysql":  "available",
		})
	})

	router.GET("/", func(c *gin.Context) {
		httpx.Success(c, http.StatusOK, gin.H{
			"service":     cfg.App.Name,
			"environment": cfg.App.Environment,
			"status":      "online",
		})
	})

	api := router.Group(cfg.App.BasePath)
	api.GET("/status", func(c *gin.Context) {
		httpx.Success(c, http.StatusOK, gin.H{"status": "online"})
	})

	if connection.SQLDB() != nil {
		identityRepository := identity.NewRepository(connection.SQLDB())
		identityService := identity.NewService(identityRepository, cfg)
		identity.NewHandler(identityService).RegisterRoutes(api)

		customerRepository := customer.NewRepository(connection.SQLDB())
		customerService := customer.NewService(customerRepository)
		customerRoutes := api.Group("")
		customerRoutes.Use(identity.AuthMiddleware(identityService))
		customer.NewHandler(customerService).RegisterRoutes(customerRoutes)

		leadRepository := lead.NewRepository(connection.SQLDB())
		leadService := lead.NewService(leadRepository)
		leadRoutes := api.Group("")
		leadRoutes.Use(identity.AuthMiddleware(identityService))
		lead.NewHandler(leadService).RegisterRoutes(leadRoutes)

		activityRepository := activity.NewRepository(connection.SQLDB())
		activityService := activity.NewService(activityRepository)
		activityRoutes := api.Group("")
		activityRoutes.Use(identity.AuthMiddleware(identityService))
		activity.NewHandler(activityService).RegisterRoutes(activityRoutes)

		catalogRepository := catalog.NewRepository(connection.SQLDB())
		catalogService := catalog.NewService(catalogRepository)
		catalogRoutes := api.Group("")
		catalogRoutes.Use(identity.AuthMiddleware(identityService))
		catalog.NewHandler(catalogService).RegisterRoutes(catalogRoutes)

		closingRepository := closing.NewRepository(connection.SQLDB())
		closingService := closing.NewService(closingRepository)
		closingRoutes := api.Group("")
		closingRoutes.Use(identity.AuthMiddleware(identityService))
		closing.NewHandler(closingService).RegisterRoutes(closingRoutes)

		walletRepository := wallet.NewRepository(connection.SQLDB())
		walletService := wallet.NewService(walletRepository)
		walletRoutes := api.Group("")
		walletRoutes.Use(identity.AuthMiddleware(identityService))
		wallet.NewHandler(walletService).RegisterRoutes(walletRoutes)

		transferRepository := transfer.NewRepository(connection.SQLDB())
		transferService := transfer.NewService(transferRepository, walletService)
		transferRoutes := api.Group("")
		transferRoutes.Use(identity.AuthMiddleware(identityService))
		transfer.NewHandler(transferService).RegisterRoutes(transferRoutes)

		subscriptionRepository := subscription.NewRepository(connection.SQLDB())
		subscriptionService := subscription.NewService(subscriptionRepository)
		subscriptionRoutes := api.Group("")
		subscriptionRoutes.Use(identity.AuthMiddleware(identityService))
		subscription.NewHandler(subscriptionService).RegisterRoutes(subscriptionRoutes)

		partnerRepository := partner.NewRepository(connection.SQLDB())
		partnerService := partner.NewService(partnerRepository, cfg)
		partnerRoutes := api.Group("")
		partnerRoutes.Use(identity.AuthMiddleware(identityService))
		partner.NewHandler(partnerService).RegisterRoutes(partnerRoutes)

		targetRepository := target.NewRepository(connection.SQLDB())
		targetService := target.NewService(targetRepository)
		targetRoutes := api.Group("")
		targetRoutes.Use(identity.AuthMiddleware(identityService))
		target.NewHandler(targetService).RegisterRoutes(targetRoutes)

		jobRepository := jobqueue.NewRepository(connection.SQLDB())
		kpiRepository := kpi.NewRepository(connection.SQLDB())
		kpiService := kpi.NewService(kpiRepository, jobRepository)
		kpiRoutes := api.Group("")
		kpiRoutes.Use(identity.AuthMiddleware(identityService))
		kpi.NewHandler(kpiService).RegisterRoutes(kpiRoutes)

		importingRepository := importing.NewRepository(connection.SQLDB())
		importingService := importing.NewService(importingRepository, jobRepository, cfg.Storage)
		importingRoutes := api.Group("")
		importingRoutes.Use(identity.AuthMiddleware(identityService))
		importing.NewHandler(importingService).RegisterRoutes(importingRoutes)

		analyticsRepository := analytics.NewRepository(connection.SQLDB())
		analyticsService := analytics.NewService(analyticsRepository)
		analyticsRoutes := api.Group("")
		analyticsRoutes.Use(identity.AuthMiddleware(identityService))
		analytics.NewHandler(analyticsService).RegisterRoutes(analyticsRoutes)

		reportingRepository := reporting.NewRepository(connection.SQLDB())
		reportingService := reporting.NewService(reportingRepository, jobRepository, cfg.Storage)
		reportingRoutes := api.Group("")
		reportingRoutes.Use(identity.AuthMiddleware(identityService))
		reporting.NewHandler(reportingService).RegisterRoutes(reportingRoutes)

		discussionRepository := discussion.NewRepository(connection.SQLDB())
		discussionService := discussion.NewService(discussionRepository)
		discussionRoutes := api.Group("")
		discussionRoutes.Use(identity.AuthMiddleware(identityService))
		discussion.NewHandler(discussionService).RegisterRoutes(discussionRoutes)
	}

	router.NoRoute(func(c *gin.Context) {
		httpx.Error(c, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route tidak ditemukan", nil)
	})
	router.NoMethod(func(c *gin.Context) {
		httpx.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "HTTP method tidak didukung", nil)
	})

	return router
}
