package partner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
		INSERT INTO partner_types (code, name, description, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())`
	result, err := r.db.ExecContext(ctx, query, pt.Code, pt.Name, pt.Description)
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
		SELECT id, code, name, description, created_at, updated_at
		FROM partner_types
		WHERE id = ?`
	var pt PartnerType
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pt.ID, &pt.Code, &pt.Name, &pt.Description, &pt.CreatedAt, &pt.UpdatedAt)
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
		SELECT id, code, name, description, created_at, updated_at
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
			&pt.ID, &pt.Code, &pt.Name, &pt.Description, &pt.CreatedAt, &pt.UpdatedAt); err != nil {
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
		SET name = ?, description = ?, updated_at = NOW()
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, pt.Name, pt.Description, id)
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