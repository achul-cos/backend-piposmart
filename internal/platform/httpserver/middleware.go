package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" || len(requestID) > 128 {
			requestID = newRequestID()
		}

		c.Set("request_id", requestID)
		c.Header(requestIDHeader, requestID)
		c.Next()
	}
}

func accessLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		duration := time.Since(startedAt)
		requestID := httpx.RequestID(c)
		status := c.Writer.Status()
		errorInfo, hasErrorInfo := httpx.CurrentError(c)
		actorID, actorRole := currentActorLogFields(c)

		if status >= http.StatusInternalServerError {
			fields := []any{
				slog.String("request_id", requestID),
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
				slog.String("raw_query", c.Request.URL.RawQuery),
				slog.Int("status", status),
				slog.Int("response_bytes", c.Writer.Size()),
				slog.Duration("duration", duration),
				slog.String("client_ip", c.ClientIP()),
				slog.Int64("actor_id", actorID),
				slog.String("actor_role", actorRole),
			}
			if hasErrorInfo {
				errorDetails := errorInfo.Details
				if errorInfo.Private != nil {
					errorDetails = errorInfo.Private
				}
				fields = append(fields,
					slog.String("error_code", errorInfo.Code),
					slog.String("error_message", errorInfo.Message),
					slog.String("error_details", formatLogDetails(errorDetails)),
				)
			}
			logger.ErrorContext(c.Request.Context(), "http server error", fields...)
			return
		}

		if status >= http.StatusBadRequest {
			fields := []any{
				slog.String("request_id", requestID),
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
				slog.String("raw_query", c.Request.URL.RawQuery),
				slog.Int("status", status),
				slog.Int("response_bytes", c.Writer.Size()),
				slog.Duration("duration", duration),
				slog.String("client_ip", c.ClientIP()),
				slog.Int64("actor_id", actorID),
				slog.String("actor_role", actorRole),
			}
			if hasErrorInfo {
				fields = append(fields,
					slog.String("error_code", errorInfo.Code),
					slog.String("error_message", errorInfo.Message),
				)
			}
			logger.WarnContext(c.Request.Context(), "http client error", fields...)
			return
		}

		logger.InfoContext(c.Request.Context(), "http request",
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Int("response_bytes", c.Writer.Size()),
			slog.Duration("duration", duration),
			slog.String("client_ip", c.ClientIP()),
			slog.Int64("actor_id", actorID),
			slog.String("actor_role", actorRole),
		)
	}
}

func recoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(c.Request.Context(), "panic recovered",
					slog.String("request_id", httpx.RequestID(c)),
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)
				httpx.Error(
					c,
					http.StatusInternalServerError,
					"INTERNAL_SERVER_ERROR",
					"Terjadi kesalahan internal",
					nil,
				)
			}
		}()
		c.Next()
	}
}

func corsMiddleware(cfg config.CORSConfig) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", requestIDHeader, "Idempotency-Key"},
		ExposeHeaders:    []string{requestIDHeader},
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           12 * time.Hour,
	}
	return cors.New(corsConfig)
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(bytes)
}

func currentActorLogFields(c *gin.Context) (int64, string) {
	user, ok := identity.CurrentUser(c)
	if !ok {
		return 0, ""
	}
	return user.ID, user.RoleCode
}

func formatLogDetails(details any) string {
	if details == nil {
		return ""
	}
	return fmt.Sprintf("%v", details)
}
