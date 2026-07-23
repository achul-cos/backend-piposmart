package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListPackages(ctx context.Context, params ListParams) ([]Package, int64, error) {
	where, args := packageWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscription_packages sp WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := packageOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)
	rows, err := r.db.QueryContext(ctx, packageSelect()+" WHERE "+where+" ORDER BY "+orderBy+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanPackages(rows, total)
}

func (r *Repository) CreatePackage(ctx context.Context, req CreatePackageRequest) (Package, error) {
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO subscription_packages (code, name, level_order, description, active)
		VALUES (?, ?, ?, ?, ?)`,
		normalizeCode(req.Code), strings.TrimSpace(req.Name), req.LevelOrder, nullableString(req.Description), active,
	)
	if err != nil {
		return Package{}, mapDuplicate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Package{}, err
	}
	return r.FindPackageByID(ctx, id)
}

func (r *Repository) FindPackageByID(ctx context.Context, id int64) (Package, error) {
	item, err := scanPackage(r.db.QueryRowContext(ctx, packageSelect()+" WHERE sp.id = ? AND sp.deleted_at IS NULL LIMIT 1", id))
	if err == sql.ErrNoRows {
		return Package{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) UpdatePackage(ctx context.Context, id int64, req UpdatePackageRequest) (Package, error) {
	current, err := r.FindPackageByID(ctx, id)
	if err != nil {
		return Package{}, err
	}
	code := current.Code
	name := current.Name
	levelOrder := current.LevelOrder
	description := current.Description
	active := current.Active
	if req.Code != nil {
		code = normalizeCode(*req.Code)
	}
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.LevelOrder != nil {
		levelOrder = *req.LevelOrder
	}
	if req.Description != nil {
		description = nullableString(*req.Description)
	}
	if req.Active != nil {
		active = *req.Active
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE subscription_packages
		SET code = ?, name = ?, level_order = ?, description = ?, active = ?
		WHERE id = ? AND deleted_at IS NULL`,
		code, name, levelOrder, description, active, id,
	)
	if err != nil {
		return Package{}, mapDuplicate(err)
	}
	return r.FindPackageByID(ctx, id)
}

func (r *Repository) DeletePackages(ctx context.Context, ids []int64) (int64, error) {
	return r.markDeleted(ctx, "subscription_packages", ids, true)
}

func (r *Repository) RestorePackages(ctx context.Context, ids []int64) (int64, error) {
	return r.markDeleted(ctx, "subscription_packages", ids, false)
}

func (r *Repository) ForceDeletePackages(ctx context.Context, ids []int64) (int64, error) {
	return r.forceDelete(ctx, "subscription_packages", ids)
}

func (r *Repository) ListPlans(ctx context.Context, params ListParams) ([]Plan, int64, error) {
	where, args := planWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM subscription_plans spl
		JOIN subscription_packages sp ON sp.id = spl.package_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := planOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)
	rows, err := r.db.QueryContext(ctx, planSelect()+" WHERE "+where+" ORDER BY "+orderBy+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanPlans(rows, total)
}

