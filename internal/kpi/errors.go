package kpi

import "errors"

var (
	ErrNotFound              = errors.New("kpi: data not found")
	ErrForbidden             = errors.New("kpi: forbidden")
	ErrInvalidMetric         = errors.New("kpi: unknown or inactive metric_code")
	ErrInvalidPeriod         = errors.New("kpi: period_month must be between 1 and 12")
	ErrInvalidWeight         = errors.New("kpi: weight must be a decimal between 0 and 100")
	ErrInvalidThreshold      = errors.New("kpi: threshold must be a decimal between 0 and 100, and threshold_near must not exceed threshold_achieved")
	ErrDuplicateDefinition   = errors.New("kpi: an active definition already exists for this metric and period")
	ErrNoActiveDefinitions   = errors.New("kpi: no active KPI definitions exist for this period")
	ErrWeightNotHundred      = errors.New("kpi: active KPI definitions for this period must have weights summing to exactly 100")
	ErrInconsistentThreshold = errors.New("kpi: active KPI definitions for this period must share the same threshold_achieved and threshold_near")
	ErrUnsupportedMetric     = errors.New("kpi: metric_code is not supported by the KPI recompute worker")
	ErrJobNotFound           = errors.New("kpi: recompute job not found")
)
