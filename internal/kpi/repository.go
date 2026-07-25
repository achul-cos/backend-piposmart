package kpi

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Repository provides KPI definition, result, and ranking persistence, plus the recompute
// algorithm itself (Recompute runs inside a caller-supplied transaction so it can be invoked
// identically by the job worker and by the demo seeder).
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new kpi Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

/* ---------- KPI Definition CRUD ---------- */

const definitionSelectColumns = `
	kd.id, kd.metric_code_id, mc.code, mc.name, mc.unit,
	kd.period_year, kd.period_month, kd.weight, kd.threshold_achieved, kd.threshold_near, kd.active,
	kd.created_by_user_id, cb.name,
	kd.created_at, kd.updated_at`

const definitionFromJoin = `
	FROM kpi_definitions kd
	JOIN metric_codes mc ON mc.id = kd.metric_code_id
	LEFT JOIN users cb ON cb.id = kd.created_by_user_id`

func scanDefinition(scanner interface {
	Scan(dest ...any) error
}) (KpiDefinition, error) {
	var d KpiDefinition
	err := scanner.Scan(
		&d.ID, &d.MetricCodeID, &d.MetricCode, &d.MetricName, &d.MetricUnit,
		&d.PeriodYear, &d.PeriodMonth, &d.Weight, &d.ThresholdAchieved, &d.ThresholdNear, &d.Active,
		&d.CreatedByUserID, &d.CreatedByName,
		&d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (r *Repository) metricCodeID(ctx context.Context, code string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM metric_codes WHERE code = ? AND active = TRUE`, code).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrInvalidMetric
	}
	if err != nil {
		return 0, fmt.Errorf("kpi: lookup metric_code %s: %w", code, err)
	}
	return id, nil
}

// CreateDefinition inserts a new KPI definition. Effective-dating for KPI definitions means
// supersede via deactivate + new row, mirroring internal/partner's commission_rules — there is
// deliberately no update endpoint.
func (r *Repository) CreateDefinition(ctx context.Context, req CreateKpiDefinitionRequest, createdByUserID int64) (int64, error) {
	metricID, err := r.metricCodeID(ctx, req.MetricCode)
	if err != nil {
		return 0, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO kpi_definitions
			(metric_code_id, period_year, period_month, weight, threshold_achieved, threshold_near, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		metricID, req.PeriodYear, req.PeriodMonth, req.Weight, req.ThresholdAchieved, req.ThresholdNear, createdByUserID,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return 0, ErrDuplicateDefinition
		}
		return 0, fmt.Errorf("kpi: create definition: %w", err)
	}
	return result.LastInsertId()
}

func isDuplicateKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate entry")
}

func (r *Repository) GetDefinitionByID(ctx context.Context, id int64) (*KpiDefinition, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+definitionSelectColumns+definitionFromJoin+` WHERE kd.id = ?`, id)
	d, err := scanDefinition(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("kpi: get definition %d: %w", id, err)
	}
	return &d, nil
}

func (r *Repository) ListDefinitions(ctx context.Context, periodYear, periodMonth *int, activeOnly bool) ([]KpiDefinition, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if periodYear != nil {
		where = append(where, "kd.period_year = ?")
		args = append(args, *periodYear)
	}
	if periodMonth != nil {
		where = append(where, "kd.period_month = ?")
		args = append(args, *periodMonth)
	}
	if activeOnly {
		where = append(where, "kd.active = TRUE")
	}
	query := `SELECT ` + definitionSelectColumns + definitionFromJoin + `
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY kd.period_year DESC, kd.period_month DESC, mc.code ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("kpi: list definitions: %w", err)
	}
	defer rows.Close()
	var items []KpiDefinition
	for rows.Next() {
		d, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// DeactivateDefinition sets active=FALSE. Recompute always re-reads active definitions fresh,
// so a deactivated definition simply drops out of the next recompute for that period.
func (r *Repository) DeactivateDefinition(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE kpi_definitions SET active = FALSE WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("kpi: deactivate definition %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

/* ---------- Recompute ---------- */

type activeDefinition struct {
	ID                int64
	MetricCodeID      int64
	MetricCode        string
	Weight            int64 // hundredths of a percent
	ThresholdAchieved int64
	ThresholdNear     int64
}

// activeSalesIDs returns every active Sales-role user. Duplicated (not imported) from
// internal/target's identical helper — small per-package query helpers are this codebase's
// established convention, not a centralized shared package.
func activeSalesIDsTx(ctx context.Context, tx *sql.Tx) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT u.id FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.code = 'SALES' AND u.status = 'ACTIVE' AND u.deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("kpi: list active sales: %w", err)
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

func activeDefinitionsTx(ctx context.Context, tx *sql.Tx, periodYear, periodMonth int) ([]activeDefinition, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT kd.id, kd.metric_code_id, mc.code, kd.weight, kd.threshold_achieved, kd.threshold_near
		FROM kpi_definitions kd
		JOIN metric_codes mc ON mc.id = kd.metric_code_id
		WHERE kd.period_year = ? AND kd.period_month = ? AND kd.active = TRUE
		ORDER BY kd.id`,
		periodYear, periodMonth,
	)
	if err != nil {
		return nil, fmt.Errorf("kpi: list active definitions: %w", err)
	}
	defer rows.Close()

	var defs []activeDefinition
	for rows.Next() {
		var (
			id, metricCodeID                             int64
			metricCode                                   string
			weightStr, thresholdAchStr, thresholdNearStr string
		)
		if err := rows.Scan(&id, &metricCodeID, &metricCode, &weightStr, &thresholdAchStr, &thresholdNearStr); err != nil {
			return nil, err
		}
		weight, err := parsePercent(weightStr)
		if err != nil {
			return nil, fmt.Errorf("kpi: stored weight %q for definition %d is invalid: %w", weightStr, id, err)
		}
		thresholdAch, err := parsePercent(thresholdAchStr)
		if err != nil {
			return nil, fmt.Errorf("kpi: stored threshold_achieved %q for definition %d is invalid: %w", thresholdAchStr, id, err)
		}
		thresholdNear, err := parsePercent(thresholdNearStr)
		if err != nil {
			return nil, fmt.Errorf("kpi: stored threshold_near %q for definition %d is invalid: %w", thresholdNearStr, id, err)
		}
		defs = append(defs, activeDefinition{
			ID: id, MetricCodeID: metricCodeID, MetricCode: metricCode,
			Weight: weight, ThresholdAchieved: thresholdAch, ThresholdNear: thresholdNear,
		})
	}
	return defs, rows.Err()
}

// supportedMetricQuery returns the SQL aggregate query (grouped by sales_id) that computes
// actual_value for one metric_code, scoped to the given period. Only metrics with a direct
// sales_id column are supported this sprint (PARTNER_CALL_COUNT requires a time-scoped join to
// partner_assignments and is explicitly out of scope — see Sprint 13 plan).
func supportedMetricQuery(metricCode string) (string, bool) {
	switch metricCode {
	case "CONFIRMED_CLOSING_COUNT":
		return `SELECT sales_id, COUNT(*) FROM sales_closings
			WHERE sales_id IN (%s) AND status = 'CONFIRMED' AND deleted_at IS NULL
			AND confirmed_at >= ? AND confirmed_at < ? GROUP BY sales_id`, true
	case "CONFIRMED_CLOSING_AMOUNT":
		return `SELECT sales_id, COALESCE(SUM(final_amount), 0) FROM sales_closings
			WHERE sales_id IN (%s) AND status = 'CONFIRMED' AND deleted_at IS NULL
			AND confirmed_at >= ? AND confirmed_at < ? GROUP BY sales_id`, true
	case "CALL_CUSTOMER_COUNT":
		return `SELECT sales_id, COUNT(*) FROM customer_interactions
			WHERE sales_id IN (%s) AND deleted_at IS NULL
			AND interaction_at >= ? AND interaction_at < ? GROUP BY sales_id`, true
	case "TRAINING_COUNT":
		return `SELECT sales_id, COUNT(*) FROM training_reports
			WHERE sales_id IN (%s) AND status = 'COMPLETED' AND deleted_at IS NULL
			AND completed_at >= ? AND completed_at < ? GROUP BY sales_id`, true
	default:
		return "", false
	}
}

func actualValuesByMetricTx(ctx context.Context, tx *sql.Tx, metricCode string, salesIDs []int64, from, to time.Time) (map[int64]float64, error) {
	values := make(map[int64]float64, len(salesIDs))
	for _, id := range salesIDs {
		values[id] = 0
	}
	if len(salesIDs) == 0 {
		return values, nil
	}
	template, ok := supportedMetricQuery(metricCode)
	if !ok {
		return nil, ErrUnsupportedMetric
	}
	placeholders := make([]string, len(salesIDs))
	args := make([]any, 0, len(salesIDs)+2)
	for i, id := range salesIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	args = append(args, from, to)
	query := fmt.Sprintf(template, strings.Join(placeholders, ","))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("kpi: compute actual values for %s: %w", metricCode, err)
	}
	defer rows.Close()
	for rows.Next() {
		var salesID int64
		var value float64
		if err := rows.Scan(&salesID, &value); err != nil {
			return nil, err
		}
		values[salesID] = value
	}
	return values, rows.Err()
}

func targetValuesByMetricTx(ctx context.Context, tx *sql.Tx, metricCodeID int64, periodYear, periodMonth int) (map[int64]float64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT sales_id, target_value FROM sales_targets
		WHERE metric_code_id = ? AND period_year = ? AND period_month = ?`,
		metricCodeID, periodYear, periodMonth,
	)
	if err != nil {
		return nil, fmt.Errorf("kpi: load targets: %w", err)
	}
	defer rows.Close()
	values := make(map[int64]float64)
	for rows.Next() {
		var salesID int64
		var raw string
		if err := rows.Scan(&salesID, &raw); err != nil {
			return nil, err
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("kpi: parse target_value %q: %w", raw, err)
		}
		values[salesID] = v
	}
	return values, rows.Err()
}

// Recompute performs a full, idempotent recomputation of KPI results for one period. It runs
// entirely inside the caller-supplied transaction (the job handler's tx, or the seeder's own
// tx) so a failure leaves no partial state, and re-running for the same period always produces
// identical output (delete-then-insert scoped strictly to period_year/period_month).
func (r *Repository) Recompute(ctx context.Context, tx *sql.Tx, periodYear, periodMonth int, jobID *int64) error {
	defs, err := activeDefinitionsTx(ctx, tx, periodYear, periodMonth)
	if err != nil {
		return err
	}
	if len(defs) == 0 {
		return ErrNoActiveDefinitions
	}

	var weightSum int64
	thresholdAch := defs[0].ThresholdAchieved
	thresholdNear := defs[0].ThresholdNear
	for _, d := range defs {
		weightSum += d.Weight
		if d.ThresholdAchieved != thresholdAch || d.ThresholdNear != thresholdNear {
			return ErrInconsistentThreshold
		}
	}
	if weightSum != 10000 {
		return ErrWeightNotHundred
	}

	salesIDs, err := activeSalesIDsTx(ctx, tx)
	if err != nil {
		return err
	}

	from := time.Date(periodYear, time.Month(periodMonth), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	type metricRow struct {
		salesID        int64
		definitionID   int64
		metricCode     string
		targetValue    float64
		actualValue    float64
		achievementPct float64
		weightedScore  float64
	}
	var metricRows []metricRow
	totals := make(map[int64]float64, len(salesIDs))
	for _, id := range salesIDs {
		totals[id] = 0
	}

	for _, def := range defs {
		actuals, err := actualValuesByMetricTx(ctx, tx, def.MetricCode, salesIDs, from, to)
		if err != nil {
			return err
		}
		targets, err := targetValuesByMetricTx(ctx, tx, def.MetricCodeID, periodYear, periodMonth)
		if err != nil {
			return err
		}
		weightPct := float64(def.Weight) / 100 // back to a plain percent, e.g. 40.00

		for _, salesID := range salesIDs {
			actual := actuals[salesID]
			target, hasTarget := targets[salesID]

			// No target set for this sales/metric/period: this metric cannot claim any
			// achievement (there's nothing to have achieved against). Documented business
			// rule — see Sprint 13 plan §D.
			var achievementPct float64
			if hasTarget && target > 0 {
				achievementPct = (actual / target) * 100
			}
			cappedAchievement := achievementPct
			if cappedAchievement > 100 {
				cappedAchievement = 100
			}
			weightedScore := cappedAchievement * weightPct / 100
			totals[salesID] += weightedScore

			metricRows = append(metricRows, metricRow{
				salesID: salesID, definitionID: def.ID, metricCode: def.MetricCode,
				targetValue: target, actualValue: actual,
				achievementPct: achievementPct, weightedScore: weightedScore,
			})
		}
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM sales_kpi_metric_results WHERE period_year = ? AND period_month = ?`,
		periodYear, periodMonth,
	); err != nil {
		return fmt.Errorf("kpi: clear previous metric results: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM sales_kpi_results WHERE period_year = ? AND period_month = ?`,
		periodYear, periodMonth,
	); err != nil {
		return fmt.Errorf("kpi: clear previous results: %w", err)
	}

	for _, m := range metricRows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sales_kpi_metric_results
				(sales_id, kpi_definition_id, period_year, period_month, target_value, actual_value, achievement_pct, weighted_score)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			m.salesID, m.definitionID, periodYear, periodMonth,
			formatDecimal(m.targetValue), formatDecimal(m.actualValue), formatDecimal(m.achievementPct), formatDecimal(m.weightedScore),
		); err != nil {
			return fmt.Errorf("kpi: insert metric result: %w", err)
		}
	}

	achievedCutoff := float64(thresholdAch) / 100
	nearCutoff := float64(thresholdNear) / 100
	for _, salesID := range salesIDs {
		total := totals[salesID]
		classification := classify(total, achievedCutoff, nearCutoff)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sales_kpi_results
				(sales_id, period_year, period_month, total_score, classification, computed_at, job_id)
			VALUES (?, ?, ?, ?, ?, NOW(), ?)`,
			salesID, periodYear, periodMonth, formatDecimal(total), classification, jobID,
		); err != nil {
			return fmt.Errorf("kpi: insert result for sales %d: %w", salesID, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE sales_kpi_results r
		JOIN (
			SELECT id, RANK() OVER (ORDER BY total_score DESC) AS rnk
			FROM sales_kpi_results
			WHERE period_year = ? AND period_month = ?
		) ranked ON ranked.id = r.id
		SET r.rank_position = ranked.rnk`,
		periodYear, periodMonth,
	); err != nil {
		return fmt.Errorf("kpi: assign ranks: %w", err)
	}

	return nil
}

func formatDecimal(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// classify applies the overall per-sales-per-period classification rule: total_score is the
// weighted sum of per-metric achievement (each metric capped at 100% of its own weight), so
// achievedCutoff/nearCutoff are compared directly against it.
func classify(totalScore, achievedCutoff, nearCutoff float64) string {
	switch {
	case totalScore >= achievedCutoff:
		return ClassificationAchieved
	case totalScore >= nearCutoff:
		return ClassificationNearAchieved
	default:
		return ClassificationNotAchieved
	}
}

/* ---------- Results & Ranking (read) ---------- */

const resultSelectColumns = `
	kr.id, kr.sales_id, u.name, u.code, kr.period_year, kr.period_month,
	kr.total_score, kr.classification, kr.rank_position, kr.computed_at, kr.job_id`

func scanResult(scanner interface {
	Scan(dest ...any) error
}) (SalesKpiResult, error) {
	var r SalesKpiResult
	err := scanner.Scan(
		&r.ID, &r.SalesID, &r.SalesName, &r.SalesCode, &r.PeriodYear, &r.PeriodMonth,
		&r.TotalScore, &r.Classification, &r.RankPosition, &r.ComputedAt, &r.JobID,
	)
	return r, err
}

func (r *Repository) metricDetailsFor(ctx context.Context, salesID int64, periodYear, periodMonth int) ([]SalesKpiMetricResultResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT mc.code, mr.target_value, mr.actual_value, mr.achievement_pct, mr.weighted_score
		FROM sales_kpi_metric_results mr
		JOIN kpi_definitions kd ON kd.id = mr.kpi_definition_id
		JOIN metric_codes mc ON mc.id = kd.metric_code_id
		WHERE mr.sales_id = ? AND mr.period_year = ? AND mr.period_month = ?
		ORDER BY mc.code`,
		salesID, periodYear, periodMonth,
	)
	if err != nil {
		return nil, fmt.Errorf("kpi: metric details: %w", err)
	}
	defer rows.Close()
	var items []SalesKpiMetricResultResponse
	for rows.Next() {
		var m SalesKpiMetricResultResponse
		if err := rows.Scan(&m.MetricCode, &m.TargetValue, &m.ActualValue, &m.AchievementPct, &m.WeightedScore); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// ListResults returns per-sales KPI summaries visible to actor, with per-metric detail
// attached. Sales sees only their own row.
func (r *Repository) ListResults(ctx context.Context, actorID int64, actorRole string, params ListResultsParams) ([]SalesKpiResult, error) {
	where := []string{}
	args := []any{}

	visibility, visibilityArgs := visibilityWhere(actorID, actorRole)
	where = append(where, visibility)
	args = append(args, visibilityArgs...)

	if params.SalesID != nil {
		where = append(where, "kr.sales_id = ?")
		args = append(args, *params.SalesID)
	}
	if params.PeriodYear != nil {
		where = append(where, "kr.period_year = ?")
		args = append(args, *params.PeriodYear)
	}
	if params.PeriodMonth != nil {
		where = append(where, "kr.period_month = ?")
		args = append(args, *params.PeriodMonth)
	}

	query := `SELECT ` + resultSelectColumns + `
		FROM sales_kpi_results kr
		JOIN users u ON u.id = kr.sales_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY kr.period_year DESC, kr.period_month DESC, kr.rank_position ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("kpi: list results: %w", err)
	}
	defer rows.Close()

	var items []SalesKpiResult
	for rows.Next() {
		result, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		metrics, err := r.metricDetailsFor(ctx, items[i].SalesID, items[i].PeriodYear, items[i].PeriodMonth)
		if err != nil {
			return nil, err
		}
		items[i].Metrics = metrics
	}
	return items, nil
}

// ListRanking returns the full ranked list for a period, unfiltered by visibility (callers must
// enforce the Admin/Supervisor-only gate at the service layer).
func (r *Repository) ListRanking(ctx context.Context, periodYear, periodMonth int) ([]SalesKpiResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+resultSelectColumns+`
		FROM sales_kpi_results kr
		JOIN users u ON u.id = kr.sales_id
		WHERE kr.period_year = ? AND kr.period_month = ?
		ORDER BY kr.rank_position ASC`,
		periodYear, periodMonth,
	)
	if err != nil {
		return nil, fmt.Errorf("kpi: list ranking: %w", err)
	}
	defer rows.Close()
	var items []SalesKpiResult
	for rows.Next() {
		result, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, result)
	}
	return items, rows.Err()
}

func visibilityWhere(actorID int64, actorRole string) (string, []any) {
	switch actorRole {
	case RoleAdmin, RoleSupervisor:
		return "1 = 1", nil
	case RoleSales:
		return "kr.sales_id = ?", []any{actorID}
	default:
		return "1 = 0", nil
	}
}
