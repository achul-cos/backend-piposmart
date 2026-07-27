package kpi

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name       string
		totalScore float64
		achieved   float64
		near       float64
		want       string
	}{
		{"exactly achieved threshold", 100, 100, 80, ClassificationAchieved},
		{"above achieved threshold", 120, 100, 80, ClassificationAchieved},
		{"exactly near threshold", 80, 100, 80, ClassificationNearAchieved},
		{"between near and achieved", 90, 100, 80, ClassificationNearAchieved},
		{"below near threshold", 79.99, 100, 80, ClassificationNotAchieved},
		{"zero score", 0, 100, 80, ClassificationNotAchieved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classify(tc.totalScore, tc.achieved, tc.near)
			if got != tc.want {
				t.Fatalf("classify(%v, %v, %v) = %s, want %s", tc.totalScore, tc.achieved, tc.near, got, tc.want)
			}
		})
	}
}

func TestVisibilityWhere(t *testing.T) {
	cases := []struct {
		name     string
		role     string
		wantArgs int
	}{
		{"admin sees all", RoleAdmin, 0},
		{"supervisor sees all", RoleSupervisor, 0},
		{"sales scoped to self", RoleSales, 1},
		{"unknown role denied", "UNKNOWN", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clause, args := visibilityWhere(1, tc.role)
			if clause == "" {
				t.Fatal("expected non-empty clause")
			}
			if len(args) != tc.wantArgs {
				t.Fatalf("expected %d args, got %d", tc.wantArgs, len(args))
			}
		})
	}
}

func TestSupportedMetricQuery(t *testing.T) {
	supported := []string{
		"CONFIRMED_CLOSING_COUNT", "CONFIRMED_CLOSING_AMOUNT", "CALL_CUSTOMER_COUNT", "TRAINING_COUNT",
	}
	for _, code := range supported {
		if _, ok := supportedMetricQuery(code); !ok {
			t.Fatalf("expected %s to be supported", code)
		}
	}
	if _, ok := supportedMetricQuery("PARTNER_CALL_COUNT"); ok {
		t.Fatal("PARTNER_CALL_COUNT is explicitly out of scope for Sprint 13 and must not be supported")
	}
	if _, ok := supportedMetricQuery("UNKNOWN_METRIC"); ok {
		t.Fatal("unknown metric must not be supported")
	}
}

func TestConfirmedClosingAmountQueryExcludesUniqueTransferCode(t *testing.T) {
	query, ok := supportedMetricQuery("CONFIRMED_CLOSING_AMOUNT")
	if !ok {
		t.Fatal("CONFIRMED_CLOSING_AMOUNT harus didukung")
	}
	if !strings.Contains(query, "SUM(base_price - discount_amount + additional_charge)") {
		t.Fatalf("query harus menghitung omzet tanpa unique_transfer_code, got: %s", query)
	}
	if strings.Contains(query, "SUM(final_amount)") {
		t.Fatalf("query tidak boleh lagi memakai final_amount, got: %s", query)
	}
}
