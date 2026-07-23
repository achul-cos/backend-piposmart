package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	setMinimumEnvironment(t)
	t.Setenv("APP_PORT", "9090")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,https://crm.example.com")

	cfg, err := LoadFromEnvironment()
	if err != nil {
		t.Fatalf("LoadFromEnvironment() error = %v", err)
	}

	if cfg.App.Port != 9090 {
		t.Fatalf("APP_PORT = %d, want 9090", cfg.App.Port)
	}
	if cfg.App.Address() != "0.0.0.0:9090" {
		t.Fatalf("Address() = %q", cfg.App.Address())
	}
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins length = %d, want 2", len(cfg.CORS.AllowedOrigins))
	}
	if strings.Contains(cfg.Database.DSN(), "localhost") {
		t.Fatalf("DSN tidak memakai DB_HOST dari environment")
	}
}

func TestLoadFromEnvironmentRejectsMissingDatabaseConfiguration(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")

	_, err := LoadFromEnvironment()
	if err == nil {
		t.Fatal("LoadFromEnvironment() seharusnya gagal")
	}
	if !strings.Contains(err.Error(), "DB_HOST") ||
		!strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("error tidak menjelaskan konfigurasi database: %v", err)
	}
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	setMinimumEnvironment(t)
	t.Setenv("APP_ENV", EnvironmentProduction)
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	_, err := LoadFromEnvironment()
	if err == nil || !strings.Contains(err.Error(), "production") {
		t.Fatalf("expected production CORS error, got %v", err)
	}
}

func TestDatabaseDSNEscapesCredentials(t *testing.T) {
	setMinimumEnvironment(t)
	t.Setenv("DB_PASSWORD", "p@ss:word/with?symbols")

	cfg, err := LoadFromEnvironment()
	if err != nil {
		t.Fatalf("LoadFromEnvironment() error = %v", err)
	}

	dsn := cfg.Database.DSN()
	if !strings.Contains(dsn, "p@ss:word/with?symbols") {
		t.Fatalf("driver DSN tidak mempertahankan password khusus: %q", dsn)
	}
}

func TestLoadDotEnvDoesNotOverrideProcessEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "EXISTING_VALUE=from-file\nQUOTED_VALUE=\"hello world\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dotenv fixture: %v", err)
	}

	t.Setenv("EXISTING_VALUE", "from-process")
	t.Setenv("QUOTED_VALUE", "")
	os.Unsetenv("QUOTED_VALUE")
	t.Cleanup(func() { os.Unsetenv("QUOTED_VALUE") })

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() error = %v", err)
	}
	if got := os.Getenv("EXISTING_VALUE"); got != "from-process" {
		t.Fatalf("EXISTING_VALUE = %q", got)
	}
	if got := os.Getenv("QUOTED_VALUE"); got != "hello world" {
		t.Fatalf("QUOTED_VALUE = %q", got)
	}
}

func setMinimumEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", EnvironmentTest)
	t.Setenv("DB_HOST", "db.test")
	t.Setenv("DB_NAME", "crm_test")
	t.Setenv("DB_USER", "crm_test")
	t.Setenv("DB_PASSWORD", "test-password")
}
