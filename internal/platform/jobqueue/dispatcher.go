package jobqueue

import (
	"context"
	"database/sql"
	"fmt"
)

// Dispatch claims and processes at most one pending job. It returns handled=true if a job was
// claimed (regardless of whether the handler succeeded or failed), so the caller's poll loop
// can decide whether to immediately try for another job or wait for the next tick.
//
// The handler runs inside its own transaction, separate from the claim (which already
// committed attempts/claimed_at in Repository.ClaimNext) — a handler failure only rolls back
// its own domain writes, never the retry bookkeeping.
func Dispatch(ctx context.Context, db *sql.DB, repository *Repository, registry Registry) (handled bool, err error) {
	job, err := repository.ClaimNext(ctx)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}

	handler, ok := registry[job.JobType]
	if !ok {
		failErr := repository.Fail(ctx, db, job.ID, fmt.Errorf("no handler registered for job_type %q", job.JobType))
		return true, failErr
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return true, fmt.Errorf("jobqueue: begin handler tx for job %d: %w", job.ID, err)
	}

	if handlerErr := handler(ctx, tx, job.ID, job.Payload); handlerErr != nil {
		_ = tx.Rollback()
		if failErr := repository.Fail(ctx, db, job.ID, handlerErr); failErr != nil {
			return true, fmt.Errorf("jobqueue: job %d failed (%v) and marking it failed also failed: %w", job.ID, handlerErr, failErr)
		}
		return true, nil
	}

	if err := repository.Complete(ctx, tx, job.ID); err != nil {
		_ = tx.Rollback()
		return true, err
	}
	if err := tx.Commit(); err != nil {
		return true, fmt.Errorf("jobqueue: commit job %d: %w", job.ID, err)
	}
	return true, nil
}
