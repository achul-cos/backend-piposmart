package analytics

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	errInvalidTimeFilter   = errors.New("time filter tidak valid")
	errUnsupportedMode     = errors.New("mode time filter tidak didukung")
	errUnsupportedCompare  = errors.New("mode comparison tidak didukung")
	errInvalidGranularity  = errors.New("granularity tidak didukung")
)

func resolveTimeFilter(req TimeFilterRequest, now time.Time) (ResolvedTimeFilter, error) {
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "date_range"
	}
	granularity := strings.TrimSpace(req.Granularity)
	if granularity == "" {
		switch mode {
		case "year_range":
			granularity = "year"
		case "month_range":
			granularity = "month"
		default:
			granularity = "day"
		}
	}
	if !isGranularitySupported(granularity) {
		return ResolvedTimeFilter{}, errInvalidGranularity
	}

	switch mode {
	case "date_range":
		start, err := time.ParseInLocation("2006-01-02", req.DateFrom, time.UTC)
		if err != nil {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: date_from harus format YYYY-MM-DD", errInvalidTimeFilter)
		}
		endDate, err := time.ParseInLocation("2006-01-02", req.DateTo, time.UTC)
		if err != nil {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: date_to harus format YYYY-MM-DD", errInvalidTimeFilter)
		}
		end := endDate.AddDate(0, 0, 1)
		if !start.Before(end) {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: date_from harus lebih kecil dari atau sama dengan date_to", errInvalidTimeFilter)
		}
		return ResolvedTimeFilter{
			Mode:        mode,
			Granularity: granularity,
			Start:       start,
			End:         end,
			Label:       fmt.Sprintf("%s - %s", start.Format("02 Jan 2006"), end.Add(-time.Nanosecond).Format("02 Jan 2006")),
		}, nil
	case "month_range":
		start, err := parseMonthStart(req.MonthFrom)
		if err != nil {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: month_from harus format YYYY-MM", errInvalidTimeFilter)
		}
		endStart, err := parseMonthStart(req.MonthTo)
		if err != nil {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: month_to harus format YYYY-MM", errInvalidTimeFilter)
		}
		end := endStart.AddDate(0, 1, 0)
		if !start.Before(end) {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: month_from harus lebih kecil dari atau sama dengan month_to", errInvalidTimeFilter)
		}
		return ResolvedTimeFilter{
			Mode:        mode,
			Granularity: granularity,
			Start:       start,
			End:         end,
			Label:       fmt.Sprintf("%s - %s", start.Format("Jan 2006"), end.Add(-time.Nanosecond).Format("Jan 2006")),
		}, nil
	case "year_range":
		if req.YearFrom == nil || req.YearTo == nil {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: year_from dan year_to wajib diisi", errInvalidTimeFilter)
		}
		start := time.Date(*req.YearFrom, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(*req.YearTo+1, 1, 1, 0, 0, 0, 0, time.UTC)
		if !start.Before(end) {
			return ResolvedTimeFilter{}, fmt.Errorf("%w: year_from harus lebih kecil dari atau sama dengan year_to", errInvalidTimeFilter)
		}
		return ResolvedTimeFilter{
			Mode:        mode,
			Granularity: granularity,
			Start:       start,
			End:         end,
			Label:       fmt.Sprintf("%d - %d", *req.YearFrom, *req.YearTo),
		}, nil
	default:
		if mode == "" {
			return resolveTimeFilter(TimeFilterRequest{
				Mode:        "date_range",
				DateFrom:    now.Format("2006-01-02"),
				DateTo:      now.Format("2006-01-02"),
				Granularity: "day",
			}, now)
		}
		return ResolvedTimeFilter{}, errUnsupportedMode
	}
}

func resolveComparison(current ResolvedTimeFilter, req ComparisonRequest, now time.Time) (ComparisonSummary, *ResolvedTimeFilter, error) {
	if !req.Enabled {
		return ComparisonSummary{Enabled: false}, nil, nil
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "previous_period"
	}
	var baseline ResolvedTimeFilter
	switch mode {
	case "previous_period":
		duration := current.End.Sub(current.Start)
		baseline = ResolvedTimeFilter{
			Mode:        current.Mode,
			Granularity: current.Granularity,
			Start:       current.Start.Add(-duration),
			End:         current.Start,
		}
	case "previous_month":
		baseline = ResolvedTimeFilter{
			Mode:        current.Mode,
			Granularity: current.Granularity,
			Start:       current.Start.AddDate(0, -1, 0),
			End:         current.End.AddDate(0, -1, 0),
		}
	case "previous_year":
		baseline = ResolvedTimeFilter{
			Mode:        current.Mode,
			Granularity: current.Granularity,
			Start:       current.Start.AddDate(-1, 0, 0),
			End:         current.End.AddDate(-1, 0, 0),
		}
	case "custom_period":
		if req.BaselineTimeFilter == nil {
			return ComparisonSummary{}, nil, fmt.Errorf("%w: baseline_time_filter wajib diisi", errUnsupportedCompare)
		}
		resolved, err := resolveTimeFilter(*req.BaselineTimeFilter, now)
		if err != nil {
			return ComparisonSummary{}, nil, err
		}
		baseline = resolved
	default:
		return ComparisonSummary{}, nil, errUnsupportedCompare
	}
	baseline.Label = humanLabel(baseline)
	return ComparisonSummary{
		Enabled:       true,
		Mode:          mode,
		BaselineLabel: baseline.Label,
	}, &baseline, nil
}

func isGranularitySupported(value string) bool {
	switch value {
	case "day", "week", "month", "quarter", "year":
		return true
	default:
		return false
	}
}

func parseMonthStart(value string) (time.Time, error) {
	parsed, err := time.ParseInLocation("2006-01", strings.TrimSpace(value), time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func humanLabel(value ResolvedTimeFilter) string {
	switch value.Mode {
	case "month_range":
		return fmt.Sprintf("%s - %s", value.Start.Format("Jan 2006"), value.End.Add(-time.Nanosecond).Format("Jan 2006"))
	case "year_range":
		return fmt.Sprintf("%d - %d", value.Start.Year(), value.End.Add(-time.Nanosecond).Year())
	default:
		return fmt.Sprintf("%s - %s", value.Start.Format("02 Jan 2006"), value.End.Add(-time.Nanosecond).Format("02 Jan 2006"))
	}
}
