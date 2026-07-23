package subscription

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
		fraction := parts[1]
		if len(fraction) > 2 {
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
	return fmt.Sprintf("%d.%02d", value/100, value%100)
}
