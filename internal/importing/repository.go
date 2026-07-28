package importing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Repository provides import_batches/import_rows persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new importing Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const batchSelectColumns = `
	ib.id, ib.code, ib.profile, ib.original_filename, ib.file_sha256, ib.file_path, ib.status,
	ib.total_rows, ib.valid_rows, ib.invalid_rows, ib.committed_rows, ib.progress_percentage,
	ib.validate_job_id, ib.commit_job_id, ib.error_message,
	ib.uploaded_by_user_id, ub.name, ib.committed_by_user_id, cb.name,
	ib.uploaded_at, ib.validated_at, ib.committed_at, ib.created_at, ib.updated_at`

const batchFromJoin = `
	FROM import_batches ib
	JOIN users ub ON ub.id = ib.uploaded_by_user_id
	LEFT JOIN users cb ON cb.id = ib.committed_by_user_id`

func scanBatch(scanner interface {
	Scan(dest ...any) error
}) (ImportBatch, error) {
	var b ImportBatch
	err := scanner.Scan(
		&b.ID, &b.Code, &b.Profile, &b.OriginalFilename, &b.FileSHA256, &b.FilePath, &b.Status,
		&b.TotalRows, &b.ValidRows, &b.InvalidRows, &b.CommittedRows, &b.ProgressPercentage,
		&b.ValidateJobID, &b.CommitJobID, &b.ErrorMessage,
		&b.UploadedByUserID, &b.UploadedByName, &b.CommittedByUserID, &b.CommittedByName,
		&b.UploadedAt, &b.ValidatedAt, &b.CommittedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	return b, err
}

// FindBatchBySHA256 returns the existing batch for a file hash, if any (upload idempotency).
func (r *Repository) FindBatchBySHA256(ctx context.Context, hash string) (*ImportBatch, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+batchSelectColumns+batchFromJoin+` WHERE ib.file_sha256 = ?`, hash)
	b, err := scanBatch(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("importing: find batch by sha256: %w", err)
	}
	return &b, nil
}

// CreateBatch inserts a new UPLOADED batch and returns its ID.
func (r *Repository) CreateBatch(ctx context.Context, code, profile, originalFilename, sha256, filePath string, uploadedByUserID int64) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO import_batches (code, profile, original_filename, file_sha256, file_path, uploaded_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		code, profile, originalFilename, sha256, filePath, uploadedByUserID,
	)
	if err != nil {
		return 0, fmt.Errorf("importing: create batch: %w", err)
	}
	return result.LastInsertId()
}

func (r *Repository) GetBatchByID(ctx context.Context, id int64) (*ImportBatch, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+batchSelectColumns+batchFromJoin+` WHERE ib.id = ?`, id)
	b, err := scanBatch(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("importing: get batch %d: %w", id, err)
	}
	return &b, nil
}

func (r *Repository) ListBatches(ctx context.Context, params ListBatchesParams) ([]ImportBatch, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if params.Status != "" {
		where = append(where, "ib.status = ?")
		args = append(args, params.Status)
	}
	if params.Profile != "" {
		where = append(where, "ib.profile = ?")
		args = append(args, params.Profile)
	}
	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) `+batchFromJoin+` `+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("importing: count batches: %w", err)
	}

	page, limit := params.Page, params.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := `SELECT ` + batchSelectColumns + batchFromJoin + ` ` + whereClause + `
		ORDER BY ib.id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("importing: list batches: %w", err)
	}
	defer rows.Close()

	var items []ImportBatch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, b)
	}
	return items, total, rows.Err()
}

func (r *Repository) SetValidateJobID(ctx context.Context, batchID, jobID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE import_batches SET validate_job_id = ? WHERE id = ?`, jobID, batchID)
	return err
}

func (r *Repository) SetCommitJobID(ctx context.Context, batchID, jobID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE import_batches SET commit_job_id = ? WHERE id = ?`, jobID, batchID)
	return err
}

// SetBatchValidating marks the batch as being processed by the validate job.
func (r *Repository) SetBatchValidating(ctx context.Context, exec execer, batchID int64) error {
	_, err := exec.ExecContext(ctx, `UPDATE import_batches SET status = ?, progress_percentage = 0 WHERE id = ?`, BatchStatusValidating, batchID)
	return err
}

// UpdateProgress updates the batch's progress_percentage during processing.
func (r *Repository) UpdateProgress(ctx context.Context, batchID int64, percentage int) error {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}
	_, err := r.db.ExecContext(ctx, `UPDATE import_batches SET progress_percentage = ? WHERE id = ?`, percentage, batchID)
	return err
}

// SetValidationResult records the outcome of a successful validation pass and sets progress to 100%.
func (r *Repository) SetValidationResult(ctx context.Context, exec execer, batchID int64, total, valid, invalid int) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE import_batches
		SET status = ?, total_rows = ?, valid_rows = ?, invalid_rows = ?, progress_percentage = 100, validated_at = NOW()
		WHERE id = ?`,
		BatchStatusValidated, total, valid, invalid, batchID,
	)
	return err
}

// SetValidationFailed records that the batch could not be parsed/validated at all (e.g. corrupt
// file, no profile detected).
func (r *Repository) SetValidationFailed(ctx context.Context, exec execer, batchID int64, message string) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE import_batches SET status = ?, error_message = ? WHERE id = ?`,
		BatchStatusValidationFailed, message, batchID,
	)
	return err
}

// SetBatchProfile persists the (auto-detected or verified) profile once known.
func (r *Repository) SetBatchProfile(ctx context.Context, exec execer, batchID int64, profile string) error {
	_, err := exec.ExecContext(ctx, `UPDATE import_batches SET profile = ? WHERE id = ?`, profile, batchID)
	return err
}

