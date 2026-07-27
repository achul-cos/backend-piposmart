package closing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListClosings(ctx context.Context, actor identity.User, params ListParams) ([]Closing, int64, error) {
	where, args := closingWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sales_closings sc
		LEFT JOIN customer_leads cl ON cl.id = sc.lead_id
		LEFT JOIN owners o ON o.id = sc.owner_id AND o.deleted_at IS NULL
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := closingOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)
	rows, err := r.db.QueryContext(ctx, closingSelect()+`
		WHERE `+where+`
		ORDER BY `+orderBy+`
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanClosings(rows, total)
}

func (r *Repository) FindClosingByID(ctx context.Context, actor identity.User, id int64) (Closing, error) {
	where, args := closingWhere(actor, ListParams{Scope: ScopeActive})
	args = append([]any{id}, args...)
	item, err := scanClosing(r.db.QueryRowContext(ctx, closingSelect()+`
		WHERE sc.id = ? AND `+where+`
		LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return Closing{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) CreateClosing(ctx context.Context, actor identity.User, leadID int64, req CreateClosingRequest) (Closing, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Closing{}, err
	}
	defer tx.Rollback()

	current, err := r.lockLead(ctx, tx, leadID)
	if err != nil {
		return Closing{}, err
	}
	if !salesOwnsLead(actor, current) {
		return Closing{}, ErrForbidden
	}
	if !current.ActiveSalesID.Valid {
		return Closing{}, ErrLeadHasNoPIC
	}
	if err := r.ensureNoOpenClosing(ctx, tx, leadID); err != nil {
		return Closing{}, err
	}

	closedAt := time.Now().UTC()
	if req.ClosedAt != nil {
		closedAt = req.ClosedAt.UTC()
	}
	pkgSnapshot, planSnapshot, err := r.findPlanSnapshot(ctx, tx, req.PlanID, closedAt)
	if err != nil {
		return Closing{}, err
	}

	additionalCharge := "0.00"
	var promotionID sql.NullInt64
	var promotionSnapshot PromotionSnapshot
	var promotionSnapshotJSON sql.NullString
	if req.PromotionID != nil {
		promotionSnapshot, err = r.findEligiblePromotionSnapshot(ctx, tx, *req.PromotionID, req.PlanID, closedAt)
		if err != nil {
			return Closing{}, err
		}
		additionalCharge = promotionSnapshot.AdditionalCharge
		promotionID = sql.NullInt64{Int64: promotionSnapshot.ID, Valid: true}
		promotionBytes, err := json.Marshal(promotionSnapshot)
		if err != nil {
			return Closing{}, err
		}
		promotionSnapshotJSON = sql.NullString{String: string(promotionBytes), Valid: true}
	}

	uniqueTransferCode := defaultUniqueTransferCode(current.ID, closedAt)
	if req.UniqueTransferCode != nil {
		uniqueTransferCode = *req.UniqueTransferCode
	}
	calc, err := calculateFinalAmount(planSnapshot.Price, req.DiscountAmount, additionalCharge, uniqueTransferCode)
	if err != nil {
		return Closing{}, err
	}
	packageBytes, err := json.Marshal(pkgSnapshot)
	if err != nil {
		return Closing{}, err
	}
	planBytes, err := json.Marshal(planSnapshot)
	if err != nil {
		return Closing{}, err
	}

	code := fmt.Sprintf("CLS-%s-%06d-%06d", closedAt.Format("20060102"), current.ID, closedAt.Nanosecond()/1000)
	result, err := tx.ExecContext(ctx, `
		INSERT INTO sales_closings
			(code, lead_id, owner_id, outlet_id, sales_id, supervisor_id, package_id, plan_id, promotion_id,
			 package_snapshot_json, plan_snapshot_json, promotion_snapshot_json, tenure_months, duration_days,
			 base_price, discount_amount, additional_charge, unique_transfer_code, final_amount, currency,
			 status, note, closed_at, created_by_user_id, updated_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		code,
		current.ID,
		current.OwnerID,
		current.OutletID,
		current.ActiveSalesID,
		current.SupervisorID,
		sql.NullInt64{Int64: pkgSnapshot.ID, Valid: true},
		sql.NullInt64{Int64: planSnapshot.ID, Valid: true},
		promotionID,
		string(packageBytes),
		string(planBytes),
		promotionSnapshotJSON,
		planSnapshot.TenureMonths,
		planSnapshot.DurationDays,
		calc.BasePrice,
		calc.DiscountAmount,
		calc.AdditionalCharge,
		calc.UniqueTransferCode,
		calc.FinalAmount,
		planSnapshot.Currency,
		StatusPending,
		nullableString(req.Note),
		closedAt,
		actor.ID,
		actor.ID,
	)
	if err != nil {
		return Closing{}, err
	}
	closingID, err := result.LastInsertId()
	if err != nil {
		return Closing{}, err
	}

	remark, err := r.resolveClosingRemark(ctx, tx)
	if err != nil {
		return Closing{}, err
	}
	if err := r.insertClosingInteraction(ctx, tx, actor, current, req, closedAt, closingID, remark); err != nil {
		return Closing{}, err
	}
	if err := r.applyClosingLeadState(ctx, tx, actor, current, closedAt, closingID, remark); err != nil {
		return Closing{}, err
	}

	if err := tx.Commit(); err != nil {
		return Closing{}, err
	}
	return r.FindClosingByID(ctx, actor, closingID)
}

func (r *Repository) ConfirmClosing(ctx context.Context, actor identity.User, id int64, req UpdateClosingStatusRequest) (Closing, error) {
	return r.updateStatus(ctx, actor, id, StatusConfirmed, nullableString(req.Note), sql.NullString{})
}

func (r *Repository) RejectClosing(ctx context.Context, actor identity.User, id int64, req UpdateClosingStatusRequest) (Closing, error) {
	reason := nullableString(req.Reason)
	if !reason.Valid {
		reason = nullableString(req.Note)
	}
	return r.updateStatus(ctx, actor, id, StatusRejected, nullableString(req.Note), reason)
}

func (r *Repository) DeleteClosings(ctx context.Context, actor identity.User, ids []int64) (int64, error) {
	return r.markDeleted(ctx, actor, ids, true)
}

func (r *Repository) RestoreClosings(ctx context.Context, actor identity.User, ids []int64) (int64, error) {
	return r.markDeleted(ctx, actor, ids, false)
}

func (r *Repository) ForceDeleteClosings(ctx context.Context, actor identity.User, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, ErrInvalidRequest
	}
	where, args := mutationWhere(actor, ids)
	result, err := r.db.ExecContext(ctx, `
		DELETE sc FROM sales_closings sc
		LEFT JOIN customer_leads cl ON cl.id = sc.lead_id
		WHERE `+where, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) updateStatus(ctx context.Context, actor identity.User, id int64, nextStatus string, note sql.NullString, reason sql.NullString) (Closing, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Closing{}, err
	}
	defer tx.Rollback()

	current, err := r.lockClosing(ctx, tx, id)
	if err != nil {
		return Closing{}, err
	}
	if !canManageClosing(actor, current) {
		return Closing{}, ErrForbidden
	}
	if current.Status != StatusPending {
		return Closing{}, ErrInvalidStatus
	}

	now := time.Now().UTC()
	confirmedAt := sql.NullTime{}
	rejectedAt := sql.NullTime{}
	if nextStatus == StatusConfirmed {
		confirmedAt = sql.NullTime{Time: now, Valid: true}
	} else if nextStatus == StatusRejected {
		rejectedAt = sql.NullTime{Time: now, Valid: true}
	} else {
		return Closing{}, ErrInvalidStatus
	}
	if note.Valid && strings.TrimSpace(current.Note.String) != "" {
		note = sql.NullString{String: strings.TrimSpace(current.Note.String) + "\n" + note.String, Valid: true}
	} else if !note.Valid {
		note = current.Note
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE sales_closings
		SET status = ?, note = ?, rejection_reason = ?, confirmed_at = ?, rejected_at = ?, updated_by_user_id = ?
		WHERE id = ? AND deleted_at IS NULL`,
		nextStatus, note, reason, confirmedAt, rejectedAt, actor.ID, id,
	); err != nil {
		return Closing{}, err
	}
	if err := tx.Commit(); err != nil {
		return Closing{}, err
	}
	return r.FindClosingByID(ctx, actor, id)
}

