package seeder

import (
	"strings"
	"testing"
	"time"

	"backend_crm_piposmart/internal/platform/config"
)

func TestParseDemoOptions(t *testing.T) {
	options, err := Parse([]string{"demo", "--preset=minimal", "--seed=20260723", "--as-of=2026-07-01"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if options.Mode != ModeDemo || options.Preset != "minimal" || options.Seed != 20260723 {
		t.Fatalf("options tidak sesuai: %+v", options)
	}
	wantAsOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if !options.AsOf.Equal(wantAsOf) {
		t.Fatalf("AsOf = %s, want %s", options.AsOf, wantAsOf)
	}
}

func TestParseRejectsUnknownPreset(t *testing.T) {
	_, err := Parse([]string{"demo", "--preset=huge"})
	if err == nil || !strings.Contains(err.Error(), "minimal") {
		t.Fatalf("expected preset error, got %v", err)
	}
}

func TestParseAcceptsLargePreset(t *testing.T) {
	options, err := Parse([]string{"demo", "--preset=large"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if options.Preset != "large" {
		t.Fatalf("options tidak sesuai: %+v", options)
	}
}

func TestValidateEnvironmentRejectsDemoSeederOnProduction(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{Environment: config.EnvironmentProduction},
	}

	err := ValidateEnvironment(cfg, Options{Mode: ModeDemo})
	if err == nil || !strings.Contains(err.Error(), "production") {
		t.Fatalf("expected production rejection, got %v", err)
	}
}

func TestValidateEnvironmentAllowsMasterOnProduction(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{Environment: config.EnvironmentProduction},
	}

	if err := ValidateEnvironment(cfg, Options{Mode: ModeMaster}); err != nil {
		t.Fatalf("master seeder should be allowed on production: %v", err)
	}
}
