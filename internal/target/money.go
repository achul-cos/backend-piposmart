package target

import (
	"strconv"
	"strings"
)

// validateDecimal checks value is a non-negative decimal with at most 2 fraction digits
// (matches the DECIMAL(18,2) column) without converting it — the raw string is stored as-is,
// same convention as internal/partner's commission_value handling.
func validateDecimal(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return ErrInvalidValue
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return ErrInvalidValue
	}
	if _, err := strconv.ParseInt(parts[0], 10, 64); err != nil {
		return ErrInvalidValue
	}
	if len(parts) == 2 {
		fraction := parts[1]
		if fraction == "" || len(fraction) > 2 {
			return ErrInvalidValue
		}
		if _, err := strconv.ParseInt(fraction, 10, 64); err != nil {
			return ErrInvalidValue
		}
	}
	return nil
}
