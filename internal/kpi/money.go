package kpi

import (
	"strconv"
	"strings"
)

// parsePercent parses a decimal percent string (e.g. "40.00") into hundredths of a percent
// (1.00% == 100) and validates it falls within [0, 100]. Mirrors
// internal/partner's parseCommissionRate — every domain package keeps its own small decimal
// helpers rather than sharing a central money package.
func parsePercent(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return 0, ErrInvalidWeight
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidWeight
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidWeight
	}
	hundredths := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if fraction == "" || len(fraction) > 2 {
			return 0, ErrInvalidWeight
		}
		if len(fraction) == 1 {
			fraction += "0"
		}
		hundredths, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, ErrInvalidWeight
		}
	}
	total := whole*100 + hundredths
	if total > 10000 {
		return 0, ErrInvalidWeight
	}
	return total, nil
}
