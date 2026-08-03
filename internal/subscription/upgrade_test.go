package subscription

import (
	"testing"
	"time"
)

func TestProratedPlanAmount(t *testing.T) {
	total, daily, err := proratedPlanAmount("300000", 30, 3)
	if err != nil {
		t.Fatalf("proratedPlanAmount error: %v", err)
	}
	if total != 3000000 {
		t.Fatalf("expected total cents 3000000, got %d", total)
	}
	if daily != 1000000 {
		t.Fatalf("expected daily cents 1000000, got %d", daily)
	}
}

func TestEffectiveUpgradeStartDateRejectsFutureOfPurchasedAt(t *testing.T) {
	purchasedAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	_, err := effectiveUpgradeStartDate("2026-08-04", purchasedAt)
	if err == nil {
		t.Fatal("expected error for future effective start date")
	}
}

func TestEffectiveUpgradeStartDateDefaultsToPurchasedDate(t *testing.T) {
	purchasedAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	got, err := effectiveUpgradeStartDate("", purchasedAt)
	if err != nil {
		t.Fatalf("effectiveUpgradeStartDate error: %v", err)
	}
	if got.Format("2006-01-02") != "2026-08-03" {
		t.Fatalf("expected 2026-08-03, got %s", got.Format("2006-01-02"))
	}
}
