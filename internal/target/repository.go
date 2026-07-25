package target

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Repository provides sales_targets persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new target Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const targetSelectColumns = `
	st.id, st.sales_id, u.name, u.code, st.metric_code_id, mc.code, mc.name, mc.unit,
	st.period_year, st.period_month, st.target_value, st.source,
	st.created_by_user_id, cb.name,
	st.created_at, st.updated_at`

const targetFromJoin = `
	FROM sales_targets st
	JOIN users u ON u.id = st.sales_id
	JOIN metric_codes mc ON mc.id = st.metric_code_id
	LEFT JOIN users cb ON cb.id = st.created_by_user_id`

func scanTarget(scanner interface {
	Scan(dest ...any) error
}) (SalesTarget, error) {
	var t SalesTarget
	err := scanner.Scan(
		&t.ID, &t.SalesID, &t.SalesName, &t.SalesCode, &t.MetricCodeID, &t.MetricCode, &t.MetricName, &t.MetricUnit,
		&t.PeriodYear, &t.PeriodMonth, &t.TargetValue, &t.Source,
		&t.CreatedByUserID, &t.CreatedByName,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

// metricCodeID looks up an active metric_codes row by its code.
func (r *Repository) metricCodeID(ctx context.Context, code string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM metric_codes WHERE code = ? AND active = TRUE`, code).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrInvalidMetric
	}
	if err != nil {
		return 0, fmt.Errorf("target: lookup metric_code %s: %w", code, err)
	}
	return id, nil
}

// activeSalesIDs returns the IDs of every active Sales-role user.
func (r *Repository) activeSalesIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.code = ? AND u.status = 'ACTIVE' AND u.deleted_at IS NULL`,
		RoleSales,
	)
	if err != nil {
		return nil, fmt.Errorf("target: list active sales: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) ensureActiveSales(ctx context.Context, salesID int64) error {
	var exists int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1 FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = ? AND r.code = ? AND u.status = 'ACTIVE' AND u.deleted_at IS NULL
		LIMIT 1`, salesID, RoleSales).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrSalesNotEligible
	}
	if err != nil {
		return fmt.Errorf("target: check sales eligibility: %w", err)
	}
	return nil
}

// BulkSet inserts a target row for every active Sales rep who does not already have one for
// this (metric, period). Existing rows — bulk or override — are never touched.
func (r *Repository) BulkSet(ctx context.Context, req BulkSetTargetRequest, createdByUserID int64) (BulkSetTargetResponse, error) {
	resp := BulkSetTargetResponse{
		MetricCode:  req.MetricCode,
		PeriodYear:  req.PeriodYear,
		PeriodMonth: req.PeriodMonth,
		TargetValue: req.TargetValue,
	}

	metricID, err := r.metricCodeID(ctx, req.MetricCode)
	if err != nil {
		return resp, err
	}

	salesIDs, err := r.activeSalesIDs(ctx)
	if err != nil {
		return resp, err
	}
	resp.EligibleSales = len(salesIDs)
	if len(salesIDs) == 0 {
		return resp, nil
	}

	placeholders := make([]string, len(salesIDs))
	args := make([]any, 0, len(salesIDs)*6)
	for i, salesID := range salesIDs {
		placeholders[i] = "(?, ?, ?, ?, ?, ?)"
		args = append(args, salesID, metricID, req.PeriodYear, req.PeriodMonth, req.TargetValue, createdByUserID)
	}

	query := `
		INSERT IGNORE INTO sales_targets
			(sales_id, metric_code_id, period_year, period_month, target_value, created_by_user_id)
		VALUES ` + strings.Join(placeholders, ", ")

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return resp, fmt.Errorf("target: bulk set: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return resp, fmt.Errorf("target: bulk set rows affected: %w", err)
	}
	resp.Created = int(created)
	return resp, nil
}

// Override upserts a single sales rep's target, always winning over a prior bulk value.
func (r *Repository) Override(ctx context.Context, salesID int64, req OverrideTargetRequest, createdByUserID int64) (*SalesTarget, error) {
	if err := r.ensureActiveSales(ctx, salesID); err != nil {
		return nil, err
	}
	metricID, err := r.metricCodeID(ctx, req.MetricCode)
	if err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO sales_targets
			(sales_id, metric_code_id, period_year, period_month, target_value, source, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, 'OVERRIDE', ?)
		ON DUPLICATE KEY UPDATE
			target_value = VALUES(target_value),
			source = 'OVERRIDE',
			created_by_user_id = VALUES(created_by_user_id)`,
		salesID, metricID, req.PeriodYear, req.PeriodMonth, req.TargetValue, createdByUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("target: override: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT `+targetSelectColumns+targetFromJoin+`
		WHERE st.sales_id = ? AND st.metric_code_id = ? AND st.period_year = ? AND st.period_month = ?`,
		salesID, metricID, req.PeriodYear, req.PeriodMonth,
	)
	target, err := scanTarget(row)
	if err != nil {
		return nil, fmt.Errorf("target: reload after override: %w", err)
	}
	return &target, nil
}

// List returns targets matching the given filters, scoped by visibilityWhere.
func (r *Repository) List(ctx context.Context, actorID int64, actorRole string, params ListTargetsParams) ([]SalesTarget, int64, error) {
	where := []string{}
	args := []any{}

	visibility, visibilityArgs := visibilityWhere(actorID, actorRole)
	where = append(where, visibility)
	args = append(args, visibilityArgs...)

	if params.SalesID != nil {
		where = append(where, "st.sales_id = ?")
		args = append(args, *params.SalesID)
	}
	if params.PeriodYear != nil {
		where = append(where, "st.period_year = ?")
		args = append(args, *params.PeriodYear)
	}
	if params.PeriodMonth != nil {
		where = append(where, "st.period_month = ?")
		args = append(args, *params.PeriodMonth)
	}
	if params.MetricCode != "" {
		where = append(where, "mc.code = ?")
		args = append(args, params.MetricCode)
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	countQuery := `SELECT COUNT(*) ` + targetFromJoin + ` ` + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("target: count: %w", err)
	}

	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := `SELECT ` + targetSelectColumns + targetFromJoin + ` ` + whereClause + `
		ORDER BY st.period_year DESC, st.period_month DESC, st.sales_id ASC
		LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("target: list: %w", err)
	}
	defer rows.Close()

	var items []SalesTarget
	for rows.Next() {
		target, err := scanTarget(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, target)
	}
	return items, total, rows.Err()
}

// visibilityWhere mirrors the pattern used by internal/lead and internal/closing:
// Admin sees everything, Supervisor is unrestricted for now (targets aren't hierarchically
// scoped per-team in this sprint), Sales only sees their own rows.
func visibilityWhere(actorID int64, actorRole string) (string, []any) {
	switch actorRole {
	case RoleAdmin, RoleSupervisor:
		return "1 = 1", nil
	case RoleSales:
		return "st.sales_id = ?", []any{actorID}
	default:
		return "1 = 0", nil
	}
}