func (r *Repository) markDeleted(ctx context.Context, actor identity.User, ids []int64, deleted bool) (int64, error) {
	if len(ids) == 0 {
		return 0, ErrInvalidRequest
	}
	value := "CURRENT_TIMESTAMP"
	if !deleted {
		value = "NULL"
	}
	where, args := mutationWhere(actor, ids)
	result, err := r.db.ExecContext(ctx, `
		UPDATE sales_closings sc
		LEFT JOIN customer_leads cl ON cl.id = sc.lead_id
		SET sc.deleted_at = `+value+`
		WHERE `+where, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) lockLead(ctx context.Context, tx *sql.Tx, id int64) (LeadState, error) {
	var item LeadState
	err := tx.QueryRowContext(ctx, `
		SELECT id, owner_id, outlet_id, active_sales_id, current_owner_user_id,
			current_owner_role, supervisor_id, stage, status, current_score
		FROM customer_leads
		WHERE id = ? AND deleted_at IS NULL
		FOR UPDATE`, id).
		Scan(
			&item.ID,
			&item.OwnerID,
			&item.OutletID,
			&item.ActiveSalesID,
			&item.CurrentOwnerUserID,
			&item.CurrentOwnerRole,
			&item.SupervisorID,
			&item.Stage,
			&item.Status,
			&item.CurrentScore,
		)
	if err == sql.ErrNoRows {
		return LeadState{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) lockClosing(ctx context.Context, tx *sql.Tx, id int64) (Closing, error) {
	item, err := scanClosing(tx.QueryRowContext(ctx, closingSelect()+`
		WHERE sc.id = ? AND sc.deleted_at IS NULL
		FOR UPDATE`, id))
	if err == sql.ErrNoRows {
		return Closing{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ensureNoOpenClosing(ctx context.Context, tx *sql.Tx, leadID int64) error {
	var total int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sales_closings
		WHERE lead_id = ? AND deleted_at IS NULL AND status IN ('PENDING_RECONCILIATION', 'CONFIRMED')`, leadID).Scan(&total); err != nil {
		return err
	}
	if total > 0 {
		return ErrAlreadyHasClosing
	}
	return nil
}

func (r *Repository) findPlanSnapshot(ctx context.Context, tx *sql.Tx, planID int64, asOf time.Time) (PackageSnapshot, PlanSnapshot, error) {
	var pkg PackageSnapshot
	var plan PlanSnapshot
	var effectiveFrom time.Time
	var effectiveTo sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT spl.id, spl.code, spl.name, spl.tenure_months, spl.duration_days, CAST(spl.price AS CHAR), spl.currency,
			spl.effective_from, spl.effective_to, sp.id, sp.code, sp.name, sp.level_order
		FROM subscription_plans spl
		JOIN subscription_packages sp ON sp.id = spl.package_id
		WHERE spl.id = ?
			AND spl.deleted_at IS NULL
			AND sp.deleted_at IS NULL
			AND spl.active = TRUE
			AND sp.active = TRUE
			AND spl.effective_from <= DATE(?)
			AND (spl.effective_to IS NULL OR spl.effective_to >= DATE(?))
		LIMIT 1`, planID, asOf, asOf).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.TenureMonths,
		&plan.DurationDays,
		&plan.Price,
		&plan.Currency,
		&effectiveFrom,
		&effectiveTo,
		&pkg.ID,
		&pkg.Code,
		&pkg.Name,
		&pkg.LevelOrder,
	)
	if err == sql.ErrNoRows {
		return PackageSnapshot{}, PlanSnapshot{}, ErrNotFound
	}
	if err != nil {
		return PackageSnapshot{}, PlanSnapshot{}, err
	}
	plan.EffectiveFrom = effectiveFrom.Format("2006-01-02")
	if effectiveTo.Valid {
		plan.EffectiveTo = effectiveTo.Time.Format("2006-01-02")
	}
	return pkg, plan, nil
}

func (r *Repository) findEligiblePromotionSnapshot(ctx context.Context, tx *sql.Tx, promotionID, planID int64, asOf time.Time) (PromotionSnapshot, error) {
	var promo PromotionSnapshot
	var effectiveFrom time.Time
	var effectiveTo sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT p.id, p.code, p.name, p.promotion_type, p.charge_type, CAST(p.additional_charge AS CHAR),
			p.priority, p.effective_from, p.effective_to
		FROM promotions p
		JOIN promotion_plan_eligibilities ppe ON ppe.promotion_id = p.id
		WHERE p.id = ?
			AND ppe.plan_id = ?
			AND p.deleted_at IS NULL
			AND p.active = TRUE
			AND p.effective_from <= DATE(?)
			AND (p.effective_to IS NULL OR p.effective_to >= DATE(?))
		LIMIT 1`, promotionID, planID, asOf, asOf).Scan(
		&promo.ID,
		&promo.Code,
		&promo.Name,
		&promo.PromotionType,
		&promo.ChargeType,
		&promo.AdditionalCharge,
		&promo.Priority,
		&effectiveFrom,
		&effectiveTo,
	)
	if err == sql.ErrNoRows {
		return PromotionSnapshot{}, ErrInvalidPromotion
	}
	if err != nil {
		return PromotionSnapshot{}, err
	}
	promo.EffectiveFrom = effectiveFrom.Format("2006-01-02")
	if effectiveTo.Valid {
		promo.EffectiveTo = effectiveTo.Time.Format("2006-01-02")
	}
	benefits, err := r.findPromotionBenefitSnapshots(ctx, tx, promotionID)
	if err != nil {
		return PromotionSnapshot{}, err
	}
	promo.Benefits = benefits
	return promo, nil
}

func (r *Repository) findPromotionBenefitSnapshots(ctx context.Context, tx *sql.Tx, promotionID int64) ([]BenefitSnapshot, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT pb.id, pb.benefit_type, pb.package_id, sp.code, sp.name, pb.duration_days, pb.quantity,
			pb.description, CAST(pb.metadata_json AS CHAR)
		FROM promotion_benefits pb
		LEFT JOIN subscription_packages sp ON sp.id = pb.package_id
		WHERE pb.promotion_id = ?
		ORDER BY pb.id ASC`, promotionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BenefitSnapshot{}
	for rows.Next() {
		var item BenefitSnapshot
		var packageID sql.NullInt64
		var packageCode, packageName, description, metadata sql.NullString
		var durationDays, quantity sql.NullInt64
		if err := rows.Scan(&item.ID, &item.BenefitType, &packageID, &packageCode, &packageName, &durationDays, &quantity, &description, &metadata); err != nil {
			return nil, err
		}
		if packageID.Valid {
			value := packageID.Int64
			item.PackageID = &value
			item.PackageCode = packageCode.String
			item.PackageName = packageName.String
		}
		if durationDays.Valid {
			value := durationDays.Int64
			item.DurationDays = &value
		}
		if quantity.Valid {
			value := quantity.Int64
			item.Quantity = &value
		}
		item.Description = description.String
		if metadata.Valid && metadata.String != "" {
			var parsed any
			if err := json.Unmarshal([]byte(metadata.String), &parsed); err == nil {
				item.MetadataJSON = parsed
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) resolveClosingRemark(ctx context.Context, tx *sql.Tx) (RemarkReason, error) {
	var item RemarkReason
	err := tx.QueryRowContext(ctx, `
		SELECT id, score, code, label
		FROM remark_reasons
		WHERE score = 3 AND active = TRUE
		ORDER BY id
		LIMIT 1`).Scan(&item.ID, &item.Score, &item.Code, &item.Label)
	if err == sql.ErrNoRows {
		return RemarkReason{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) insertClosingInteraction(ctx context.Context, tx *sql.Tx, actor identity.User, current LeadState, req CreateClosingRequest, closedAt time.Time, closingID int64, remark RemarkReason) error {
	interactionType, err := normalizeInteractionType(req.InteractionType)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO customer_interactions
			(lead_id, owner_id, outlet_id, sales_id, supervisor_id, interaction_type, interaction_at,
			 contact_name, contact_phone, remark_reason_id, remark_score, remark_code, remark_label,
			 note, customer_response, stage_before, stage_after, status_before, status_after,
			 score_before, score_after, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		current.ID,
		current.OwnerID,
		current.OutletID,
		current.ActiveSalesID,
		current.SupervisorID,
		interactionType,
		closedAt,
		nullableString(req.ContactName),
		nullableString(req.ContactPhone),
		remark.ID,
		remark.Score,
		remark.Code,
		remark.Label,
		nullableString(req.Note),
		nullableString(req.CustomerResponse),
		nullableString(current.Stage),
		StageClosing,
		nullableString(current.Status),
		StatusOpen,
		current.CurrentScore,
		sql.NullInt64{Int64: 3, Valid: true},
		actor.ID,
	)
	return err
}

func (r *Repository) applyClosingLeadState(ctx context.Context, tx *sql.Tx, actor identity.User, current LeadState, closedAt time.Time, closingID int64, remark RemarkReason) error {
	nextScore := sql.NullInt64{Int64: 3, Valid: true}
	if current.Stage != StageClosing || current.Status != StatusOpen || !current.CurrentScore.Valid || current.CurrentScore.Int64 != 3 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO lead_stage_histories
				(lead_id, owner_id, from_stage, to_stage, from_status, to_status, from_score, to_score,
				 changed_by_user_id, source_type, source_id, reason)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'CLOSING', ?, ?)`,
			current.ID,
			current.OwnerID,
			nullableString(current.Stage),
			StageClosing,
			nullableString(current.Status),
			StatusOpen,
			current.CurrentScore,
			nextScore,
			actor.ID,
			closingID,
			nullableString(remark.Label),
		); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE customer_leads
		SET stage = ?, status = ?, current_score = ?, last_interaction_at = ?, next_follow_up_at = NULL,
			invalidated_at = NULL, invalidated_by_sales_id = NULL
		WHERE id = ?`,
		StageClosing,
		StatusOpen,
		nextScore,
		closedAt,
		current.ID,
	)
	return err
}

func closingSelect() string {
	return `
		SELECT
			sc.id, sc.code, sc.lead_id, cl.code, sc.owner_id, o.code, o.name, sc.outlet_id,
			sc.sales_id, su.name, sc.supervisor_id, spvu.name,
			sc.package_id, sp.code, sp.name, sc.plan_id, spl.code, spl.name,
			sc.promotion_id, p.code, p.name,
			CAST(sc.package_snapshot_json AS CHAR), CAST(sc.plan_snapshot_json AS CHAR), CAST(sc.promotion_snapshot_json AS CHAR),
			sc.tenure_months, sc.duration_days, CAST(sc.base_price AS CHAR), CAST(sc.discount_amount AS CHAR),
			CAST(sc.additional_charge AS CHAR), sc.unique_transfer_code, CAST(sc.final_amount AS CHAR), sc.currency,
			sc.status, sc.note, sc.rejection_reason, sc.closed_at, sc.confirmed_at, sc.rejected_at,
			sc.created_by_user_id, cbu.name, sc.updated_by_user_id, ubu.name, sc.created_at, sc.updated_at
		FROM sales_closings sc
		LEFT JOIN customer_leads cl ON cl.id = sc.lead_id
		LEFT JOIN owners o ON o.id = sc.owner_id AND o.deleted_at IS NULL
		LEFT JOIN users su ON su.id = sc.sales_id
		LEFT JOIN users spvu ON spvu.id = sc.supervisor_id
		LEFT JOIN subscription_packages sp ON sp.id = sc.package_id
		LEFT JOIN subscription_plans spl ON spl.id = sc.plan_id
		LEFT JOIN promotions p ON p.id = sc.promotion_id
		LEFT JOIN users cbu ON cbu.id = sc.created_by_user_id
		LEFT JOIN users ubu ON ubu.id = sc.updated_by_user_id`
}

func closingWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{}
	switch params.Scope {
	case ScopeDeleted:
		where = append(where, "sc.deleted_at IS NOT NULL")
	case ScopeAll:
		where = append(where, "1 = 1")
	default:
		where = append(where, "sc.deleted_at IS NULL")
	}
	visibility, visibilityArgs := visibilityWhere(actor)
	where = append(where, visibility)
	args := []any{}
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(sc.code LIKE ? OR o.name LIKE ? OR cl.code LIKE ?)")
		args = append(args, pattern, pattern, pattern)
	}
	if params.Status != "" {
		where = append(where, "sc.status = ?")
		args = append(args, params.Status)
	}
	if params.LeadID != nil {
		where = append(where, "sc.lead_id = ?")
		args = append(args, *params.LeadID)
	}
	if params.OwnerID != nil {
		where = append(where, "sc.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.SalesID != nil {
		where = append(where, "sc.sales_id = ?")
		args = append(args, *params.SalesID)
	}
	if params.SupervisorID != nil {
		where = append(where, "sc.supervisor_id = ?")
		args = append(args, *params.SupervisorID)
	}
	if params.PlanID != nil {
		where = append(where, "sc.plan_id = ?")
		args = append(args, *params.PlanID)
	}
	if params.ClosedFrom != nil {
		where = append(where, "sc.closed_at >= ?")
		args = append(args, *params.ClosedFrom)
	}
	if params.ClosedTo != nil {
		where = append(where, "sc.closed_at <= ?")
		args = append(args, *params.ClosedTo)
	}
	return strings.Join(where, " AND "), args
}

func mutationWhere(actor identity.User, ids []int64) (string, []any) {
	visibility, visibilityArgs := visibilityWhere(actor)
	args := int64SliceToAny(ids)
	args = append(args, visibilityArgs...)
	return "sc.id IN (" + placeholders(len(ids)) + ") AND " + visibility, args
}

func visibilityWhere(actor identity.User) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "1 = 1", nil
	case RoleSupervisor:
		return "(sc.supervisor_id = ? OR cl.current_owner_user_id = ? OR cl.supervisor_id = ?)", []any{actor.ID, actor.ID, actor.ID}
	case RoleSales:
		return "(sc.sales_id = ? OR (cl.current_owner_role = 'SALES' AND cl.current_owner_user_id = ?))", []any{actor.ID, actor.ID}
	default:
		return "1 = 0", nil
	}
}

func closingOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"closed_at":    "sc.closed_at",
		"created_at":   "sc.created_at",
		"updated_at":   "sc.updated_at",
		"status":       "sc.status",
		"final_amount": "sc.final_amount",
		"code":         "sc.code",
	}, "sc.closed_at DESC, sc.id DESC")
}

func scanClosings(rows *sql.Rows, total int64) ([]Closing, int64, error) {
	items := []Closing{}
	for rows.Next() {
		item, err := scanClosing(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanClosing(row scanner) (Closing, error) {
	var item Closing
	err := row.Scan(
		&item.ID,
		&item.Code,
		&item.LeadID,
		&item.LeadCode,
		&item.OwnerID,
		&item.OwnerCode,
		&item.OwnerName,
		&item.OutletID,
		&item.SalesID,
		&item.SalesName,
		&item.SupervisorID,
		&item.SupervisorName,
		&item.PackageID,
		&item.PackageCode,
		&item.PackageName,
		&item.PlanID,
		&item.PlanCode,
		&item.PlanName,
		&item.PromotionID,
		&item.PromotionCode,
		&item.PromotionName,
		&item.PackageSnapshotJSON,
		&item.PlanSnapshotJSON,
		&item.PromotionSnapshotJSON,
		&item.TenureMonths,
		&item.DurationDays,
		&item.BasePrice,
		&item.DiscountAmount,
		&item.AdditionalCharge,
		&item.UniqueTransferCode,
		&item.FinalAmount,
		&item.Currency,
		&item.Status,
		&item.Note,
		&item.RejectionReason,
		&item.ClosedAt,
		&item.ConfirmedAt,
		&item.RejectedAt,
		&item.CreatedByUserID,
		&item.CreatedByName,
		&item.UpdatedByUserID,
		&item.UpdatedByName,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func orderBy(sort string, allowed map[string]string, fallback string) (string, error) {
	sort = strings.TrimSpace(sort)
	if sort == "" {
		return fallback, nil
	}
	direction := "ASC"
	if strings.HasPrefix(sort, "-") {
		direction = "DESC"
		sort = strings.TrimPrefix(sort, "-")
	}
	column, ok := allowed[sort]
	if !ok {
		return "", ErrInvalidSort
	}
	return column + " " + direction, nil
}

func salesOwnsLead(actor identity.User, item LeadState) bool {
	return actor.RoleCode == RoleSales &&
		item.CurrentOwnerRole == RoleSales &&
		item.CurrentOwnerUserID.Valid &&
		item.CurrentOwnerUserID.Int64 == actor.ID &&
		item.ActiveSalesID.Valid &&
		item.ActiveSalesID.Int64 == actor.ID
}

func canManageClosing(actor identity.User, item Closing) bool {
	switch actor.RoleCode {
	case RoleAdmin:
		return true
	case RoleSupervisor:
		return item.SupervisorID.Valid && item.SupervisorID.Int64 == actor.ID
	default:
		return false
	}
}

func normalizeInteractionType(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return InteractionCall, nil
	}
	switch value {
	case InteractionCall, InteractionChat:
		return value, nil
	default:
		return "", ErrInvalidRequest
	}
}

func normalizeStatus(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func defaultUniqueTransferCode(leadID int64, closedAt time.Time) int {
	return int((closedAt.UnixNano()+leadID)%900) + 100
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func int64SliceToAny(values []int64) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func like(value string) string {
	return "%" + strings.TrimSpace(value) + "%"
}
