package target

import "errors"

var (
	ErrNotFound         = errors.New("target: data not found")
	ErrForbidden        = errors.New("target: forbidden")
	ErrInvalidMetric    = errors.New("target: unknown or inactive metric_code")
	ErrInvalidPeriod    = errors.New("target: period_month must be between 1 and 12")
	ErrInvalidValue     = errors.New("target: target_value must be a non-negative decimal")
	ErrSalesNotEligible = errors.New("target: sales_id is not an active Sales user")
)
