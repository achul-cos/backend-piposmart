package partner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Repository defines the data access layer for partner entities.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Partner repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type queryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

/* ---------- PartnerType ---------- */

func (r *Repository) CreatePartnerType(ctx context.Context, pt PartnerType) (int64, error) {
	query := `
		INSERT INTO partner_types (code, name, commission_mode, commission_value, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, pt.Code, pt.Name, pt.CommissionMode, pt.CommissionValue, pt.Description, pt.CreatedAt, pt.UpdatedAt)
	if err != nil {
		return 0, mapDuplicateError(err, "uq_partner_types_code")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CountPartnerTypes(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM partner_types").Scan(&count)
	return count, err
}

func (r *Repository) DeletePartnerType(ctx context.Context, id int64) error {
	// Block delete if any partners still reference this type
	var count int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM partners WHERE partner_type_id = ?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrPartnerTypeInUse
	}

	// Delete commission rules first
	if _, err := r.db.ExecContext(ctx, "DELETE FROM commission_rules WHERE partner_type_id = ?", id); err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM partner_types WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetPartnerTypeByID(ctx context.Context, id int64) (*PartnerType, error) {
	query := `
		SELECT id, code, name, commission_mode, commission_value, description, created_at, updated_at
		FROM partner_types
		WHERE id = ?`
	var pt PartnerType
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pt.ID, &pt.Code, &pt.Name, &pt.CommissionMode, &pt.CommissionValue, &pt.Description, &pt.CreatedAt, &pt.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &pt, nil
}

func (r *Repository) ListPartnerTypes(ctx context.Context, params PartnerTypeListParams) ([]PartnerType, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if strings.TrimSpace(params.Search) != "" {
		pattern := "%" + strings.TrimSpace(params.Search) + "%"
		where = append(where, "(name LIKE ? OR code LIKE ?)")
		args = append(args, pattern, pattern)
	}
	if params.CreatedFrom != nil {
		where = append(where, "created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	query := `
		SELECT id, code, name, commission_mode, commission_value, description, created_at, updated_at
		FROM partner_types
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PartnerType
	for rows.Next() {
		var pt PartnerType
		if err := rows.Scan(
			&pt.ID, &pt.Code, &pt.Name, &pt.CommissionMode, &pt.CommissionValue, &pt.Description, &pt.CreatedAt, &pt.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, pt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) UpdatePartnerType(ctx context.Context, id int64, pt PartnerType) error {
	query := `
		UPDATE partner_types
		SET name = ?, commission_mode = ?, commission_value = ?, description = ?, updated_at = NOW()
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, pt.Name, pt.CommissionMode, pt.CommissionValue, pt.Description, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

/* ---------- Partner ---------- */

func (r *Repository) CreatePartner(ctx context.Context, p Partner) (int64, error) {
	query := `
		INSERT INTO partners (
			partner_type_id, code, name, phone, email, address,
			bank_account_encrypted, bank_account_last4, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		p.PartnerTypeID,
		p.Code,
		p.Name,
		p.Phone,
		p.Email,
		p.Address,
		p.BankAccountEncrypted,
		p.BankAccountLast4,
		p.Status,
		p.CreatedAt,
		p.UpdatedAt)
	if err != nil {
		return 0, mapDuplicateError(err, "uq_partners_code")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetPartnerByID(ctx context.Context, id int64) (*Partner, error) {
	query := `
		SELECT id, partner_type_id, code, name, phone, email, address,
		       bank_account_encrypted, bank_account_last4, status, created_at, updated_at
		FROM partners
		WHERE id = ?`
	var p Partner
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.PartnerTypeID, &p.Code, &p.Name, &p.Phone, &p.Email, &p.Address,
		&p.BankAccountEncrypted, &p.BankAccountLast4, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) GetPartnerByCode(ctx context.Context, code string) (*Partner, error) {
	query := `
		SELECT id, partner_type_id, code, name, phone, email, address,
		       bank_account_encrypted, bank_account_last4, status, created_at, updated_at
		FROM partners
		WHERE code = ?`
	var p Partner
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&p.ID, &p.PartnerTypeID, &p.Code, &p.Name, &p.Phone, &p.Email, &p.Address,
		&p.BankAccountEncrypted, &p.BankAccountLast4, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListPartners(ctx context.Context, params PartnerListParams) ([]Partner, int64, error) {
	var args []any
	whereParts := []string{"1 = 1"}
	if strings.TrimSpace(params.Search) != "" {
		pattern := "%" + strings.TrimSpace(params.Search) + "%"
		whereParts = append(whereParts, "(name LIKE ? OR code LIKE ?)")
		args = append(args, pattern, pattern)
	}
	if params.CreatedFrom != nil {
		whereParts = append(whereParts, "created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		whereParts = append(whereParts, "created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	where := "WHERE " + strings.Join(whereParts, " AND ")
	// Count query
	countQuery := "SELECT COUNT(*) FROM partners " + where
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	// Data query
	query := "SELECT id, partner_type_id, code, name, phone, email, address, bank_account_encrypted, bank_account_last4, status, created_at, updated_at FROM partners " + where + " ORDER BY name"
	dataArgs := append([]any{}, args...)
	if params.Limit > 0 {
		query += " LIMIT ? OFFSET ?"
		dataArgs = append(dataArgs, params.Limit, params.Offset)
	}
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []Partner
	for rows.Next() {
		var p Partner
		if err := rows.Scan(
			&p.ID, &p.PartnerTypeID, &p.Code, &p.Name, &p.Phone, &p.Email, &p.Address,
			&p.BankAccountEncrypted, &p.BankAccountLast4, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) UpdatePartner(ctx context.Context, id int64, p Partner) error {
	query := `
		UPDATE partners
		SET name = ?, phone = ?, email = ?, address = ?,
		    bank_account_encrypted = ?, bank_account_last4 = ?,
		    status = ?, updated_at = NOW()
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query,
		p.Name, p.Phone, p.Email, p.Address,
		p.BankAccountEncrypted, p.BankAccountLast4, p.Status, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Soft delete by setting status to INACTIVE
func (r *Repository) DeactivatePartner(ctx context.Context, id int64) error {
	query := `
		UPDATE partners
		SET status = ?, updated_at = NOW()
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, StatusInactive, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

/* ---------- PartnerAssignment ---------- */

func (r *Repository) CreatePartnerAssignment(ctx context.Context, a PartnerAssignment) (int64, error) {
	query := `
		INSERT INTO partner_assignments (
			partner_id, user_id, assigned_by_id, assigned_at, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		a.PartnerID, a.UserID, a.AssignedByID, a.AssignedAt, a.Active, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AssignPIC atomically deactivates the current active assignment (if any) and
// inserts the new one, under a row lock on the partner. This serializes
// concurrent AssignPIC calls for the same partner_id so only one active
// assignment can ever exist — the uq_partner_assignments_one_active
// constraint (migration 20260724001100) is the DB-level backstop for the
// same guarantee.
func (r *Repository) AssignPIC(ctx context.Context, a PartnerAssignment) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT id FROM partners WHERE id = ? FOR UPDATE`, a.PartnerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_assignments
		SET unassigned_at = ?, active = FALSE, updated_at = ?
		WHERE partner_id = ? AND active = TRUE`, a.AssignedAt, a.UpdatedAt, a.PartnerID); err != nil {
		return 0, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO partner_assignments (
			partner_id, user_id, assigned_by_id, assigned_at, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, TRUE, ?, ?)`,
		a.PartnerID, a.UserID, a.AssignedByID, a.AssignedAt, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return 0, mapDuplicateError(err, "uq_partner_assignments_one_active")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetActiveAssignmentForPartner(ctx context.Context, partnerID int64) (*PartnerAssignment, error) {
	query := `
		SELECT id, partner_id, user_id, assigned_by_id, assigned_at,
		       unassigned_at, active, created_at, updated_at
		FROM partner_assignments
		WHERE partner_id = ? AND active = TRUE
		ORDER BY assigned_at DESC
		LIMIT 1`
	var a PartnerAssignment
	err := r.db.QueryRowContext(ctx, query, partnerID).Scan(
		&a.ID, &a.PartnerID, &a.UserID, &a.AssignedByID, &a.AssignedAt,
		&a.UnassignedAt, &a.Active, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repository) ListPartnerAssignments(ctx context.Context, partnerID int64, params PartnerHistoryListParams) ([]PartnerAssignment, error) {
	where := []string{"partner_id = ?"}
	args := []any{partnerID}
	if params.CreatedFrom != nil {
		where = append(where, "created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	query := `
		SELECT id, partner_id, user_id, assigned_by_id, assigned_at,
		       unassigned_at, active, created_at, updated_at
		FROM partner_assignments
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY assigned_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PartnerAssignment
	for rows.Next() {
		var a PartnerAssignment
		if err := rows.Scan(
			&a.ID, &a.PartnerID, &a.UserID, &a.AssignedByID, &a.AssignedAt,
			&a.UnassignedAt, &a.Active, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) DeactivatePartnerAssignment(ctx context.Context, assignmentID int64) error {
	query := `
		UPDATE partner_assignments
		SET unassigned_at = NOW(), active = FALSE, updated_at = NOW()
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, assignmentID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

/* ---------- PartnerInteraction ---------- */

func (r *Repository) CreatePartnerInteraction(ctx context.Context, i PartnerInteraction) (int64, error) {
	query := `
		INSERT INTO partner_interactions (
			partner_id, interaction_type, interaction_at, note, created_at)
		VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		i.PartnerID, i.InteractionType, i.InteractionAt, i.Note, i.CreatedAt)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) ListPartnerInteractions(ctx context.Context, partnerID int64, params PartnerHistoryListParams) ([]PartnerInteraction, int64, error) {
	var args []any
	whereParts := []string{}
	if partnerID > 0 {
		whereParts = append(whereParts, "partner_id = ?")
		args = append(args, partnerID)
	}
	if params.CreatedFrom != nil {
		whereParts = append(whereParts, "created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		whereParts = append(whereParts, "created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	where := ""
	if len(whereParts) > 0 {
		where = "WHERE " + strings.Join(whereParts, " AND ")
	}
	countQuery := "SELECT COUNT(*) FROM partner_interactions " + where
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	query := `
		SELECT id, partner_id, interaction_type, interaction_at, note, created_at
		FROM partner_interactions ` + where + `
		ORDER BY interaction_at DESC`
	queryArgs := append([]any{}, args...)
	if params.Limit > 0 {
		query += `
		LIMIT ? OFFSET ?`
		queryArgs = append(queryArgs, params.Limit, params.Offset)
	}
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []PartnerInteraction
	for rows.Next() {
		var i PartnerInteraction
		if err := rows.Scan(
			&i.ID, &i.PartnerID, &i.InteractionType, &i.InteractionAt, &i.Note, &i.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, i)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

/* ---------- PartnerReferral ---------- */

func (r *Repository) CreatePartnerReferral(ctx context.Context, ref PartnerReferral) (int64, error) {
	query := `
		INSERT INTO partner_referrals (
			partner_id, lead_id, referral_date, notes, created_at)
		VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		ref.PartnerID, ref.LeadID, ref.ReferralDate, ref.Notes, ref.CreatedAt)
	if err != nil {
		return 0, mapDuplicateError(err, "uq_partner_referrals_partner_lead")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) ListPartnerReferrals(ctx context.Context, partnerID int64, params PartnerHistoryListParams) ([]PartnerReferral, error) {
	where := []string{"partner_id = ?"}
	args := []any{partnerID}
	if params.CreatedFrom != nil {
		where = append(where, "created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	query := `
		SELECT id, partner_id, lead_id, referral_date, notes, created_at
		FROM partner_referrals
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY referral_date DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PartnerReferral
	for rows.Next() {
		var pr PartnerReferral
		if err := rows.Scan(
			&pr.ID, &pr.PartnerID, &pr.LeadID, &pr.ReferralDate, &pr.Notes, &pr.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// HasReferralInMonth reports whether partnerID has at least one partner_referrals row whose
// referral_date falls within the given calendar month — the basis for the monthly partner
// activity status (BELUM_MEMBERIKAN_REFERAL / TELAH_MEMBERIKAN_REFERAL).
func (r *Repository) HasReferralInMonth(ctx context.Context, partnerID int64, year int, month int) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM partner_referrals
		WHERE partner_id = ? AND YEAR(referral_date) = ? AND MONTH(referral_date) = ?`,
		partnerID, year, month).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetReferralByPartnerLead returns a referral for a given partner-lead pair, if exists.
func (r *Repository) GetReferralByPartnerLead(ctx context.Context, partnerID int64, leadID int64) (*PartnerReferral, error) {
	query := `
		SELECT id, partner_id, lead_id, referral_date, notes, created_at
		FROM partner_referrals
		WHERE partner_id = ? AND lead_id = ?`
	var pr PartnerReferral
	err := r.db.QueryRowContext(ctx, query, partnerID, leadID).Scan(
		&pr.ID, &pr.PartnerID, &pr.LeadID, &pr.ReferralDate, &pr.Notes, &pr.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &pr, nil
}

/* ---------- PartnerCommission ---------- */

type commissionSyncCandidate struct {
	ReferralID      int64
	ClosingID       int64
	FinalAmount     string
	Currency        string
	PlanID          sql.NullInt64
	ConfirmedAt     time.Time
	PartnerTypeID   int64
	CommissionMode  string // legacy partner_types fallback, used when no commission_rules row matches
	CommissionValue string
}

// findSyncableClosings finds CONFIRMED closings tied to this partner's referrals
// (matched by lead_id) that do not yet have a commission row. commission_mode/value here
// are the legacy partner_type fallback rate; SyncCommissions additionally consults
// commission_rules (package/effective-dated overlay, resolveCommissionRule) before falling
// back to these columns.
func (r *Repository) findSyncableClosings(ctx context.Context, tx *sql.Tx, partnerID int64) ([]commissionSyncCandidate, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT pr.id, sc.id, CAST(sc.final_amount AS CHAR), sc.currency, sc.plan_id, sc.confirmed_at,
		       p.partner_type_id, pt.commission_mode, CAST(pt.commission_value AS CHAR)
		FROM partner_referrals pr
		JOIN sales_closings sc ON sc.lead_id = pr.lead_id AND sc.status = 'CONFIRMED' AND sc.deleted_at IS NULL
		JOIN partners p ON p.id = pr.partner_id
		JOIN partner_types pt ON pt.id = p.partner_type_id
		LEFT JOIN partner_commissions pc ON pc.closing_id = sc.id
		WHERE pr.partner_id = ? AND pc.id IS NULL`, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []commissionSyncCandidate
	for rows.Next() {
		var c commissionSyncCandidate
		if err := rows.Scan(&c.ReferralID, &c.ClosingID, &c.FinalAmount, &c.Currency, &c.PlanID, &c.ConfirmedAt,
			&c.PartnerTypeID, &c.CommissionMode, &c.CommissionValue); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// resolvedCommissionRule is the outcome of resolveCommissionRule: a specific commission_rules
// row to apply. Mode may be TIER, requiring a further resolveTier lookup by the caller.
type resolvedCommissionRule struct {
	ID    int64
	Mode  string
	Value sql.NullString // NULL when Mode == TIER (rate lives in commission_tiers)
}

// resolveCommissionRule finds the most specific active commission_rules row covering
// partnerTypeID (+ optional planID) on asOf's date: a plan-specific rule beats a
// type-wide rule, ties broken by most recent effective_from. Returns (nil, nil) if no rule
// matches — the caller falls back to the legacy partner_types.commission_mode/value.
func (r *Repository) resolveCommissionRule(ctx context.Context, tx *sql.Tx, partnerTypeID int64, planID sql.NullInt64, asOf time.Time) (*resolvedCommissionRule, error) {
	asOfDate := asOf.Format("2006-01-02")
	row := tx.QueryRowContext(ctx, `
		SELECT id, mode, value
		FROM commission_rules
		WHERE partner_type_id = ?
		  AND active = TRUE
		  AND effective_from <= ?
		  AND (effective_to IS NULL OR effective_to >= ?)
		  AND (plan_id = ? OR plan_id IS NULL)
		ORDER BY (plan_id IS NOT NULL) DESC, effective_from DESC, id DESC
		LIMIT 1`, partnerTypeID, asOfDate, asOfDate, planID)
	var rule resolvedCommissionRule
	if err := row.Scan(&rule.ID, &rule.Mode, &rule.Value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// monthlyClosingOrdinal returns the 1-based rank of closingID among partnerID's CONFIRMED
// closings within confirmedAt's calendar month (ordered by confirmed_at then id) — the
// volume count a TIER commission_rule's tiers are bracketed by.
func (r *Repository) monthlyClosingOrdinal(ctx context.Context, tx *sql.Tx, partnerID int64, closingID int64, confirmedAt time.Time) (int, error) {
	var ordinal int
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sales_closings sc2
		JOIN partner_referrals pr2 ON pr2.lead_id = sc2.lead_id
		WHERE pr2.partner_id = ?
		  AND sc2.status = 'CONFIRMED' AND sc2.deleted_at IS NULL
		  AND YEAR(sc2.confirmed_at) = YEAR(?) AND MONTH(sc2.confirmed_at) = MONTH(?)
		  AND (sc2.confirmed_at < ? OR (sc2.confirmed_at = ? AND sc2.id <= ?))`,
		partnerID, confirmedAt, confirmedAt, confirmedAt, confirmedAt, closingID).Scan(&ordinal)
	if err != nil {
		return 0, err
	}
	return ordinal, nil
}

// resolveTier finds the commission_tiers row whose [min_closings, max_closings] range
// contains ordinal, for the given TIER-mode commission_rule.
func (r *Repository) resolveTier(ctx context.Context, tx *sql.Tx, ruleID int64, ordinal int) (mode string, value string, err error) {
	err = tx.QueryRowContext(ctx, `
		SELECT mode, CAST(value AS CHAR)
		FROM commission_tiers
		WHERE commission_rule_id = ?
		  AND min_closings <= ?
		  AND (max_closings IS NULL OR max_closings >= ?)
		ORDER BY tier_order
		LIMIT 1`, ruleID, ordinal, ordinal).Scan(&mode, &value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrNoMatchingTier
		}
		return "", "", err
	}
	return mode, value, nil
}

// SyncCommissions scans confirmed closings tied to partnerID's referrals that don't
// have a commission record yet and creates PENDING rows for them. Idempotent: closings
// already synced (unique on closing_id) are skipped on subsequent calls.
func (r *Repository) SyncCommissions(ctx context.Context, partnerID int64) ([]PartnerCommission, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	candidates, err := r.findSyncableClosings(ctx, tx, partnerID)
	if err != nil {
		return nil, err
	}

	createdIDs := make([]int64, 0, len(candidates))
	for _, cand := range candidates {
		baseCents, err := parseMoneyToCents(cand.FinalAmount)
		if err != nil {
			return nil, err
		}

		// Resolve the effective rate: a matching commission_rules row (package-specific >
		// type-wide, TIER resolved by monthly closing volume) overrides the legacy
		// partner_types fallback already carried on cand.CommissionMode/Value.
		effectiveMode, effectiveValue := cand.CommissionMode, cand.CommissionValue
		var ruleID sql.NullInt64
		var tierOrdinal sql.NullInt64

		rule, err := r.resolveCommissionRule(ctx, tx, cand.PartnerTypeID, cand.PlanID, cand.ConfirmedAt)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			ruleID = sql.NullInt64{Int64: rule.ID, Valid: true}
			if rule.Mode == CommissionModeTier {
				ordinal, err := r.monthlyClosingOrdinal(ctx, tx, partnerID, cand.ClosingID, cand.ConfirmedAt)
				if err != nil {
					return nil, err
				}
				tierMode, tierValue, err := r.resolveTier(ctx, tx, rule.ID, ordinal)
				if err != nil {
					return nil, err
				}
				effectiveMode, effectiveValue = tierMode, tierValue
				tierOrdinal = sql.NullInt64{Int64: int64(ordinal), Valid: true}
			} else {
				effectiveMode, effectiveValue = rule.Mode, rule.Value.String
			}
		}

		commissionCents, err := calculateCommissionAmountCents(effectiveMode, effectiveValue, baseCents)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		code := fmt.Sprintf("COM-%s-%06d-%06d", now.Format("20060102"), cand.ClosingID, now.Nanosecond()/1000)
		result, err := tx.ExecContext(ctx, `
			INSERT INTO partner_commissions (
				code, partner_id, referral_id, closing_id, commission_mode, commission_value,
				commission_rule_id, tier_ordinal,
				base_amount, commission_amount, currency, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			code, partnerID, cand.ReferralID, cand.ClosingID, effectiveMode, effectiveValue,
			ruleID, tierOrdinal,
			formatCents(baseCents), formatCents(commissionCents), cand.Currency, CommissionStatusPending)
		if err != nil {
			return nil, mapDuplicateError(err, "uq_partner_commissions_closing")
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		createdIDs = append(createdIDs, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	created := make([]PartnerCommission, 0, len(createdIDs))
	for _, id := range createdIDs {
		pc, err := r.GetPartnerCommissionByID(ctx, id)
		if err != nil {
			return nil, err
		}
		created = append(created, *pc)
	}
	return created, nil
}

const commissionSelectColumns = `
		pc.id, pc.code, pc.partner_id, p.code, p.name, pc.referral_id, pc.closing_id, sc.code,
		pc.commission_mode, pc.commission_value, pc.commission_rule_id, pc.tier_ordinal,
		pc.base_amount, pc.commission_amount, pc.currency, pc.status, pc.note,
		pc.approved_by_user_id, au.name, pc.approved_at,
		pc.paid_by_user_id, pu.name, pc.paid_at,
		ppi.payout_id,
		pc.created_at, pc.updated_at`

const commissionFromJoin = `
		FROM partner_commissions pc
		JOIN partners p ON p.id = pc.partner_id
		JOIN sales_closings sc ON sc.id = pc.closing_id
		LEFT JOIN users au ON au.id = pc.approved_by_user_id
		LEFT JOIN users pu ON pu.id = pc.paid_by_user_id
		LEFT JOIN partner_payout_items ppi ON ppi.commission_id = pc.id AND ppi.released_at IS NULL`

func scanCommission(scanner interface {
	Scan(dest ...any) error
}) (PartnerCommission, error) {
	var c PartnerCommission
	err := scanner.Scan(
		&c.ID, &c.Code, &c.PartnerID, &c.PartnerCode, &c.PartnerName, &c.ReferralID, &c.ClosingID, &c.ClosingCode,
		&c.CommissionMode, &c.CommissionValue, &c.CommissionRuleID, &c.TierOrdinal,
		&c.BaseAmount, &c.CommissionAmount, &c.Currency, &c.Status, &c.Note,
		&c.ApprovedByUserID, &c.ApprovedByName, &c.ApprovedAt,
		&c.PaidByUserID, &c.PaidByName, &c.PaidAt,
		&c.ActivePayoutID,
		&c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) GetPartnerCommissionByID(ctx context.Context, id int64) (*PartnerCommission, error) {
	query := "SELECT " + commissionSelectColumns + commissionFromJoin + " WHERE pc.id = ?"
	c, err := scanCommission(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListPartnerCommissions(ctx context.Context, partnerID int64, status string, limit int, offset int) ([]PartnerCommission, int64, error) {
	args := []any{partnerID}
	where := "WHERE pc.partner_id = ?"
	if status != "" {
		where += " AND pc.status = ?"
		args = append(args, status)
	}
	var total int64
	countQuery := "SELECT COUNT(*) FROM partner_commissions pc " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT " + commissionSelectColumns + commissionFromJoin + " " + where + " ORDER BY pc.created_at DESC"
	dataArgs := append([]any{}, args...)
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		dataArgs = append(dataArgs, limit, offset)
	}
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []PartnerCommission
	for rows.Next() {
		c, err := scanCommission(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) lockCommissionStatus(ctx context.Context, tx *sql.Tx, id int64) (string, error) {
	var status string
	err := tx.QueryRowContext(ctx, `SELECT status FROM partner_commissions WHERE id = ? FOR UPDATE`, id).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *Repository) ApproveCommission(ctx context.Context, id int64, approvedByID int64) (*PartnerCommission, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	status, err := r.lockCommissionStatus(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if status != CommissionStatusPending {
		return nil, ErrInvalidCommissionStatus
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_commissions
		SET status = ?, approved_by_user_id = ?, approved_at = NOW(), updated_at = NOW()
		WHERE id = ?`, CommissionStatusApproved, approvedByID, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetPartnerCommissionByID(ctx, id)
}

// ensureCommissionNotInPayout locks (FOR UPDATE) and checks that commissionID has no active
// (non-released) payout_item row — guards against double-payment via the individual pay/
// cancel endpoints while the commission sits reserved inside a batch payout. Must be called
// inside the same transaction as the caller's status lock.
func (r *Repository) ensureCommissionNotInPayout(ctx context.Context, tx *sql.Tx, commissionID int64) error {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1 FROM partner_payout_items WHERE commission_id = ? AND released_at IS NULL FOR UPDATE`, commissionID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return ErrCommissionInPayout
}

func (r *Repository) MarkCommissionPaid(ctx context.Context, id int64, paidByID int64) (*PartnerCommission, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	status, err := r.lockCommissionStatus(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if status != CommissionStatusApproved {
		return nil, ErrInvalidCommissionStatus
	}
	if err := r.ensureCommissionNotInPayout(ctx, tx, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_commissions
		SET status = ?, paid_by_user_id = ?, paid_at = NOW(), updated_at = NOW()
		WHERE id = ?`, CommissionStatusPaid, paidByID, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetPartnerCommissionByID(ctx, id)
}

func (r *Repository) CancelCommission(ctx context.Context, id int64, note string) (*PartnerCommission, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	status, err := r.lockCommissionStatus(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if status == CommissionStatusPaid || status == CommissionStatusCancelled {
		return nil, ErrInvalidCommissionStatus
	}
	if err := r.ensureCommissionNotInPayout(ctx, tx, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_commissions
		SET status = ?, note = ?, updated_at = NOW()
		WHERE id = ?`, CommissionStatusCancelled, nullableString(note), id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetPartnerCommissionByID(ctx, id)
}

/* ---------- CommissionRule ---------- */

const commissionRuleSelectColumns = `
		cr.id, cr.partner_type_id, cr.plan_id, spl.code, spl.name,
		cr.mode, cr.value, cr.effective_from, cr.effective_to, cr.active,
		cr.created_by_user_id, cu.name, cr.created_at, cr.updated_at`

const commissionRuleFromJoin = `
		FROM commission_rules cr
		LEFT JOIN subscription_plans spl ON spl.id = cr.plan_id
		LEFT JOIN users cu ON cu.id = cr.created_by_user_id`

func scanCommissionRule(scanner interface {
	Scan(dest ...any) error
}) (CommissionRule, error) {
	var r CommissionRule
	err := scanner.Scan(
		&r.ID, &r.PartnerTypeID, &r.PlanID, &r.PlanCode, &r.PlanName,
		&r.Mode, &r.Value, &r.EffectiveFrom, &r.EffectiveTo, &r.Active,
		&r.CreatedByUserID, &r.CreatedByName, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

// CreateCommissionRule inserts a commission_rules row and, when tiers is non-empty (TIER
// mode), its child commission_tiers rows, in a single transaction.
func (r *Repository) CreateCommissionRule(ctx context.Context, rule CommissionRule, tiers []CommissionTier) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if rule.PlanID.Valid {
		if _, err := tx.ExecContext(ctx, `
			UPDATE commission_rules
			SET active = FALSE, effective_to = COALESCE(effective_to, CURDATE()), updated_at = NOW()
			WHERE partner_type_id = ? AND plan_id = ? AND active = TRUE`,
			rule.PartnerTypeID, rule.PlanID.Int64); err != nil {
			return 0, fmt.Errorf("database partner: %w", err)
		}
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO commission_rules (
			partner_type_id, plan_id, mode, value, effective_from, effective_to, active, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, NOW(), NOW())`,
		rule.PartnerTypeID, rule.PlanID, rule.Mode, rule.Value, rule.EffectiveFrom, rule.EffectiveTo, rule.CreatedByUserID)
	if err != nil {
		return 0, fmt.Errorf("database partner: %w", err)
	}
	ruleID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, t := range tiers {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO commission_tiers (commission_rule_id, tier_order, min_closings, max_closings, mode, value, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			ruleID, t.TierOrder, t.MinClosings, t.MaxClosings, t.Mode, t.Value); err != nil {
			return 0, fmt.Errorf("database partner: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return ruleID, nil
}

func (r *Repository) listCommissionTiers(ctx context.Context, ruleID int64) ([]CommissionTier, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, commission_rule_id, tier_order, min_closings, max_closings, mode, CAST(value AS CHAR), created_at, updated_at
		FROM commission_tiers WHERE commission_rule_id = ? ORDER BY tier_order`, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []CommissionTier
	for rows.Next() {
		var t CommissionTier
		if err := rows.Scan(&t.ID, &t.CommissionRuleID, &t.TierOrder, &t.MinClosings, &t.MaxClosings, &t.Mode, &t.Value, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) GetCommissionRuleByID(ctx context.Context, id int64) (*CommissionRule, error) {
	query := "SELECT " + commissionRuleSelectColumns + commissionRuleFromJoin + " WHERE cr.id = ?"
	rule, err := scanCommissionRule(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	tiers, err := r.listCommissionTiers(ctx, id)
	if err != nil {
		return nil, err
	}
	rule.Tiers = tiers
	return &rule, nil
}

func (r *Repository) ListCommissionRules(ctx context.Context, partnerTypeID int64, planID *int64, activeOnly bool) ([]CommissionRule, error) {
	args := []any{partnerTypeID}
	where := "WHERE cr.partner_type_id = ?"
	if planID != nil {
		where += " AND cr.plan_id = ?"
		args = append(args, *planID)
	}
	if activeOnly {
		where += " AND cr.active = TRUE"
	}
	query := "SELECT " + commissionRuleSelectColumns + commissionRuleFromJoin + " " + where + " ORDER BY cr.effective_from DESC, cr.id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []CommissionRule
	for rows.Next() {
		rule, err := scanCommissionRule(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// DeactivateCommissionRule sets active=FALSE and, if effective_to was still open-ended,
// closes it to today — rules are superseded (create a new one), never edited in place.
func (r *Repository) DeactivateCommissionRule(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE commission_rules
		SET active = FALSE, effective_to = COALESCE(effective_to, CURDATE()), updated_at = NOW()
		WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

/* ---------- PartnerPayout ---------- */

const payoutSelectColumns = `
		pp.id, pp.code, pp.partner_id, p.code, p.name,
		pp.total_amount, pp.currency, pp.status, pp.note,
		pp.prepared_by_user_id, pru.name, pp.paid_by_user_id, pau.name, pp.paid_at,
		pp.created_at, pp.updated_at`

const payoutFromJoin = `
		FROM partner_payouts pp
		JOIN partners p ON p.id = pp.partner_id
		LEFT JOIN users pru ON pru.id = pp.prepared_by_user_id
		LEFT JOIN users pau ON pau.id = pp.paid_by_user_id`

func scanPayout(scanner interface {
	Scan(dest ...any) error
}) (PartnerPayout, error) {
	var p PartnerPayout
	err := scanner.Scan(
		&p.ID, &p.Code, &p.PartnerID, &p.PartnerCode, &p.PartnerName,
		&p.TotalAmount, &p.Currency, &p.Status, &p.Note,
		&p.PreparedByUserID, &p.PreparedByName, &p.PaidByUserID, &p.PaidByName, &p.PaidAt,
		&p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// CreatePayout locks every APPROVED, not-yet-batched commission for partnerID (FOR UPDATE,
// serializing against MarkCommissionPaid/CancelCommission/a concurrent CreatePayout on the
// same partner via ensureCommissionNotInPayout's same NOT EXISTS shape) and batches them
// into one new PENDING payout. Returns ErrNoPayableCommissions if none are eligible,
// ErrMixedCurrency if the eligible commissions don't all share one currency.
func (r *Repository) CreatePayout(ctx context.Context, partnerID int64, preparedByID int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT pc.id, pc.commission_amount, pc.currency
		FROM partner_commissions pc
		WHERE pc.partner_id = ? AND pc.status = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM partner_payout_items ppi
		    WHERE ppi.commission_id = pc.id AND ppi.released_at IS NULL
		  )
		FOR UPDATE`, partnerID, CommissionStatusApproved)
	if err != nil {
		return 0, err
	}
	type eligibleCommission struct {
		ID       int64
		Amount   string
		Currency string
	}
	var items []eligibleCommission
	for rows.Next() {
		var e eligibleCommission
		if err := rows.Scan(&e.ID, &e.Amount, &e.Currency); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	if len(items) == 0 {
		return 0, ErrNoPayableCommissions
	}

	currency := items[0].Currency
	totalCents := int64(0)
	for _, it := range items {
		if it.Currency != currency {
			return 0, ErrMixedCurrency
		}
		cents, err := parseMoneyToCents(it.Amount)
		if err != nil {
			return 0, err
		}
		totalCents += cents
	}

	now := time.Now().UTC()
	code := fmt.Sprintf("PAYOUT-%s-%06d-%06d", now.Format("20060102"), partnerID, now.Nanosecond()/1000)
	result, err := tx.ExecContext(ctx, `
		INSERT INTO partner_payouts (code, partner_id, total_amount, currency, status, prepared_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		code, partnerID, formatCents(totalCents), currency, PayoutStatusPending, preparedByID)
	if err != nil {
		return 0, fmt.Errorf("database partner: %w", err)
	}
	payoutID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, it := range items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO partner_payout_items (payout_id, commission_id, amount, created_at, updated_at)
			VALUES (?, ?, ?, NOW(), NOW())`, payoutID, it.ID, it.Amount); err != nil {
			return 0, mapDuplicateError(err, "uq_partner_payout_items_active_commission")
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return payoutID, nil
}

func (r *Repository) lockPayoutStatus(ctx context.Context, tx *sql.Tx, id int64) (string, error) {
	var status string
	err := tx.QueryRowContext(ctx, `SELECT status FROM partner_payouts WHERE id = ? FOR UPDATE`, id).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *Repository) listPayoutItems(ctx context.Context, payoutID int64) ([]PartnerPayoutItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ppi.id, ppi.payout_id, ppi.commission_id, pc.code, ppi.amount, ppi.released_at, ppi.created_at
		FROM partner_payout_items ppi
		JOIN partner_commissions pc ON pc.id = ppi.commission_id
		WHERE ppi.payout_id = ?
		ORDER BY ppi.id`, payoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PartnerPayoutItem
	for rows.Next() {
		var it PartnerPayoutItem
		if err := rows.Scan(&it.ID, &it.PayoutID, &it.CommissionID, &it.CommissionCode, &it.Amount, &it.ReleasedAt, &it.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) GetPayoutByID(ctx context.Context, id int64) (*PartnerPayout, error) {
	query := "SELECT " + payoutSelectColumns + payoutFromJoin + " WHERE pp.id = ?"
	p, err := scanPayout(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	items, err := r.listPayoutItems(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Items = items
	return &p, nil
}

func (r *Repository) ListPartnerPayouts(ctx context.Context, partnerID int64, status string, limit int, offset int) ([]PartnerPayout, int64, error) {
	args := []any{partnerID}
	where := "WHERE pp.partner_id = ?"
	if status != "" {
		where += " AND pp.status = ?"
		args = append(args, status)
	}
	var total int64
	countQuery := "SELECT COUNT(*) FROM partner_payouts pp " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT " + payoutSelectColumns + payoutFromJoin + " " + where + " ORDER BY pp.created_at DESC"
	dataArgs := append([]any{}, args...)
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		dataArgs = append(dataArgs, limit, offset)
	}
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []PartnerPayout
	for rows.Next() {
		p, err := scanPayout(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// MarkPayoutPaid transitions a PENDING payout to PAID and cascades PAID status (with
// paid_by_user_id/paid_at) to every commission still actively linked to it. This is the
// only place a batched commission's status actually changes — it stays APPROVED for the
// entire time it merely sits reserved in a PENDING payout.
func (r *Repository) MarkPayoutPaid(ctx context.Context, id int64, paidByID int64) (*PartnerPayout, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	status, err := r.lockPayoutStatus(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if status != PayoutStatusPending {
		return nil, ErrInvalidPayoutStatus
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_commissions
		SET status = ?, paid_by_user_id = ?, paid_at = NOW(), updated_at = NOW()
		WHERE id IN (
			SELECT commission_id FROM partner_payout_items WHERE payout_id = ? AND released_at IS NULL
		)`, CommissionStatusPaid, paidByID, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_payouts
		SET status = ?, paid_by_user_id = ?, paid_at = NOW(), updated_at = NOW()
		WHERE id = ?`, PayoutStatusPaid, paidByID, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetPayoutByID(ctx, id)
}

// CancelPayout transitions a PENDING payout to CANCELLED and soft-releases every commission
// still actively linked to it (released_at = NOW()). Those commissions were never mutated
// by CreatePayout in the first place, so "released back to APPROVED" is true by
// construction — nothing needs reverting on partner_commissions itself.
func (r *Repository) CancelPayout(ctx context.Context, id int64, note string) (*PartnerPayout, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	status, err := r.lockPayoutStatus(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if status != PayoutStatusPending {
		return nil, ErrInvalidPayoutStatus
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_payout_items SET released_at = NOW(), updated_at = NOW()
		WHERE payout_id = ? AND released_at IS NULL`, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE partner_payouts SET status = ?, note = ?, updated_at = NOW()
		WHERE id = ?`, PayoutStatusCancelled, nullableString(note), id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetPayoutByID(ctx, id)
}

/* ---------- Helper functions ---------- */

func mapDuplicateError(err error, uniqueKey string) error {
	message := err.Error()
	if strings.Contains(message, "Duplicate entry") || strings.Contains(message, uniqueKey) {
		switch uniqueKey {
		case "uq_partner_types_code":
			return ErrDuplicateType
		case "uq_partners_code":
			return ErrDuplicatePartner
		case "uq_partner_referrals_partner_lead":
			return ErrDuplicateReferral
		case "uq_partner_commissions_closing":
			return ErrCommissionAlreadyExists
		case "uq_partner_assignments_one_active":
			return ErrInvalidAssignment
		case "uq_partner_payout_items_active_commission":
			return ErrCommissionInPayout
		}
	}
	return fmt.Errorf("database partner: %w", err)
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
