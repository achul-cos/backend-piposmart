package wallet

import (
	"fmt"
	"strconv"
	"strings"
)

func parseMoneyToCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
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
		fraction := parts[1]
		if fraction == "" || len(fraction) > 2 {
			return 0, ErrInvalidMoney
		}
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

func parsePositiveMoneyToCents(value string) (int64, error) {
	cents, err := parseMoneyToCents(value)
	if err != nil {
		return 0, err
	}
	if cents <= 0 {
		return 0, ErrInvalidMoney
	}
	return cents, nil
}

func formatCents(value int64) string {
	if value < 0 {
		value = -value
		return fmt.Sprintf("-%d.%02d", value/100, value%100)
	}
	return fmt.Sprintf("%d.%02d", value/100, value%100)
}

func applyBalance(balanceCents, amountCents int64, direction string) (int64, error) {
	switch direction {
	case DirectionCredit:
		return balanceCents + amountCents, nil
	case DirectionDebit:
		next := balanceCents - amountCents
		if next < 0 {
			return 0, ErrInsufficientBalance
		}
		return next, nil
	default:
		return 0, ErrInvalidDirection
	}
}
