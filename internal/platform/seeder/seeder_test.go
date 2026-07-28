package seeder

import (
	"strings"
	"testing"
	"time"

	"backend_crm_piposmart/internal/platform/config"
)

func TestParseDemoOptions(t *testing.T) {
	options, err := Parse([]string{"demo", "--preset=minimal", "--seed=20260723", "--from=2026-07-01", "--to=2026-07-03"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if options.Mode != ModeDemo || options.Preset != "minimal" || options.Seed != 20260723 {
		t.Fatalf("options tidak sesuai: %+v", options)
	}
	wantFrom := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	if !options.From.Equal(wantFrom) {
		t.Fatalf("From = %s, want %s", options.From, wantFrom)
	}
	if !options.To.Equal(wantTo) {
		t.Fatalf("To = %s, want %s", options.To, wantTo)
	}
	if !options.AsOf.Equal(wantTo) {
		t.Fatalf("AsOf = %s, want %s", options.AsOf, wantTo)
	}
	if options.Variation != 0.5 {
		t.Fatalf("Variation = %v, want 0.5", options.Variation)
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
	if options.Scale != defaultLargeSeedScale {
		t.Fatalf("default scale = %d, want %d", options.Scale, defaultLargeSeedScale)
	}
}

func TestParseAcceptsLargePresetWithScale(t *testing.T) {
	options, err := Parse([]string{"demo", "--preset=large", "--scale=4", "--variation=0.75"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if options.Scale != 4 {
		t.Fatalf("scale = %d, want 4", options.Scale)
	}
	if options.Variation != 0.75 {
		t.Fatalf("variation = %v, want 0.75", options.Variation)
	}
}

func TestParseRejectsScaleForMinimalPreset(t *testing.T) {
	_, err := Parse([]string{"demo", "--preset=minimal", "--scale=2"})
	if err == nil || !strings.Contains(err.Error(), "--scale hanya didukung") {
		t.Fatalf("expected scale rejection for minimal preset, got %v", err)
	}
}

func TestLargeSeedOwnerCountForScale(t *testing.T) {
	count, err := largeSeedOwnerCountForScale(10)
	if err != nil {
		t.Fatalf("largeSeedOwnerCountForScale() error = %v", err)
	}
	if count != 18000 {
		t.Fatalf("count = %d, want 18000", count)
	}
}

func TestLargeSeedOwnerCountRejectsUnknownScale(t *testing.T) {
	_, err := largeSeedOwnerCountForScale(11)
	if err == nil || !strings.Contains(err.Error(), "--scale") {
		t.Fatalf("expected scale validation error, got %v", err)
	}
}

func TestParseRejectsInvalidVariation(t *testing.T) {
	_, err := Parse([]string{"demo", "--preset=large", "--variation=1.5"})
	if err == nil || !strings.Contains(err.Error(), "--variation") {
		t.Fatalf("expected variation validation error, got %v", err)
	}
}

func TestParseRejectsFromAfterTo(t *testing.T) {
	_, err := Parse([]string{"demo", "--preset=large", "--from=2026-07-10", "--to=2026-07-01"})
	if err == nil || !strings.Contains(err.Error(), "--from") {
		t.Fatalf("expected from/to validation error, got %v", err)
	}
}

func TestParseAsOfBackwardsCompatibility(t *testing.T) {
	options, err := Parse([]string{"demo", "--preset=minimal", "--as-of=2026-07-28"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	if !options.From.Equal(want) || !options.To.Equal(want) || !options.AsOf.Equal(want) {
		t.Fatalf("as-of compatibility mismatch: %+v", options)
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
