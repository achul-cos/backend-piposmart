package kpi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"backend_crm_piposmart/internal/platform/jobqueue"
)

// RecomputeHandler adapts Repository.Recompute to the jobqueue.Handler signature, so
// internal/app can register it into the worker's job registry without internal/platform/jobqueue
// needing to know anything about KPI.
func RecomputeHandler(repo *Repository) jobqueue.Handler {
	return func(ctx context.Context, tx *sql.Tx, jobID int64, payload json.RawMessage) error {
		var p RecomputeJobPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("kpi: unmarshal recompute payload: %w", err)
		}
		if p.PeriodMonth < 1 || p.PeriodMonth > 12 {
			return ErrInvalidPeriod
		}
		id := jobID
		return repo.Recompute(ctx, tx, p.PeriodYear, p.PeriodMonth, &id)
	}
}
