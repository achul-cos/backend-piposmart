package jobqueue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// execer is satisfied by both *sql.DB and *sql.Tx, so Complete/Fail can run standalone or as
// part of a caller-owned transaction (the KPI recompute handler commits its own domain writes
// together with the job's COMPLETED transition in one transaction).
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Repository provides job_queue persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new jobqueue Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const jobColumns = `id, job_type, payload, status, attempts, max_attempts, last_error,
	available_at, claimed_at, completed_at, created_by_user_id, created_at, updated_at`

func scanJob(row *sql.Row) (*Job, error) {
	var job Job
	if err := row.Scan(
		&job.ID, &job.JobType, &job.Payload, &job.Status, &job.Attempts, &job.MaxAttempts,
		&job.LastError, &job.AvailableAt, &job.ClaimedAt, &job.CompletedAt,
		&job.CreatedByUserID, &job.CreatedAt, &job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &job, nil
}

// Enqueue inserts a new PENDING job and returns its ID.
func (r *Repository) Enqueue(ctx context.Context, jobType string, payload any, createdByUserID *int64) (int64, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("jobqueue: marshal payload: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO job_queue (job_type, payload, created_by_user_id)
		VALUES (?, ?, ?)`,
		jobType, body, createdByUserID,
	)
	if err != nil {
		return 0, fmt.Errorf("jobqueue: enqueue: %w", err)
	}
	return result.LastInsertId()
}

// GetByID fetches a single job by ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Job, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM job_queue WHERE id = ?`, id)
	job, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jobqueue: get by id: %w", err)
	}
	return job, nil
}

// ClaimNext atomically claims the oldest available PENDING job (SKIP LOCKED, safe under
// concurrent worker instances) and increments its attempt counter. Returns (nil, nil) if no
// job is available. The claim (including the attempts increment) commits immediately and
// independently of whatever the caller's handler does next — a handler failure must not be
// able to roll back the attempts count, or retry limits would never be enforced.
func (r *Repository) ClaimNext(ctx context.Context) (*Job, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("jobqueue: begin claim tx: %w", err)
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM job_queue
		WHERE status = ? AND available_at <= NOW()
		ORDER BY id
		LIMIT 1
		FOR UPDATE SKIP LOCKED`,
		StatusPending,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jobqueue: select next job: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE job_queue SET status = ?, claimed_at = NOW(), attempts = attempts + 1
		WHERE id = ?`,
		StatusProcessing, id,
	); err != nil {
		return nil, fmt.Errorf("jobqueue: claim job %d: %w", id, err)
	}

	row := tx.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM job_queue WHERE id = ?`, id)
	job, err := scanJob(row)
	if err != nil {
		return nil, fmt.Errorf("jobqueue: reload claimed job %d: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("jobqueue: commit claim: %w", err)
	}
	return job, nil
}

// Complete marks a job COMPLETED. Safe to call within the caller's own domain transaction so
// the job's completion commits atomically with the work it triggered.
func (r *Repository) Complete(ctx context.Context, exec execer, jobID int64) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE job_queue SET status = ?, completed_at = NOW() WHERE id = ?`,
		StatusCompleted, jobID,
	)
	if err != nil {
		return fmt.Errorf("jobqueue: complete job %d: %w", jobID, err)
	}
	return nil
}

// Fail records a handler error against a job. If attempts remain, the job goes back to PENDING
// with a linear backoff; otherwise it is marked FAILED permanently.
func (r *Repository) Fail(ctx context.Context, exec execer, jobID int64, jobErr error) error {
	var attempts, maxAttempts int
	if err := exec.QueryRowContext(ctx, `
		SELECT attempts, max_attempts FROM job_queue WHERE id = ?`, jobID,
	).Scan(&attempts, &maxAttempts); err != nil {
		return fmt.Errorf("jobqueue: read attempts for job %d: %w", jobID, err)
	}

	message := jobErr.Error()
	if attempts < maxAttempts {
		backoff := time.Duration(attempts) * 30 * time.Second
		availableAt := time.Now().Add(backoff)
		_, err := exec.ExecContext(ctx, `
			UPDATE job_queue SET status = ?, available_at = ?, last_error = ? WHERE id = ?`,
			StatusPending, availableAt, message, jobID,
		)
		if err != nil {
			return fmt.Errorf("jobqueue: schedule retry for job %d: %w", jobID, err)
		}
		return nil
	}

	if _, err := exec.ExecContext(ctx, `
		UPDATE job_queue SET status = ?, last_error = ? WHERE id = ?`,
		StatusFailed, message, jobID,
	); err != nil {
		return fmt.Errorf("jobqueue: mark job %d failed: %w", jobID, err)
	}
	return nil
}

// ReclaimStale returns PROCESSING jobs stuck longer than staleTimeout back to PENDING (worker
// crashed mid-job). Returns the number of jobs reclaimed.
func (r *Repository) ReclaimStale(ctx context.Context, staleTimeout time.Duration) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE job_queue
		SET status = ?, last_error = 'reclaimed after stale timeout'
		WHERE status = ? AND claimed_at < ?`,
		StatusPending, StatusProcessing, time.Now().Add(-staleTimeout),
	)
	if err != nil {
		return 0, fmt.Errorf("jobqueue: reclaim stale jobs: %w", err)
	}
	return result.RowsAffected()
}
