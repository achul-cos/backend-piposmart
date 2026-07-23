package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"backend_crm_piposmart/internal/activity"
	"backend_crm_piposmart/internal/catalog"
	"backend_crm_piposmart/internal/closing"
	"backend_crm_piposmart/internal/customer"
	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/lead"
	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/httpx"
	"backend_crm_piposmart/internal/subscription"
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

		subscriptionRepository := subscription.NewRepository(connection.SQLDB())
		subscriptionService := subscription.NewService(subscriptionRepository)
		subscriptionRoutes := api.Group("")
		subscriptionRoutes.Use(identity.AuthMiddleware(identityService))
		subscription.NewHandler(subscriptionService).RegisterRoutes(subscriptionRoutes)
	}

	router.NoRoute(func(c *gin.Context) {
		httpx.Error(c, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route tidak ditemukan", nil)
	})
	router.NoMethod(func(c *gin.Context) {
		httpx.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "HTTP method tidak didukung", nil)
	})

	return router
}