func (r *Repository) SetBatchCommitting(ctx context.Context, batchID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE import_batches SET status = ?, progress_percentage = 0 WHERE id = ?`, BatchStatusCommitting, batchID)
	return err
}

// SetCommitResult records the outcome of a commit pass and sets progress to 100%.
func (r *Repository) SetCommitResult(ctx context.Context, exec execer, batchID int64, committedRows int, committedByUserID int64) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE import_batches
		SET status = ?, committed_rows = ?, committed_by_user_id = ?, progress_percentage = 100, committed_at = NOW()
		WHERE id = ?`,
		BatchStatusCommitted, committedRows, committedByUserID, batchID,
	)
	return err
}

/* ---------- Rows ---------- */

const rowSelectColumns = `
	id, batch_id, row_index, raw_payload, status, validation_errors,
	owner_id, outlet_id, lead_id, commit_error, created_at, updated_at`

func scanRow(scanner interface {
	Scan(dest ...any) error
}) (ImportRow, error) {
	var row ImportRow
	err := scanner.Scan(
		&row.ID, &row.BatchID, &row.RowIndex, &row.RawPayload, &row.Status, &row.ValidationErrors,
		&row.OwnerID, &row.OutletID, &row.LeadID, &row.CommitError, &row.CreatedAt, &row.UpdatedAt,
	)
	return row, err
}

// InsertRow inserts one parsed Excel row. payload and validationErrors are marshaled to JSON;
// validationErrors may be nil for VALID rows.
func (r *Repository) InsertRow(ctx context.Context, exec execer, batchID int64, rowIndex int, status string, payload any, validationErrors []string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("importing: marshal row payload: %w", err)
	}
	var errJSON []byte
	if len(validationErrors) > 0 {
		errJSON, err = json.Marshal(validationErrors)
		if err != nil {
			return fmt.Errorf("importing: marshal validation errors: %w", err)
		}
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO import_rows (batch_id, row_index, raw_payload, status, validation_errors)
		VALUES (?, ?, ?, ?, ?)`,
		batchID, rowIndex, payloadJSON, status, nullableJSON(errJSON),
	)
	if err != nil {
		return fmt.Errorf("importing: insert row %d: %w", rowIndex, err)
	}
	return nil
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func (r *Repository) ListRows(ctx context.Context, batchID int64, params ListRowsParams) ([]ImportRow, int64, error) {
	where := []string{"batch_id = ?"}
	args := []any{batchID}
	if params.Status != "" {
		where = append(where, "status = ?")
		args = append(args, params.Status)
	}
	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM import_rows `+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("importing: count rows: %w", err)
	}

	page, limit := params.Page, params.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit

	query := `SELECT ` + rowSelectColumns + ` FROM import_rows ` + whereClause + `
		ORDER BY row_index ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("importing: list rows: %w", err)
	}
	defer rows.Close()

	var items []ImportRow
	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	return items, total, rows.Err()
}

// ListRowsByStatus returns every row of the given status for a batch, unpaginated — used by the
// commit worker (VALID) and the rejected-rows export (INVALID).
func (r *Repository) ListRowsByStatus(ctx context.Context, batchID int64, status string) ([]ImportRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+rowSelectColumns+` FROM import_rows WHERE batch_id = ? AND status = ? ORDER BY row_index ASC`,
		batchID, status,
	)
	if err != nil {
		return nil, fmt.Errorf("importing: list rows by status: %w", err)
	}
	defer rows.Close()
	var items []ImportRow
	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func (r *Repository) MarkRowCommitted(ctx context.Context, rowID int64, ownerID int64, outletID, leadID *int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE import_rows SET status = ?, owner_id = ?, outlet_id = ?, lead_id = ? WHERE id = ?`,
		RowStatusCommitted, ownerID, outletID, leadID, rowID,
	)
	return err
}

func (r *Repository) MarkRowCommitFailed(ctx context.Context, rowID int64, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE import_rows SET status = ?, commit_error = ? WHERE id = ?`,
		RowStatusCommitFailed, message, rowID,
	)
	return err
}

// FindOwnerIDByCode looks up an existing owner by code, for OWNER_OUTLET rows that reference an
// owner already created earlier in the same (or a previous) batch.
func (r *Repository) FindOwnerIDByCode(ctx context.Context, code string) (int64, bool, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM owners WHERE code = ? AND deleted_at IS NULL`, code).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("importing: find owner by code: %w", err)
	}
	return id, true, nil
}

// CountOutletsForOwner is used to generate a deterministic outlet code (<owner_code>-OUT-<NN>)
// for the OWNER_OUTLET profile, since the source workbook has no explicit outlet code column.
func (r *Repository) CountOutletsForOwner(ctx context.Context, ownerID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outlets WHERE owner_id = ? AND deleted_at IS NULL`, ownerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("importing: count outlets for owner %d: %w", ownerID, err)
	}
	return count, nil
}

// FindLeadIDByOwnerID looks up an owner's existing lead. Lead codes are deterministically derived
// from owner_id (see internal/lead/repository.go, "LEAD-%06d"), which structurally means at most
// one lead can ever exist per owner — a real-world duplicate phone number across two Non-Register
// rows would otherwise try to create a second lead for the same (reused) owner and collide on
// that code. Checking first makes re-encountering the same phone idempotent instead of a
// COMMIT_FAILED row.
func (r *Repository) FindLeadIDByOwnerID(ctx context.Context, ownerID int64) (int64, bool, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM customer_leads WHERE owner_id = ? LIMIT 1`, ownerID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("importing: find lead by owner: %w", err)
	}
	return id, true, nil
}

// execer is satisfied by *sql.DB and *sql.Tx — kept minimal on purpose, mirroring
// internal/platform/jobqueue's execer, since import status updates run both standalone and
// inside the job handler's transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
