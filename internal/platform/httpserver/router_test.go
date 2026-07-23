package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/logging"
)

type readinessStub struct {
	err error
}

func (s readinessStub) PingContext(context.Context) error {
	return s.err
}

func (s readinessStub) SQLDB() *sql.DB {
	return nil
}

func TestLiveHealthCheck(t *testing.T) {
	router := NewRouter(testConfig(), discardLogger(), readinessStub{})
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Header().Get(requestIDHeader) == "" {
		t.Fatal("response tidak memiliki X-Request-ID")
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"status":"alive"`)) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestReadyHealthCheckWhenDatabaseUnavailable(t *testing.T) {
	router := NewRouter(
		testConfig(),
		discardLogger(),
		readinessStub{err: errors.New("connection refused")},
	)
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"code":"SERVICE_NOT_READY"`)) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestRequestIDIsPreserved(t *testing.T) {
	router := NewRouter(testConfig(), discardLogger(), readinessStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	request.Header.Set(requestIDHeader, "request-from-client")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(requestIDHeader); got != "request-from-client" {
		t.Fatalf("X-Request-ID = %q", got)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"request_id":"request-from-client"`)) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestNotFoundUsesStandardErrorEnvelope(t *testing.T) {
	router := NewRouter(testConfig(), discardLogger(), readinessStub{})
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"code":"ROUTE_NOT_FOUND"`)) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestOpenAPISpecIsServed(t *testing.T) {
	router := NewRouter(testConfig(), discardLogger(), readinessStub{})
	request := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("openapi: 3.0.3")) {
		t.Fatalf("unexpected spec: %s", recorder.Body.String())
	}
}

func testConfig() config.Config {
	cfg := config.Config{
		App: config.AppConfig{
			Name:        "crm-test",
			Environment: config.EnvironmentTest,
			BasePath:    "/api/v1",
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"http://localhost:3000"},
		},
	}
	return cfg
}

func discardLogger() *slog.Logger {
	return logging.NewForWriter(&bytes.Buffer{}, slog.LevelDebug)
}
