package partner

import (
	"fmt"
	"strconv"
	"strings"
)

// parseMoneyToCents parses a non-negative decimal money string (e.g. "150000.00")
// into integer cents, avoiding float rounding issues.
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

func formatCents(value int64) string {
	if value < 0 {
		value = -value
		return fmt.Sprintf("-%d.%02d", value/100, value%100)
	}
	return fmt.Sprintf("%d.%02d", value/100, value%100)
}

// parseCommissionRate parses a decimal percent string (e.g. "5.25") into hundredths
// of a percent (1.00% == 100), and validates it falls within [0, 100].
func parseCommissionRate(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return 0, ErrInvalidCommissionRate
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidCommissionRate
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidCommissionRate
	}
	hundredths := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if fraction == "" || len(fraction) > 2 {
			return 0, ErrInvalidCommissionRate
		}
		if len(fraction) == 1 {
			fraction += "0"
		}
		hundredths, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, ErrInvalidCommissionRate
		}
	}
	total := whole*100 + hundredths
	if total > 10000 { // > 100.00%
		return 0, ErrInvalidCommissionRate
	}
	return total, nil
}

// calculateCommissionCents computes commission_amount cents from a base amount in
// cents and a rate expressed in hundredths-of-a-percent (see parseCommissionRate),
// rounding half up to the nearest cent.
func calculateCommissionCents(baseCents int64, rateHundredths int64) int64 {
	numerator := baseCents * rateHundredths
	return (numerator + 5000) / 10000
}

// validateCommissionValue checks that a partner_type's commission_value string is a
// valid decimal for the given commission_mode: a percent in [0, 100] for PERCENTAGE, or
// a non-negative money amount for FIXED.
func validateCommissionValue(mode string, value string) error {
	switch mode {
	case CommissionModePercentage:
		_, err := parseCommissionRate(value)
		return err
	case CommissionModeFixed:
		_, err := parseMoneyToCents(value)
		return err
	default:
		return ErrInvalidCommissionMode
	}
}

// calculateCommissionAmountCents computes commission owed in cents for a confirmed
// closing, given the partner_type's commission_mode/commission_value and the closing's
// final_amount in cents. FIXED commissions are paid in full regardless of the closing's
// size (not capped by baseCents).
func calculateCommissionAmountCents(mode string, value string, baseCents int64) (int64, error) {
	switch mode {
	case CommissionModePercentage:
		rateHundredths, err := parseCommissionRate(value)
		if err != nil {
			return 0, err
		}
		return calculateCommissionCents(baseCents, rateHundredths), nil
	case CommissionModeFixed:
		return parseMoneyToCents(value)
	default:
		return 0, ErrInvalidCommissionMode
	}
}
