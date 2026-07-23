package closing

import (
	"fmt"
	"strconv"
	"strings"
)

func parseMoneyToCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if strings.HasPrefix(value, "-") {
		return 0, ErrInvalidMoney
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidMoney
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidMoney
	}
	cents := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) > 2 || parts[1] == "" {
			return 0, ErrInvalidMoney
		}
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		cents, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, ErrInvalidMoney
		}
	}
	return whole*100 + cents, nil
}

func formatCents(value int64) string {
	if value < 0 {
		value = -value
		return fmt.Sprintf("-%d.%02d", value/100, value%100)
	}
	return fmt.Sprintf("%d.%02d", value/100, value%100)
}

func calculateFinalAmount(basePrice, discount, additionalCharge string, uniqueTransferCode int) (calculation, error) {
	if uniqueTransferCode < 0 || uniqueTransferCode > 999 {
		return calculation{}, ErrInvalidRequest
	}
	baseCents, err := parseMoneyToCents(basePrice)
	if err != nil {
		return calculation{}, err
	}
	discountCents, err := parseMoneyToCents(discount)
	if err != nil {
		return calculation{}, err
	}
	additionalCents, err := parseMoneyToCents(additionalCharge)
	if err != nil {
		return calculation{}, err
	}
	finalCents := baseCents - discountCents + additionalCents + int64(uniqueTransferCode*100)
	if finalCents < 0 {
		return calculation{}, ErrFinalAmountNegative
	}
	return calculation{
		BasePrice:          formatCents(baseCents),
		DiscountAmount:     formatCents(discountCents),
		AdditionalCharge:   formatCents(additionalCents),
		UniqueTransferCode: uniqueTransferCode,
		FinalAmount:        formatCents(finalCents),
	}, nil
}

type calculation struct {
	BasePrice          string
	DiscountAmount     string
	AdditionalCharge   string
	UniqueTransferCode int
	FinalAmount        string
}
