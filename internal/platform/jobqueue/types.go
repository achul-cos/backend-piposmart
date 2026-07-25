// Package jobqueue implements a generic MySQL-backed job queue (no Redis, per the project's
// locked technical decision). It knows nothing about specific job types — domain packages
// (e.g. internal/kpi) register a Handler for their job_type and enqueue payloads; this package
// only owns claim/retry/stale-reclaim mechanics.
package jobqueue

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"
)

// Job is one row of job_queue.
type Job struct {
	ID              int64
	JobType         string
	Payload         json.RawMessage
	Status          string
	Attempts        int
	MaxAttempts     int
	LastError       sql.NullString
	AvailableAt     time.Time
	ClaimedAt       sql.NullTime
	CompletedAt     sql.NullTime
	CreatedByUserID sql.NullInt64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Handler processes one claimed job's payload inside the same transaction it was claimed in.
// jobID is passed through for handlers that want to record traceability back to the job row
// (e.g. KPI results snapshot which job produced them). Returning an error marks the job for
// retry (or FAILED once MaxAttempts is exhausted).
type Handler func(ctx context.Context, tx *sql.Tx, jobID int64, payload json.RawMessage) error

// Registry maps job_type to its Handler.
type Registry map[string]Handler
