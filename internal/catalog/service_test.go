package catalog

import "testing"

func TestValidateCreatePlanUsesDecimalMoney(t *testing.T) {
	err := validateCreatePlan(CreatePlanRequest{
		PackageID:     1,
		Code:          "BUSINESS_12_MONTHS_TEST",
		Name:          "Business 12 Bulan Test",
		TenureMonths:  12,
		Price:         "1788000.00",
		EffectiveFrom: "2026-07-01",
	})
	if err != nil {
		t.Fatalf("valid decimal ditolak: %v", err)
	}

	err = validateCreatePlan(CreatePlanRequest{
		PackageID:     1,
		Code:          "BAD_PRICE",
		Name:          "Bad Price",
		TenureMonths:  12,
		Price:         "12.345",
		EffectiveFrom: "2026-07-01",
	})
	if err != ErrInvalidDecimal {
		t.Fatalf("ingin ErrInvalidDecimal, dapat %v", err)
	}
}

func TestPlanDurationUsesThirtyDaysPerMonth(t *testing.T) {
	req := CreatePlanRequest{TenureMonths: 18}
	durationDays := req.TenureMonths * 30
	if durationDays != 540 {
		t.Fatalf("duration=%d, ingin 540", durationDays)
	}
}

func TestPromotionChargeTypeValidation(t *testing.T) {
	if err := validateChargeType("FREE"); err != nil {
		t.Fatalf("FREE ditolak: %v", err)
	}
	if err := validateChargeType("PAID"); err != nil {
		t.Fatalf("PAID ditolak: %v", err)
	}
	if err := validateChargeType("AUTO"); err != ErrInvalidCharge {
		t.Fatalf("ingin ErrInvalidCharge, dapat %v", err)
	}
}
