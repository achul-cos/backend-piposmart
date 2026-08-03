package importing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	ib.id, ib.code, ib.profile, ib.sheet_name, ib.target_sales_user_id, ib.original_filename, ib.file_sha256, ib.file_path, ib.status,
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
		&b.ID, &b.Code, &b.Profile, &b.SheetName, &b.TargetSalesUserID, &b.OriginalFilename, &b.FileSHA256, &b.FilePath, &b.Status,
		&b.TotalRows, &b.ValidRows, &b.InvalidRows, &b.CommittedRows, &b.ProgressPercentage,
		&b.ValidateJobID, &b.CommitJobID, &b.ErrorMessage,
		&b.UploadedByUserID, &b.UploadedByName, &b.CommittedByUserID, &b.CommittedByName,
		&b.UploadedAt, &b.ValidatedAt, &b.CommittedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	return b, err
}

// FindBatchBySHA256AndProfile returns the existing batch for a (file hash, profile, sheet_name)
// combination, if any (upload idempotency). Scoped beyond the file hash alone because
// SALES_CALL_CHAT/SALES_TARGET deliberately re-upload the same physical file once per
// profile+sheet — deduping on hash alone would make the second upload return the first batch
// under the wrong profile.
func (r *Repository) FindBatchBySHA256AndProfile(ctx context.Context, hash, profile, sheetName string) (*ImportBatch, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+batchSelectColumns+batchFromJoin+` WHERE ib.file_sha256 = ? AND ib.profile = ? AND ib.sheet_name = ?`,
		hash, profile, sheetName,
	)
	b, err := scanBatch(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("importing: find batch by sha256/profile/sheet: %w", err)
	}
	return &b, nil
}

// CreateBatch inserts a new UPLOADED batch and returns its ID. sheetName is empty for profiles
// that auto-detect their sheet; required (enforced in Service.Upload) for SALES_CALL_CHAT/
// SALES_TARGET, whose workbooks have several structurally-identical sheets. targetSalesUserID is
// nil unless the same two profiles require it (the sales rep is only encoded in the sheet name).
func (r *Repository) CreateBatch(ctx context.Context, code, profile, sheetName string, targetSalesUserID *int64, originalFilename, sha256, filePath string, fileBlob []byte, uploadedByUserID int64) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO import_batches (code, profile, sheet_name, target_sales_user_id, original_filename, file_sha256, file_path, file_blob, uploaded_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		code, profile, sheetName, targetSalesUserID, originalFilename, sha256, filePath, fileBlob, uploadedByUserID,
	)
	if err != nil {
		return 0, fmt.Errorf("importing: create batch: %w", err)
	}
	return result.LastInsertId()
}

func (r *Repository) GetBatchFileSource(ctx context.Context, id int64) (ImportBatch, error) {
	var item ImportBatch
	row := r.db.QueryRowContext(ctx, `
		SELECT id, original_filename, file_path, file_blob
		FROM import_batches
		WHERE id = ?`, id)
	if err := row.Scan(&item.ID, &item.OriginalFilename, &item.FilePath, &item.FileBlob); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ImportBatch{}, ErrNotFound
		}
		return ImportBatch{}, fmt.Errorf("importing: get batch file source %d: %w", id, err)
	}
	return item, nil
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
	query := `SELECT ` + batchSelectColumns + batchFromJoin + ` ` + whereClause + `
		ORDER BY ib.id DESC`
	queryArgs := append([]any{}, args...)
	if !params.All {
		offset := (page - 1) * limit
		query += ` LIMIT ? OFFSET ?`
		queryArgs = append(queryArgs, limit, offset)
	}
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
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

// GetBatchStatusCounts powers GET /imports/summary — a count of batches per status, so an admin
// doesn't have to page through GET /imports?status=... one status at a time to know how many
// batches currently need attention (VALIDATION_FAILED/COMMIT_FAILED especially).
func (r *Repository) GetBatchStatusCounts(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM import_batches GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("importing: count batch statuses: %w", err)
	}
	defer rows.Close()
	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
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
func clampProgress(percentage int) int {
	if percentage < 0 {
		return 0
	}
	if percentage > 100 {
		return 100
	}
	return percentage
}