func (r *Repository) CreatePlan(ctx context.Context, req CreatePlanRequest) (Plan, error) {
	currency := normalizeCurrency(req.Currency)
	effectiveFrom, err := parseDate(req.EffectiveFrom)
	if err != nil {
		return Plan{}, err
	}
	effectiveTo, err := parseNullDate(req.EffectiveTo)
	if err != nil {
		return Plan{}, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO subscription_plans
			(package_id, code, name, tenure_months, duration_days, price, currency, effective_from, effective_to, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.PackageID,
		normalizeCode(req.Code),
		strings.TrimSpace(req.Name),
		req.TenureMonths,
		req.TenureMonths*30,
		req.Price,
		currency,
		effectiveFrom,
		effectiveTo,
		activeOrDefault(req.Active),
	)
	if err != nil {
		return Plan{}, mapDuplicate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Plan{}, err
	}
	return r.FindPlanByID(ctx, id)
}

func (r *Repository) FindPlanByID(ctx context.Context, id int64) (Plan, error) {
	item, err := scanPlan(r.db.QueryRowContext(ctx, planSelect()+" WHERE spl.id = ? AND spl.deleted_at IS NULL LIMIT 1", id))
	if err == sql.ErrNoRows {
		return Plan{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) UpdatePlan(ctx context.Context, id int64, req UpdatePlanRequest) (Plan, error) {
	current, err := r.FindPlanByID(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	packageID := current.PackageID
	code := current.Code
	name := current.Name
	tenure := current.TenureMonths
	price := current.Price
	currency := current.Currency
	effectiveFrom := current.EffectiveFrom
	effectiveTo := current.EffectiveTo
	active := current.Active
	if req.PackageID != nil {
		packageID = *req.PackageID
	}
	if req.Code != nil {
		code = normalizeCode(*req.Code)
	}
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.TenureMonths != nil {
		tenure = *req.TenureMonths
	}
	if req.Price != nil {
		price = strings.TrimSpace(*req.Price)
	}
	if req.Currency != nil {
		currency = normalizeCurrency(*req.Currency)
	}
	if req.EffectiveFrom != nil {
		effectiveFrom, err = parseDate(*req.EffectiveFrom)
		if err != nil {
			return Plan{}, err
		}
	}
	if req.EffectiveTo != nil {
		effectiveTo, err = parseNullDate(req.EffectiveTo)
		if err != nil {
			return Plan{}, err
		}
	}
	if req.Active != nil {
		active = *req.Active
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE subscription_plans
		SET package_id = ?, code = ?, name = ?, tenure_months = ?, duration_days = ?,
			price = ?, currency = ?, effective_from = ?, effective_to = ?, active = ?
		WHERE id = ? AND deleted_at IS NULL`,
		packageID, code, name, tenure, tenure*30, price, currency, effectiveFrom, effectiveTo, active, id,
	)
	if err != nil {
		return Plan{}, mapDuplicate(err)
	}
	return r.FindPlanByID(ctx, id)
}

func (r *Repository) DeletePlans(ctx context.Context, ids []int64) (int64, error) {
	return r.markDeleted(ctx, "subscription_plans", ids, true)
}

func (r *Repository) RestorePlans(ctx context.Context, ids []int64) (int64, error) {
	return r.markDeleted(ctx, "subscription_plans", ids, false)
}

func (r *Repository) ForceDeletePlans(ctx context.Context, ids []int64) (int64, error) {
	return r.forceDelete(ctx, "subscription_plans", ids)
}

func (r *Repository) ListPromotions(ctx context.Context, params ListParams) ([]Promotion, int64, error) {
	where, args := promotionWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM promotions p WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := promotionOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)
	rows, err := r.db.QueryContext(ctx, promotionSelect()+" WHERE "+where+" ORDER BY "+orderBy+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, _, err := scanPromotions(rows, total)
	if err != nil {
		return nil, 0, err
	}
	if err := r.attachBenefits(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) CreatePromotion(ctx context.Context, req CreatePromotionRequest) (Promotion, error) {
	effectiveFrom, err := parseDate(req.EffectiveFrom)
	if err != nil {
		return Promotion{}, err
	}
	effectiveTo, err := parseNullDate(req.EffectiveTo)
	if err != nil {
		return Promotion{}, err
	}
	additionalCharge := strings.TrimSpace(req.AdditionalCharge)
	if additionalCharge == "" {
		additionalCharge = "0.00"
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO promotions
			(code, name, promotion_type, charge_type, additional_charge, priority, description, effective_from, effective_to, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalizeCode(req.Code),
		strings.TrimSpace(req.Name),
		normalizeCode(req.PromotionType),
		normalizeCode(req.ChargeType),
		additionalCharge,
		req.Priority,
		nullableString(req.Description),
		effectiveFrom,
		effectiveTo,
		activeOrDefault(req.Active),
	)
	if err != nil {
		return Promotion{}, mapDuplicate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Promotion{}, err
	}
	return r.FindPromotionByID(ctx, id)
}

func (r *Repository) FindPromotionByID(ctx context.Context, id int64) (Promotion, error) {
	item, err := scanPromotion(r.db.QueryRowContext(ctx, promotionSelect()+" WHERE p.id = ? AND p.deleted_at IS NULL LIMIT 1", id))
	if err == sql.ErrNoRows {
		return Promotion{}, ErrNotFound
	}
	if err != nil {
		return Promotion{}, err
	}
	items := []Promotion{item}
	if err := r.attachBenefits(ctx, items); err != nil {
		return Promotion{}, err
	}
	return items[0], nil
}

func (r *Repository) UpdatePromotion(ctx context.Context, id int64, req UpdatePromotionRequest) (Promotion, error) {
	current, err := r.FindPromotionByID(ctx, id)
	if err != nil {
		return Promotion{}, err
	}
	code := current.Code
	name := current.Name
	promotionType := current.PromotionType
	chargeType := current.ChargeType
	additionalCharge := current.AdditionalCharge
	priority := current.Priority
	description := current.Description
	effectiveFrom := current.EffectiveFrom
	effectiveTo := current.EffectiveTo
	active := current.Active
	if req.Code != nil {
		code = normalizeCode(*req.Code)
	}
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.PromotionType != nil {
		promotionType = normalizeCode(*req.PromotionType)
	}
	if req.ChargeType != nil {
		chargeType = normalizeCode(*req.ChargeType)
	}
	if req.AdditionalCharge != nil {
		additionalCharge = strings.TrimSpace(*req.AdditionalCharge)
	}
	if req.Priority != nil {
		priority = *req.Priority
	}
	if req.Description != nil {
		description = nullableString(*req.Description)
	}
	if req.EffectiveFrom != nil {
		effectiveFrom, err = parseDate(*req.EffectiveFrom)
		if err != nil {
			return Promotion{}, err
		}
	}
	if req.EffectiveTo != nil {
		effectiveTo, err = parseNullDate(req.EffectiveTo)
		if err != nil {
			return Promotion{}, err
		}
	}
	if req.Active != nil {
		active = *req.Active
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE promotions
		SET code = ?, name = ?, promotion_type = ?, charge_type = ?, additional_charge = ?,
			priority = ?, description = ?, effective_from = ?, effective_to = ?, active = ?
		WHERE id = ? AND deleted_at IS NULL`,
		code, name, promotionType, chargeType, additionalCharge, priority, description, effectiveFrom, effectiveTo, active, id,
	)
	if err != nil {
		return Promotion{}, mapDuplicate(err)
	}
	return r.FindPromotionByID(ctx, id)
}

func (r *Repository) DeletePromotions(ctx context.Context, ids []int64) (int64, error) {
	return r.markDeleted(ctx, "promotions", ids, true)
}

func (r *Repository) RestorePromotions(ctx context.Context, ids []int64) (int64, error) {
	return r.markDeleted(ctx, "promotions", ids, false)
}

func (r *Repository) ForceDeletePromotions(ctx context.Context, ids []int64) (int64, error) {
	return r.forceDelete(ctx, "promotions", ids)
}

func (r *Repository) CreateBenefit(ctx context.Context, promotionID int64, req CreateBenefitRequest) (Benefit, error) {
	metadata := string(req.MetadataJSON)
	if strings.TrimSpace(metadata) == "" {
		metadata = "{}"
	}
	if !json.Valid([]byte(metadata)) {
		metadata = "{}"
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO promotion_benefits
			(promotion_id, benefit_type, package_id, duration_days, quantity, description, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		promotionID,
		normalizeCode(req.BenefitType),
		nullableInt64PtrInput(req.PackageID),
		nullableInt64PtrInput(req.DurationDays),
		nullableInt64PtrInput(req.Quantity),
		nullableString(req.Description),
		metadata,
	)
	if err != nil {
		return Benefit{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Benefit{}, err
	}
	return r.FindBenefitByID(ctx, id)
}

func (r *Repository) ListBenefits(ctx context.Context, promotionID int64) ([]Benefit, error) {
	rows, err := r.db.QueryContext(ctx, benefitSelect()+" WHERE pb.promotion_id = ? ORDER BY pb.id ASC", promotionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBenefits(rows)
}

func (r *Repository) FindBenefitByID(ctx context.Context, id int64) (Benefit, error) {
	item, err := scanBenefit(r.db.QueryRowContext(ctx, benefitSelect()+" WHERE pb.id = ? LIMIT 1", id))
	if err == sql.ErrNoRows {
		return Benefit{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) DeleteBenefit(ctx context.Context, promotionID, benefitID int64) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM promotion_benefits WHERE id = ? AND promotion_id = ?", benefitID, promotionID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) SetEligibility(ctx context.Context, promotionID int64, planIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM promotion_plan_eligibilities WHERE promotion_id = ?", promotionID); err != nil {
		return err
	}
	for _, planID := range planIDs {
		if _, err := tx.ExecContext(ctx, "INSERT INTO promotion_plan_eligibilities (promotion_id, plan_id) VALUES (?, ?)", promotionID, planID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) EligiblePromotions(ctx context.Context, planID int64, asOf time.Time) ([]Promotion, error) {
	rows, err := r.db.QueryContext(ctx, promotionSelect()+`
		JOIN promotion_plan_eligibilities ppe ON ppe.promotion_id = p.id
		WHERE ppe.plan_id = ?
			AND p.deleted_at IS NULL
			AND p.active = TRUE
			AND p.effective_from <= ?
			AND (p.effective_to IS NULL OR p.effective_to >= ?)
		ORDER BY CASE WHEN p.charge_type = 'FREE' THEN 0 ELSE 1 END, p.priority ASC, p.id ASC`,
		planID, asOf, asOf,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, _, err := scanPromotions(rows, 0)
	if err != nil {
		return nil, err
	}
	if err := r.attachBenefits(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) markDeleted(ctx context.Context, table string, ids []int64, deleted bool) (int64, error) {
	if len(ids) == 0 {
		return 0, ErrEmptyBulk
	}
	value := "CURRENT_TIMESTAMP"
	if !deleted {
		value = "NULL"
	}
	query := fmt.Sprintf("UPDATE %s SET deleted_at = %s WHERE id IN (%s)", table, value, placeholders(len(ids)))
	args := int64SliceToAny(ids)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) forceDelete(ctx context.Context, table string, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, ErrEmptyBulk
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", table, placeholders(len(ids)))
	result, err := r.db.ExecContext(ctx, query, int64SliceToAny(ids)...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) attachBenefits(ctx context.Context, promotions []Promotion) error {
	if len(promotions) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(promotions))
	index := map[int64]int{}
	for i := range promotions {
		ids = append(ids, promotions[i].ID)
		index[promotions[i].ID] = i
	}
	rows, err := r.db.QueryContext(ctx, benefitSelect()+" WHERE pb.promotion_id IN ("+placeholders(len(ids))+") ORDER BY pb.id ASC", int64SliceToAny(ids)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		benefit, err := scanBenefit(rows)
		if err != nil {
			return err
		}
		promotions[index[benefit.PromotionID]].Benefits = append(promotions[index[benefit.PromotionID]].Benefits, benefit)
	}
	return rows.Err()
}

func packageSelect() string {
	return "SELECT sp.id, sp.code, sp.name, sp.level_order, sp.description, sp.active, sp.created_at, sp.updated_at FROM subscription_packages sp"
}

func planSelect() string {
	return `
		SELECT spl.id, spl.package_id, sp.code, sp.name, spl.code, spl.name,
			spl.tenure_months, spl.duration_days, CAST(spl.price AS CHAR), spl.currency,
			spl.effective_from, spl.effective_to, spl.active, spl.created_at, spl.updated_at
		FROM subscription_plans spl
		JOIN subscription_packages sp ON sp.id = spl.package_id`
}

func promotionSelect() string {
	return `
		SELECT p.id, p.code, p.name, p.promotion_type, p.charge_type, CAST(p.additional_charge AS CHAR),
			p.priority, p.description, p.effective_from, p.effective_to, p.active, p.created_at, p.updated_at
		FROM promotions p`
}

func benefitSelect() string {
	return `
		SELECT pb.id, pb.promotion_id, pb.benefit_type, pb.package_id, sp.code, sp.name,
			pb.duration_days, pb.quantity, pb.description, CAST(pb.metadata_json AS CHAR), pb.created_at
		FROM promotion_benefits pb
		LEFT JOIN subscription_packages sp ON sp.id = pb.package_id`
}

func packageWhere(params ListParams) (string, []any) {
	where := []string{"sp.deleted_at IS NULL"}
	args := []any{}
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(sp.code LIKE ? OR sp.name LIKE ?)")
		args = append(args, pattern, pattern)
	}
	if params.Active != nil {
		where = append(where, "sp.active = ?")
		args = append(args, *params.Active)
	}
	return strings.Join(where, " AND "), args
}

func planWhere(params ListParams) (string, []any) {
	where := []string{"spl.deleted_at IS NULL", "sp.deleted_at IS NULL"}
	args := []any{}
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(spl.code LIKE ? OR spl.name LIKE ? OR sp.code LIKE ? OR sp.name LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if params.PackageID != nil {
		where = append(where, "spl.package_id = ?")
		args = append(args, *params.PackageID)
	}
	if params.Active != nil {
		where = append(where, "spl.active = ?")
		args = append(args, *params.Active)
	}
	if params.AsOf != nil {
		where = append(where, "spl.effective_from <= ? AND (spl.effective_to IS NULL OR spl.effective_to >= ?)")
		args = append(args, *params.AsOf, *params.AsOf)
	}
	return strings.Join(where, " AND "), args
}

func promotionWhere(params ListParams) (string, []any) {
	where := []string{"p.deleted_at IS NULL"}
	args := []any{}
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(p.code LIKE ? OR p.name LIKE ? OR p.description LIKE ?)")
		args = append(args, pattern, pattern, pattern)
	}
	if params.Active != nil {
		where = append(where, "p.active = ?")
		args = append(args, *params.Active)
	}
	if params.ChargeType != "" {
		where = append(where, "p.charge_type = ?")
		args = append(args, normalizeCode(params.ChargeType))
	}
	if params.AsOf != nil {
		where = append(where, "p.effective_from <= ? AND (p.effective_to IS NULL OR p.effective_to >= ?)")
		args = append(args, *params.AsOf, *params.AsOf)
	}
	return strings.Join(where, " AND "), args
}

func packageOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"code":        "sp.code",
		"name":        "sp.name",
		"level_order": "sp.level_order",
		"created_at":  "sp.created_at",
	}, "sp.level_order ASC, sp.id ASC")
}

func planOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"code":           "spl.code",
		"name":           "spl.name",
		"tenure_months":  "spl.tenure_months",
		"duration_days":  "spl.duration_days",
		"price":          "spl.price",
		"effective_from": "spl.effective_from",
		"created_at":     "spl.created_at",
	}, "sp.level_order ASC, spl.tenure_months ASC, spl.id ASC")
}

func promotionOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"code":           "p.code",
		"name":           "p.name",
		"priority":       "p.priority",
		"charge_type":    "p.charge_type",
		"effective_from": "p.effective_from",
		"created_at":     "p.created_at",
	}, "CASE WHEN p.charge_type = 'FREE' THEN 0 ELSE 1 END, p.priority ASC, p.id ASC")
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPackages(rows *sql.Rows, total int64) ([]Package, int64, error) {
	items := []Package{}
	for rows.Next() {
		item, err := scanPackage(rows)
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

func scanPackage(row scanner) (Package, error) {
	var item Package
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.LevelOrder, &item.Description, &item.Active, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanPlans(rows *sql.Rows, total int64) ([]Plan, int64, error) {
	items := []Plan{}
	for rows.Next() {
		item, err := scanPlan(rows)
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

func scanPlan(row scanner) (Plan, error) {
	var item Plan
	err := row.Scan(
		&item.ID, &item.PackageID, &item.PackageCode, &item.PackageName, &item.Code, &item.Name,
		&item.TenureMonths, &item.DurationDays, &item.Price, &item.Currency,
		&item.EffectiveFrom, &item.EffectiveTo, &item.Active, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanPromotions(rows *sql.Rows, total int64) ([]Promotion, int64, error) {
	items := []Promotion{}
	for rows.Next() {
		item, err := scanPromotion(rows)
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

func scanPromotion(row scanner) (Promotion, error) {
	var item Promotion
	err := row.Scan(
		&item.ID, &item.Code, &item.Name, &item.PromotionType, &item.ChargeType,
		&item.AdditionalCharge, &item.Priority, &item.Description, &item.EffectiveFrom,
		&item.EffectiveTo, &item.Active, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanBenefits(rows *sql.Rows) ([]Benefit, error) {
	items := []Benefit{}
	for rows.Next() {
		item, err := scanBenefit(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanBenefit(row scanner) (Benefit, error) {
	var item Benefit
	err := row.Scan(
		&item.ID, &item.PromotionID, &item.BenefitType, &item.PackageID,
		&item.PackageCode, &item.PackageName, &item.DurationDays, &item.Quantity,
		&item.Description, &item.MetadataJSON, &item.CreatedAt,
	)
	return item, err
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return parsed, nil
}

func parseNullDate(value *string) (sql.NullTime, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return sql.NullTime{}, nil
	}
	parsed, err := parseDate(*value)
	if err != nil {
		return sql.NullTime{}, err
	}
	return sql.NullTime{Time: parsed, Valid: true}, nil
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

func placeholders(count int) string {
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func int64SliceToAny(ids []int64) []any {
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	return args
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableInt64PtrInput(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeCurrency(value string) string {
	value = normalizeCode(value)
	if value == "" {
		return "IDR"
	}
	return value
}

func activeOrDefault(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func like(value string) string {
	return "%" + strings.TrimSpace(value) + "%"
}

func mapDuplicate(err error) error {
	if strings.Contains(err.Error(), "Duplicate entry") {
		return ErrCodeExists
	}
	return err
}
