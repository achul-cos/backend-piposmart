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
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := r.db.ExecContext(ctx, query, pt.Code, pt.Name, pt.CommissionMode, pt.CommissionValue, pt.Description)
	if err != nil {
		return 0, mapDuplicateError(err, "uq_partner_types_code")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
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

func (r *Repository) ListPartnerTypes(ctx context.Context) ([]PartnerType, error) {
	query := `
		SELECT id, code, name, commission_mode, commission_value, description, created_at, updated_at
		FROM partner_types
		ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := r.db.ExecContext(ctx, query,
		p.PartnerTypeID,
		p.Code,
		p.Name,
		p.Phone,
		p.Email,
		p.Address,
		p.BankAccountEncrypted,
		p.BankAccountLast4,
		p.Status)
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

func (r *Repository) ListPartners(ctx context.Context, limit int, offset int, search string) ([]Partner, int64, error) {
	var args []any
	var where string
	if search != "" {
		where = "WHERE (name LIKE ? OR code LIKE ?)"
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
	}
	// Count query
	countQuery := "SELECT COUNT(*) FROM partners " + where
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	// Data query
	query := "SELECT id, partner_type_id, code, name, phone, email, address, bank_account_encrypted, bank_account_last4, status, created_at, updated_at FROM partners " + where + " ORDER BY name LIMIT ? OFFSET ?"
	dataArgs := append(args, limit, offset)
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
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := r.db.ExecContext(ctx, query,
		a.PartnerID, a.UserID, a.AssignedByID, a.AssignedAt, a.Active)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
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

func (r *Repository) ListPartnerAssignments(ctx context.Context, partnerID int64) ([]PartnerAssignment, error) {
	query := `
		SELECT id, partner_id, user_id, assigned_by_id, assigned_at,
		       unassigned_at, active, created_at, updated_at
		FROM partner_assignments
		WHERE partner_id = ?
		ORDER BY assigned_at DESC`
	rows, err := r.db.QueryContext(ctx, query, partnerID)
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
		VALUES (?, ?, ?, ?, NOW())`
	result, err := r.db.ExecContext(ctx, query,
		i.PartnerID, i.InteractionType, i.InteractionAt, i.Note)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) ListPartnerInteractions(ctx context.Context, partnerID int64, limit int, offset int) ([]PartnerInteraction, int64, error) {
	var args []any
	var where string
	if partnerID > 0 {
		where = "WHERE partner_id = ?"
		args = append(args, partnerID)
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
		ORDER BY interaction_at DESC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
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
		VALUES (?, ?, ?, ?, NOW())`
	result, err := r.db.ExecContext(ctx, query,
		ref.PartnerID, ref.LeadID, ref.ReferralDate, ref.Notes)
	if err != nil {
		return 0, mapDuplicateError(err, "uq_partner_referrals_partner_lead")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) ListPartnerReferrals(ctx context.Context, partnerID int64) ([]PartnerReferral, error) {
	query := `
		SELECT id, partner_id, lead_id, referral_date, notes, created_at
		FROM partner_referrals
		WHERE partner_id = ?
		ORDER BY referral_date DESC`
	rows, err := r.db.QueryContext(ctx, query, partnerID)
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
	CommissionMode  string
	CommissionValue string
}

// findSyncableClosings finds CONFIRMED closings tied to this partner's referrals
// (matched by lead_id) that do not yet have a commission row. The commission rate is
// read from the partner's partner_type (commission_mode/commission_value), not from the
// partner itself.
func (r *Repository) findSyncableClosings(ctx context.Context, tx *sql.Tx, partnerID int64) ([]commissionSyncCandidate, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT pr.id, sc.id, CAST(sc.final_amount AS CHAR), sc.currency, pt.commission_mode, CAST(pt.commission_value AS CHAR)
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
		if err := rows.Scan(&c.ReferralID, &c.ClosingID, &c.FinalAmount, &c.Currency, &c.CommissionMode, &c.CommissionValue); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
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
		commissionCents, err := calculateCommissionAmountCents(cand.CommissionMode, cand.CommissionValue, baseCents)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		code := fmt.Sprintf("COM-%s-%06d-%06d", now.Format("20060102"), cand.ClosingID, now.Nanosecond()/1000)
		result, err := tx.ExecContext(ctx, `
			INSERT INTO partner_commissions (
				code, partner_id, referral_id, closing_id, commission_mode, commission_value,
				base_amount, commission_amount, currency, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			code, partnerID, cand.ReferralID, cand.ClosingID, cand.CommissionMode, cand.CommissionValue,
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
		pc.commission_mode, pc.commission_value, pc.base_amount, pc.commission_amount, pc.currency, pc.status, pc.note,
		pc.approved_by_user_id, au.name, pc.approved_at,
		pc.paid_by_user_id, pu.name, pc.paid_at,
		pc.created_at, pc.updated_at`

const commissionFromJoin = `
		FROM partner_commissions pc
		JOIN partners p ON p.id = pc.partner_id
		JOIN sales_closings sc ON sc.id = pc.closing_id
		LEFT JOIN users au ON au.id = pc.approved_by_user_id
		LEFT JOIN users pu ON pu.id = pc.paid_by_user_id`

func scanCommission(scanner interface {
	Scan(dest ...any) error
}) (PartnerCommission, error) {
	var c PartnerCommission
	err := scanner.Scan(
		&c.ID, &c.Code, &c.PartnerID, &c.PartnerCode, &c.PartnerName, &c.ReferralID, &c.ClosingID, &c.ClosingCode,
		&c.CommissionMode, &c.CommissionValue, &c.BaseAmount, &c.CommissionAmount, &c.Currency, &c.Status, &c.Note,
		&c.ApprovedByUserID, &c.ApprovedByName, &c.ApprovedAt,
		&c.PaidByUserID, &c.PaidByName, &c.PaidAt,
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

	query := "SELECT " + commissionSelectColumns + commissionFromJoin + " " + where + " ORDER BY pc.created_at DESC LIMIT ? OFFSET ?"
	dataArgs := append(args, limit, offset)
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