func (r *Repository) UpdateProgress(ctx context.Context, batchID int64, percentage int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE import_batches SET progress_percentage = ? WHERE id = ?`, clampProgress(percentage), batchID)
	return err
}

// UpdateProgressWithExec mirrors UpdateProgress but writes through the caller's execer (usually
// the worker transaction) to avoid self-deadlocking on the same batch row during validation.
func (r *Repository) UpdateProgressWithExec(ctx context.Context, exec execer, batchID int64, percentage int) error {
	_, err := exec.ExecContext(ctx, `UPDATE import_batches SET progress_percentage = ? WHERE id = ?`, clampProgress(percentage), batchID)
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

// FindRowByID fetches a single row scoped to batchID — the relink handler uses this to confirm
// the row actually belongs to the batch in the URL before mutating it.
func (r *Repository) FindRowByID(ctx context.Context, batchID, rowID int64) (ImportRow, error) {
	row, err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT `+rowSelectColumns+` FROM import_rows WHERE id = ? AND batch_id = ?`, rowID, batchID))
	if errors.Is(err, sql.ErrNoRows) {
		return ImportRow{}, ErrNotFound
	}
	return row, err
}

// RelinkRow is the manual resolution for a reconciliation candidate (RowStatusUnmatched): admin
// supplies the entity ID(s) the row's own data couldn't resolve automatically (owner/outlet/lead
// code didn't match anything at commit time), moving the row back to VALID so the next batch
// commit picks it up again. Deliberately does NOT touch raw_payload — the row's original parsed
// data is unchanged, only which entities it resolves to.
func (r *Repository) RelinkRow(ctx context.Context, rowID int64, ownerID, outletID, leadID *int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE import_rows
		SET status = ?, owner_id = COALESCE(?, owner_id), outlet_id = COALESCE(?, outlet_id),
		    lead_id = COALESCE(?, lead_id), commit_error = NULL
		WHERE id = ?`,
		RowStatusValid, ownerID, outletID, leadID, rowID,
	)
	return err
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
	query := `SELECT ` + rowSelectColumns + ` FROM import_rows ` + whereClause + `
		ORDER BY row_index ASC`
	queryArgs := append([]any{}, args...)
	if !params.All {
		offset := (page - 1) * limit
		query += ` LIMIT ? OFFSET ?`
		queryArgs = append(queryArgs, limit, offset)
	}
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
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

// MarkRowCommittedNoEntity marks a row committed without an owner/outlet/lead link — for profiles
// whose target entity is neither (e.g. SALES_TARGET, which commits against a Sales user via
// target_sales_user_id on the batch, not any of these three FK columns).
func (r *Repository) MarkRowCommittedNoEntity(ctx context.Context, rowID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE import_rows SET status = ? WHERE id = ?`,
		RowStatusCommitted, rowID,
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

// MarkRowUnmatched records a structurally-valid row whose owner/outlet/partner/package/sales-user
// reference wasn't found — a reconciliation candidate, not a hard failure. message explains what
// was missing (e.g. "owner code 4918 not found"), surfaced the same way commit_error is.
func (r *Repository) MarkRowUnmatched(ctx context.Context, rowID int64, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE import_rows SET status = ?, commit_error = ? WHERE id = ?`,
		RowStatusUnmatched, message, rowID,
	)
	return err
}

// FindOutletIDByOwnerAndName looks up an existing outlet for an owner by name (case-insensitive) —
// used by profiles that identify the outlet by its display name rather than its code (e.g.
// NEW_SUBSCRIBE's "Project/Outlet" column), unlike OWNER_OUTLET which always creates the outlet
// alongside its owner and therefore never needs to look one up by name.
func (r *Repository) FindOutletIDByOwnerAndName(ctx context.Context, ownerID int64, name string) (int64, bool, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM outlets
		WHERE owner_id = ? AND deleted_at IS NULL AND LOWER(name) = LOWER(?)
		ORDER BY id ASC LIMIT 1`,
		ownerID, name,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("importing: find outlet by owner+name: %w", err)
	}
	return id, true, nil
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
func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func (r *Repository) UpsertOutletMonthlyActivitySnapshot(ctx context.Context, batchID int64, outletID int64, activity monthlyActiveEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO outlet_monthly_activity_snapshot (
			outlet_id, period_year, period_month, raw_code, category, package_code, tenor_months, import_batch_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			raw_code = VALUES(raw_code),
			category = VALUES(category),
			package_code = VALUES(package_code),
			tenor_months = VALUES(tenor_months),
			import_batch_id = VALUES(import_batch_id)`,
		outletID,
		activity.PeriodYear,
		activity.PeriodMonth,
		activity.RawCode,
		activity.Category,
		nullableString(activity.PackageCode),
		nullableInt(activity.TenorMonths),
		batchID,
	)
	if err != nil {
		return fmt.Errorf("importing: upsert monthly activity snapshot outlet=%d %04d-%02d: %w", outletID, activity.PeriodYear, activity.PeriodMonth, err)
	}
	return nil
}

func (r *Repository) UpsertPartnerBonusReferralSnapshot(ctx context.Context, batchID int64, sourceRowIndex int, referredOwnerID int64, referredOutletID, referredLeadID *int64, row bonusMitraRow) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO partner_bonus_referral_snapshots (
			import_batch_id, source_row_index,
			partner_name, partner_owner_code, partner_owner_name, partner_brand_name, partner_category,
			referred_owner_code, referred_owner_name, referred_project_name, referred_outlet_name,
			package_name, sales_pic_name, top_up_date, renewal_date, payout_date_1, payout_date_2,
			subscription_status, payout_status, cmo_name,
			unpaid_amount, stage1_amount, stage2_amount, paid_amount, total_amount,
			referred_owner_id, referred_outlet_id, referred_lead_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			partner_name = VALUES(partner_name),
			partner_owner_code = VALUES(partner_owner_code),
			partner_owner_name = VALUES(partner_owner_name),
			partner_brand_name = VALUES(partner_brand_name),
			partner_category = VALUES(partner_category),
			referred_owner_code = VALUES(referred_owner_code),
			referred_owner_name = VALUES(referred_owner_name),
			referred_project_name = VALUES(referred_project_name),
			referred_outlet_name = VALUES(referred_outlet_name),
			package_name = VALUES(package_name),
			sales_pic_name = VALUES(sales_pic_name),
			top_up_date = VALUES(top_up_date),
			renewal_date = VALUES(renewal_date),
			payout_date_1 = VALUES(payout_date_1),
			payout_date_2 = VALUES(payout_date_2),
			subscription_status = VALUES(subscription_status),
			payout_status = VALUES(payout_status),
			cmo_name = VALUES(cmo_name),
			unpaid_amount = VALUES(unpaid_amount),
			stage1_amount = VALUES(stage1_amount),
			stage2_amount = VALUES(stage2_amount),
			paid_amount = VALUES(paid_amount),
			total_amount = VALUES(total_amount),
			referred_owner_id = VALUES(referred_owner_id),
			referred_outlet_id = VALUES(referred_outlet_id),
			referred_lead_id = VALUES(referred_lead_id)`,
		batchID,
		sourceRowIndex,
		row.PartnerName,
		nullableString(row.PartnerOwnerCode),
		nullableString(row.PartnerOwnerName),
		nullableString(row.PartnerBrandName),
		nullableString(row.PartnerCategory),
		row.ReferredOwnerCode,
		nullableString(row.ReferredOwnerName),
		nullableString(row.ReferredProject),
		nullableString(row.ReferredOutletName),
		nullableString(row.PackageName),
		nullableString(row.SalesPICName),
		nullableString(row.TopUpDate),
		nullableString(row.RenewalDate),
		nullableString(row.PayoutDate1),
		nullableString(row.PayoutDate2),
		nullableString(row.SubscriptionStatus),
		nullableString(row.PayoutStatus),
		nullableString(row.CMOName),
		row.UnpaidAmount,
		row.Stage1Amount,
		row.Stage2Amount,
		row.PaidAmount,
		row.TotalAmount,
		referredOwnerID,
		referredOutletID,
		referredLeadID,
	)
	if err != nil {
		return fmt.Errorf("importing: upsert partner bonus snapshot batch=%d row=%d: %w", batchID, sourceRowIndex, err)
	}
	return nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
