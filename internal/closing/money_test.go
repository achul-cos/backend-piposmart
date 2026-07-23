package closing

import (
	"errors"
	"testing"
)

func TestCalculateFinalAmount(t *testing.T) {
	result, err := calculateFinalAmount("1698600.00", "100000.00", "0.00", 123)
	if err != nil {
		t.Fatalf("calculate final amount: %v", err)
	}
	if result.BasePrice != "1698600.00" {
		t.Fatalf("base price = %s", result.BasePrice)
	}
	if result.DiscountAmount != "100000.00" {
		t.Fatalf("discount = %s", result.DiscountAmount)
	}
	if result.FinalAmount != "1598723.00" {
		t.Fatalf("final amount = %s, want 1598723.00", result.FinalAmount)
	}
}

func TestCalculateFinalAmountRejectsInvalidDecimal(t *testing.T) {
	_, err := calculateFinalAmount("1000.000", "0.00", "0.00", 0)
	if !errors.Is(err, ErrInvalidMoney) {
		t.Fatalf("err = %v, want ErrInvalidMoney", err)
	}
}

func TestCalculateFinalAmountRejectsNegativeFinal(t *testing.T) {
	_, err := calculateFinalAmount("1000.00", "2000.00", "0.00", 0)
	if !errors.Is(err, ErrFinalAmountNegative) {
		t.Fatalf("err = %v, want ErrFinalAmountNegative", err)
	}
}

func TestCalculateFinalAmountRejectsInvalidUniqueTransferCode(t *testing.T) {
	_, err := calculateFinalAmount("1000.00", "0.00", "0.00", 1000)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}
