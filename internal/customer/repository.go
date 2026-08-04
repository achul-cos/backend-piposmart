package customer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

type queryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOwner(ctx context.Context, actor Actor, req CreateOwnerRequest, normalizedPhone string) (Owner, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Owner{}, err
	}
	defer tx.Rollback()

	owner, err := r.createOwner(ctx, tx, actor, req, normalizedPhone)
	if err != nil {
		return Owner{}, err
	}
	if err := tx.Commit(); err != nil {
		return Owner{}, err
	}
	return owner, nil
}

func (r *Repository) CreateOwners(ctx context.Context, actor Actor, requests []CreateOwnerRequest, normalizedPhones []string) ([]Owner, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	owners := make([]Owner, 0, len(requests))
	for index, req := range requests {
		owner, err := r.createOwner(ctx, tx, actor, req, normalizedPhones[index])
		if err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return owners, nil
}

func (r *Repository) createOwner(ctx context.Context, q queryExecutor, actor Actor, req CreateOwnerRequest, normalizedPhone string) (Owner, error) {
	query := `
		INSERT INTO owners (code, name, phone, email, brand_name, province, city, district, sub_district, address, status`
	args := []any{
		trim(req.Code),
		trim(req.Name),
		nullableString(normalizedPhone),
		nullableString(req.Email),
		nullableString(req.BrandName),
		nullableString(req.Province),
		nullableString(req.City),
		nullableString(req.District),
		nullableString(req.SubDistrict),
		nullableString(req.Address),
	}
	values := `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE'`
	if req.CreatedAt != nil {
		query += `, created_at`
		values += `, ?`
		args = append(args, req.CreatedAt.UTC())
	}
	query += values + `)`
	result, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return Owner{}, mapDuplicateError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Owner{}, err
	}
	if err := r.createLeadForOwner(ctx, q, id, actor); err != nil {
		return Owner{}, err
	}
	return r.findOwnerByID(ctx, q, id, false)
}

func (r *Repository) FindOwnerByID(ctx context.Context, actor Actor, id int64) (Owner, error) {
	if actor.RoleCode == RoleAdmin {
		return r.findOwnerByID(ctx, r.db, id, false)
	}
	return r.findOwnerByIDVisible(ctx, r.db, actor, id, false)
}

func (r *Repository) findOwnerByIDRaw(ctx context.Context, id int64) (Owner, error) {
	return r.findOwnerByID(ctx, r.db, id, false)
}

func (r *Repository) findOwnerByID(ctx context.Context, q queryExecutor, id int64, includeDeleted bool) (Owner, error) {
	deletedClause := "AND o.deleted_at IS NULL"
	if includeDeleted {
		deletedClause = ""
	}
	owner, err := scanOwner(q.QueryRowContext(ctx, `
		SELECT
			o.id, o.code, o.name, o.phone, o.email, o.brand_name, o.province, o.city, o.district, o.sub_district,
			o.address, o.status, COUNT(ot.id) AS outlet_count,
			(
				SELECT COUNT(DISTINCT s.outlet_id)
				FROM subscriptions s
				JOIN outlets ot4 ON ot4.id = s.outlet_id
				WHERE ot4.owner_id = o.id AND s.deleted_at IS NULL
			) AS subscribed_outlet_count,
			o.created_at, o.updated_at
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		WHERE o.id = ? `+deletedClause+`
		GROUP BY o.id`, id))
	if err == sql.ErrNoRows {
		return Owner{}, ErrNotFound
	}
	return owner, err
}

func (r *Repository) findOwnerByIDVisible(ctx context.Context, q queryExecutor, actor Actor, id int64, includeDeleted bool) (Owner, error) {
	where := []string{"o.id = ?"}
	args := []any{id}
	if !includeDeleted {
		where = append(where, "o.deleted_at IS NULL")
	}
	visibility, visibilityArgs := ownerVisibilityWhere(actor)
	where = append(where, visibility)
	args = append(args, visibilityArgs...)

	owner, err := scanOwner(q.QueryRowContext(ctx, `
		SELECT
			o.id, o.code, o.name, o.phone, o.email, o.brand_name, o.province, o.city, o.district, o.sub_district,
			o.address, o.status, COUNT(ot.id) AS outlet_count,
			(
				SELECT COUNT(DISTINCT s.outlet_id)
				FROM subscriptions s
				JOIN outlets ot4 ON ot4.id = s.outlet_id
				WHERE ot4.owner_id = o.id AND s.deleted_at IS NULL
			) AS subscribed_outlet_count,
			o.created_at, o.updated_at
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY o.id`, args...))
	if err == sql.ErrNoRows {
		return Owner{}, ErrNotFound
	}
	return owner, err
}

func (r *Repository) ListOwners(ctx context.Context, actor Actor, params ListParams) ([]Owner, int64, error) {
	where, args := ownerWhere(actor, params)
	countQuery := "SELECT COUNT(*) FROM owners o WHERE " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := ownerOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := `
		SELECT
			o.id, o.code, o.name, o.phone, o.email, o.brand_name, o.province, o.city,
			o.district, o.sub_district, o.address, o.status, COUNT(ot.id) AS outlet_count,
			(
				SELECT COUNT(DISTINCT s.outlet_id)
				FROM subscriptions s
				JOIN outlets ot4 ON ot4.id = s.outlet_id
				WHERE ot4.owner_id = o.id AND s.deleted_at IS NULL
			) AS subscribed_outlet_count,
			o.created_at, o.updated_at
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		LEFT JOIN wallet_accounts wa ON wa.owner_id = o.id
		WHERE ` + where + `
		GROUP BY o.id
		ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		args = append(args, params.Limit, offset)
		query += `
		LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var owners []Owner
	for rows.Next() {
		owner, err := scanOwner(rows)
		if err != nil {
			return nil, 0, err
		}
		owners = append(owners, owner)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return owners, total, nil
}

func (r *Repository) UpdateOwner(ctx context.Context, id int64, req UpdateOwnerRequest, normalizedPhone *string) (Owner, error) {
	return r.updateOwner(ctx, r.db, OwnerUpdateInput{
		ID:              id,
		Request:         req,
		NormalizedPhone: normalizedPhone,
	})
}

func (r *Repository) UpdateOwners(ctx context.Context, updates []OwnerUpdateInput) ([]Owner, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	owners := make([]Owner, 0, len(updates))
	for _, update := range updates {
		owner, err := r.updateOwner(ctx, tx, update)
		if err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return owners, nil
}

func (r *Repository) updateOwner(ctx context.Context, q queryExecutor, input OwnerUpdateInput) (Owner, error) {
	current, err := r.findOwnerByID(ctx, q, input.ID, false)
	if err != nil {
		return Owner{}, err
	}

	code := current.Code
	name := current.Name
	phone := current.Phone
	email := current.Email
	brandName := current.BrandName
	province := current.Province
	city := current.City
	district := current.District
	subDistrict := current.SubDistrict
	address := current.Address

	if input.Request.Code != nil {
		code = trim(*input.Request.Code)
	}
	if input.Request.Name != nil {
		name = trim(*input.Request.Name)
	}
	if input.NormalizedPhone != nil {
		phone = nullableString(*input.NormalizedPhone)
	}
	if input.Request.Email != nil {
		email = nullableString(*input.Request.Email)
	}
	if input.Request.BrandName != nil {
		brandName = nullableString(*input.Request.BrandName)
	}
	if input.Request.Province != nil {
		province = nullableString(*input.Request.Province)
	}
	if input.Request.City != nil {
		city = nullableString(*input.Request.City)
	}
	if input.Request.Address != nil {
		address = nullableString(*input.Request.Address)
	}
	if input.Request.District != nil {
		district = nullableString(*input.Request.District)
	}
	if input.Request.SubDistrict != nil {
		subDistrict = nullableString(*input.Request.SubDistrict)
	}

	_, err = q.ExecContext(ctx, `
		UPDATE owners
		SET code = ?, name = ?, phone = ?, email = ?, brand_name = ?, province = ?, city = ?, district = ?, sub_district = ?, address = ?
		WHERE id = ? AND deleted_at IS NULL`,
		code, name, phone, email, brandName, province, city, district, subDistrict, address, input.ID,
	)
	if err != nil {
		return Owner{}, mapDuplicateError(err)
	}
	return r.findOwnerByID(ctx, q, input.ID, false)
}

func (r *Repository) RestoreOwner(ctx context.Context, id int64) (Owner, error) {
	if _, err := r.findOwnerByID(ctx, r.db, id, true); err != nil {
		return Owner{}, err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE owners
		SET status = ?, deleted_at = NULL
		WHERE id = ?`, StatusActive, id)
	if err != nil {
		return Owner{}, err
	}
	return r.findOwnerByIDRaw(ctx, id)
}

func (r *Repository) SoftDeleteOwner(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE owners
		SET status = ?, deleted_at = COALESCE(deleted_at, ?)
		WHERE id = ? AND deleted_at IS NULL`, StatusDeleted, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	return ensureSingleDeleteResult(ctx, r.db, result, "owners", id)
}

func (r *Repository) ForceDeleteOwner(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM owners WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("force delete owner: %w", err)
	}
	return ensureAffected(result)
}

func (r *Repository) SoftDeleteOwners(ctx context.Context, ids []int64) (int64, error) {
	args := idArgs(ids, time.Now().UTC())
	result, err := r.db.ExecContext(ctx, `
		UPDATE owners
		SET status = ?, deleted_at = COALESCE(deleted_at, ?)
		WHERE id IN (`+placeholders(len(ids))+`) AND deleted_at IS NULL`, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) ForceDeleteOwners(ctx context.Context, ids []int64) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM owners WHERE id IN ("+placeholders(len(ids))+")", int64sToAny(ids)...)
	if err != nil {
		return 0, fmt.Errorf("bulk force delete owners: %w", err)
	}
	return result.RowsAffected()
}

func (r *Repository) CreateOutlet(ctx context.Context, ownerID int64, req CreateOutletRequest, normalizedPhone string) (Outlet, error) {
	if _, err := r.findOwnerByIDRaw(ctx, ownerID); err != nil {
		return Outlet{}, err
	}
	return r.createOutlet(ctx, r.db, ownerID, req, normalizedPhone)
}

func (r *Repository) CreateOutlets(ctx context.Context, ownerID int64, requests []CreateOutletRequest, normalizedPhones []string) ([]Outlet, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := r.findOwnerByID(ctx, tx, ownerID, false); err != nil {
		return nil, err
	}

	outlets := make([]Outlet, 0, len(requests))
	for index, req := range requests {
		outlet, err := r.createOutlet(ctx, tx, ownerID, req, normalizedPhones[index])
		if err != nil {
			return nil, err
		}
		outlets = append(outlets, outlet)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return outlets, nil
}

func (r *Repository) createOutlet(ctx context.Context, q queryExecutor, ownerID int64, req CreateOutletRequest, normalizedPhone string) (Outlet, error) {
	query := `
		INSERT INTO outlets (owner_id, code, name, phone, province, city, district, sub_district, address, status`
	args := []any{
		ownerID,
		trim(req.Code),
		trim(req.Name),
		nullableString(normalizedPhone),
		nullableString(req.Province),
		nullableString(req.City),
		nullableString(req.District),
		nullableString(req.SubDistrict),
		nullableString(req.Address),
	}
	values := `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE'`
	if req.CreatedAt != nil {
		query += `, created_at`
		values += `, ?`
		args = append(args, req.CreatedAt.UTC())
	}
	query += values + `)`
	result, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return Outlet{}, mapDuplicateError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Outlet{}, err
	}
	return r.findOutletByID(ctx, q, ownerID, id, false)
}

func (r *Repository) FindOutletByID(ctx context.Context, actor Actor, ownerID, outletID int64) (Outlet, error) {
	if _, err := r.FindOwnerByID(ctx, actor, ownerID); err != nil {
		return Outlet{}, err
	}
	return r.findOutletByID(ctx, r.db, ownerID, outletID, false)
}

func (r *Repository) findOutletByID(ctx context.Context, q queryExecutor, ownerID, outletID int64, includeDeleted bool) (Outlet, error) {
	deletedClause := "AND deleted_at IS NULL"
	if includeDeleted {
		deletedClause = ""
	}
	outlet, err := scanOutlet(q.QueryRowContext(ctx, `
		SELECT id, owner_id, code, name, phone, province, city, district, sub_district, address, status, created_at, updated_at
		FROM outlets
		WHERE id = ? AND owner_id = ? `+deletedClause, outletID, ownerID))
	if err == sql.ErrNoRows {
		return Outlet{}, ErrNotFound
	}
	return outlet, err
}

func (r *Repository) ListOutlets(ctx context.Context, actor Actor, ownerID int64, params ListParams) ([]Outlet, int64, error) {
	includeDeletedOwner := params.Scope != ScopeActive
	var err error
	if actor.RoleCode == RoleAdmin {
		_, err = r.findOwnerByID(ctx, r.db, ownerID, includeDeletedOwner)
	} else {
		_, err = r.findOwnerByIDVisible(ctx, r.db, actor, ownerID, includeDeletedOwner)
	}
	if err != nil {
		return nil, 0, err
	}
	where, args := outletWhere(ownerID, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM outlets ot WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := outletOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := `
		SELECT id, owner_id, code, name, phone, province, city, district, sub_district, address, status, created_at, updated_at
		FROM outlets ot
		WHERE ` + where + `
		ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		args = append(args, params.Limit, offset)
		query += `
		LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var outlets []Outlet
	for rows.Next() {
		outlet, err := scanOutlet(rows)
		if err != nil {
			return nil, 0, err
		}
		outlets = append(outlets, outlet)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return outlets, total, nil
}

func (r *Repository) ListGlobalOutlets(ctx context.Context, actor Actor, params ListParams) ([]OutletOverview, int64, error) {
	where, args := globalOutletWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM outlets ot
		LEFT JOIN owners o ON o.id = ot.owner_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := outletOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := outletOverviewSelect() + `
		WHERE ` + where + `
		ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		args = append(args, params.Limit, offset)
		query += `
		LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanOutletOverviews(rows, total)
}

func (r *Repository) FindGlobalOutletByID(ctx context.Context, actor Actor, outletID int64) (OutletOverview, error) {
	params := ListParams{Scope: ScopeActive}
	where, args := globalOutletWhere(actor, params)
	args = append([]any{outletID}, args...)
	item, err := scanOutletOverview(r.db.QueryRowContext(ctx, outletOverviewSelect()+`
		WHERE ot.id = ? AND `+where+`
		LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return OutletOverview{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) UpdateOutlet(ctx context.Context, ownerID, outletID int64, req UpdateOutletRequest, normalizedPhone *string) (Outlet, error) {
	return r.updateOutlet(ctx, r.db, ownerID, OutletUpdateInput{
		ID:              outletID,
		Request:         req,
		NormalizedPhone: normalizedPhone,
	})
}

func (r *Repository) UpdateOutlets(ctx context.Context, ownerID int64, updates []OutletUpdateInput) ([]Outlet, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	outlets := make([]Outlet, 0, len(updates))
	for _, update := range updates {
		outlet, err := r.updateOutlet(ctx, tx, ownerID, update)
		if err != nil {
			return nil, err
		}
		outlets = append(outlets, outlet)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return outlets, nil
}

func (r *Repository) updateOutlet(ctx context.Context, q queryExecutor, ownerID int64, input OutletUpdateInput) (Outlet, error) {
	current, err := r.findOutletByID(ctx, q, ownerID, input.ID, false)
	if err != nil {
		return Outlet{}, err
	}

	code := current.Code
	name := current.Name
	phone := current.Phone
	province := current.Province
	city := current.City
	district := current.District
	subDistrict := current.SubDistrict
	address := current.Address
	if input.Request.Code != nil {
		code = trim(*input.Request.Code)
	}
	if input.Request.Name != nil {
		name = trim(*input.Request.Name)
	}
	if input.NormalizedPhone != nil {
		phone = nullableString(*input.NormalizedPhone)
	}
	if input.Request.Province != nil {
		province = nullableString(*input.Request.Province)
	}
	if input.Request.City != nil {
		city = nullableString(*input.Request.City)
	}
	if input.Request.Address != nil {
		address = nullableString(*input.Request.Address)
	}
	if input.Request.District != nil {
		district = nullableString(*input.Request.District)
	}
	if input.Request.SubDistrict != nil {
		subDistrict = nullableString(*input.Request.SubDistrict)
	}

	_, err = q.ExecContext(ctx, `
		UPDATE outlets
		SET code = ?, name = ?, phone = ?, province = ?, city = ?, district = ?, sub_district = ?, address = ?
		WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
		code, name, phone, province, city, district, subDistrict, address, input.ID, ownerID,
	)
	if err != nil {
		return Outlet{}, mapDuplicateError(err)
	}
	return r.findOutletByID(ctx, q, ownerID, input.ID, false)
}

func (r *Repository) RestoreOutlet(ctx context.Context, ownerID, outletID int64) (Outlet, error) {
	if _, err := r.findOutletByID(ctx, r.db, ownerID, outletID, true); err != nil {
		return Outlet{}, err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE outlets
		SET status = ?, deleted_at = NULL
		WHERE id = ? AND owner_id = ?`, StatusActive, outletID, ownerID)
	if err != nil {
		return Outlet{}, err
	}
	return r.findOutletByID(ctx, r.db, ownerID, outletID, false)
}

func (r *Repository) SoftDeleteOutlet(ctx context.Context, ownerID, outletID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE outlets
		SET status = ?, deleted_at = COALESCE(deleted_at, ?)
		WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`, StatusDeleted, time.Now().UTC(), outletID, ownerID)
	if err != nil {
		return err
	}
	return ensureNestedDeleteResult(ctx, r.db, result, "outlets", outletID, "owner_id", ownerID)
}

func (r *Repository) ForceDeleteOutlet(ctx context.Context, ownerID, outletID int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM outlets WHERE id = ? AND owner_id = ?", outletID, ownerID)
	if err != nil {
		return fmt.Errorf("force delete outlet: %w", err)
	}
	return ensureAffected(result)
}

func (r *Repository) SoftDeleteOutlets(ctx context.Context, ownerID int64, ids []int64) (int64, error) {
	args := []any{StatusDeleted, time.Now().UTC(), ownerID}
	args = append(args, int64sToAny(ids)...)
	result, err := r.db.ExecContext(ctx, `
		UPDATE outlets
		SET status = ?, deleted_at = COALESCE(deleted_at, ?)
		WHERE owner_id = ? AND id IN (`+placeholders(len(ids))+`) AND deleted_at IS NULL`, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) ForceDeleteOutlets(ctx context.Context, ownerID int64, ids []int64) (int64, error) {
	args := []any{ownerID}
	args = append(args, int64sToAny(ids)...)
	result, err := r.db.ExecContext(ctx, "DELETE FROM outlets WHERE owner_id = ? AND id IN ("+placeholders(len(ids))+")", args...)
	if err != nil {
		return 0, fmt.Errorf("bulk force delete outlets: %w", err)
	}
	return result.RowsAffected()
}

func (r *Repository) createLeadForOwner(ctx context.Context, q queryExecutor, ownerID int64, actor Actor) error {
	now := time.Now().UTC()
	var outletID sql.NullInt64
	var firstOutletID int64
	if err := q.QueryRowContext(ctx, "SELECT id FROM outlets WHERE owner_id = ? AND deleted_at IS NULL ORDER BY id LIMIT 1", ownerID).Scan(&firstOutletID); err == nil {
		outletID = sql.NullInt64{Int64: firstOutletID, Valid: true}
	}

	result, err := q.ExecContext(ctx, `
		INSERT INTO customer_leads
			(code, owner_id, outlet_id, source_type, source_reference, stage, status, current_score, current_owner_user_id, current_owner_role)
		VALUES (?, ?, ?, 'MANUAL', ?, 'NEW', 'OPEN', 1, ?, ?)`,
		fmt.Sprintf("LEAD-%06d", ownerID),
		ownerID,
		outletID,
		fmt.Sprintf("owner:%d", ownerID),
		actor.ID,
		actor.RoleCode,
	)
	if err != nil {
		return mapDuplicateError(err)
	}
	leadID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `
		INSERT INTO lead_assignments
			(lead_id, owner_id, to_user_id, to_role, assigned_by_user_id, action, score, active, started_at)
		VALUES (?, ?, ?, ?, ?, 'CREATED_BY_ADMIN', 1, TRUE, ?)`,
		leadID,
		ownerID,
		actor.ID,
		actor.RoleCode,
		actor.ID,
		now,
	)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanOwner(row scanner) (Owner, error) {
	var owner Owner
	err := row.Scan(
		&owner.ID,
		&owner.Code,
		&owner.Name,
		&owner.Phone,
		&owner.Email,
		&owner.BrandName,
		&owner.Province,
		&owner.City,
		&owner.District,
		&owner.SubDistrict,
		&owner.Address,
		&owner.Status,
		&owner.OutletCount,
		&owner.SubscribedOutletCount,
		&owner.CreatedAt,
		&owner.UpdatedAt,
	)
	if err != nil {
		return Owner{}, err
	}
	owner.SubscriptionStatus = CalculateOwnerSubscriptionStatus(owner.CreatedAt, owner.SubscribedOutletCount)
	return owner, nil
}

func scanOutlet(row scanner) (Outlet, error) {
	var outlet Outlet
	err := row.Scan(
		&outlet.ID,
		&outlet.OwnerID,
		&outlet.Code,
		&outlet.Name,
		&outlet.Phone,
		&outlet.Province,
		&outlet.City,
		&outlet.District,
		&outlet.SubDistrict,
		&outlet.Address,
		&outlet.Status,
		&outlet.CreatedAt,
		&outlet.UpdatedAt,
	)
	return outlet, err
}

func scanOutletOverviews(rows *sql.Rows, total int64) ([]OutletOverview, int64, error) {
	items := []OutletOverview{}
	for rows.Next() {
		item, err := scanOutletOverview(rows)
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

func scanOutletOverview(row scanner) (OutletOverview, error) {
	var item OutletOverview
	err := row.Scan(
		&item.ID,
		&item.OwnerID,
		&item.OwnerCode,
		&item.OwnerName,
		&item.OwnerPhone,
		&item.OwnerEmail,
		&item.OwnerBrandName,
		&item.Code,
		&item.Name,
		&item.Phone,
		&item.Province,
		&item.City,
		&item.District,
		&item.SubDistrict,
		&item.Address,
		&item.Status,
		&item.SubscriptionCount,
		&item.ActiveSubscriptionCount,
		&item.LatestSubscriptionStatus,
		&item.LatestSubscriptionStart,
		&item.LatestSubscriptionUntil,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

// ownerAgeThreshold is the NEW/OLD cutoff for OwnerOverview.AgeStatus. 90 days chosen provisionally
// (flagged to the user as unconfirmed in the Sprint 15a plan) — distinct from the 60-day EXPIRED
// threshold in outlet_subscription_status.go, which answers a different question (subscription
// lapse, not account age).
const ownerAgeThresholdDays = 90

func scanOwnerOverview(row scanner) (OwnerOverview, error) {
	var item OwnerOverview
	err := row.Scan(
		&item.ID,
		&item.Code,
		&item.Name,
		&item.Phone,
		&item.Email,
		&item.BrandName,
		&item.Province,
		&item.City,
		&item.District,
		&item.SubDistrict,
		&item.Address,
		&item.Status,
		&item.AccountCode,
		&item.WalletID,
		&item.WalletBalance,
		&item.WalletLedgerBalance,
		&item.WalletStatus,
		&item.WalletCreatedAt,
		&item.WalletUpdatedAt,
		&item.TotalTransferred,
		&item.TotalTopup,
		&item.TotalSpent,
		&item.OutletCount,
		&item.SubscribedOutletCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return OwnerOverview{}, err
	}
	if time.Since(item.CreatedAt) > ownerAgeThresholdDays*24*time.Hour {
		item.AgeStatus = OwnerAgeOld
	} else {
		item.AgeStatus = OwnerAgeNew
	}
	item.SubscriptionStatus = CalculateOwnerSubscriptionStatus(item.CreatedAt, item.SubscribedOutletCount)
	return item, nil
}

// GetOwnerOverview returns the Owner-level rollup (wallet + transfer/topup/spent totals + age and
// subscription status). actor visibility rules mirror ListOwners (ownerVisibilityWhere).
func (r *Repository) GetOwnerOverview(ctx context.Context, actor Actor, ownerID int64) (OwnerOverview, error) {
	visibility, args := ownerVisibilityWhere(actor)
	args = append([]any{ownerID}, args...)
	item, err := scanOwnerOverview(r.db.QueryRowContext(ctx, ownerOverviewSelect()+`
		WHERE o.id = ? AND o.deleted_at IS NULL AND `+visibility+`
		LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return OwnerOverview{}, ErrNotFound
	}
	return item, err
}

func ownerWhere(actor Actor, params ListParams) (string, []any) {
	where := []string{scopeCondition("o.deleted_at", params.Scope)}
	var args []any
	visibility, visibilityArgs := ownerVisibilityWhere(actor)
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(o.code LIKE ? OR o.name LIKE ? OR o.phone LIKE ? OR o.brand_name LIKE ? OR o.city LIKE ? OR o.province LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern, pattern, pattern)
	}
	if params.Code != "" {
		where = append(where, "o.code LIKE ?")
		args = append(args, like(params.Code))
	}
	if params.Name != "" {
		where = append(where, "o.name LIKE ?")
		args = append(args, like(params.Name))
	}
	if params.Phone != "" {
		where = append(where, "o.phone LIKE ?")
		args = append(args, like(params.Phone))
	}
	if params.BrandName != "" {
		where = append(where, "o.brand_name LIKE ?")
		args = append(args, like(params.BrandName))
	}
	if params.Province != "" {
		where = append(where, "o.province LIKE ?")
		args = append(args, like(params.Province))
	}
	if params.City != "" {
		where = append(where, "o.city LIKE ?")
		args = append(args, like(params.City))
	}
	if params.CreatedFrom != nil {
		where = append(where, "o.created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "o.created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	return strings.Join(where, " AND "), args
}

func ownerVisibilityWhere(actor Actor) (string, []any) {
	return ownerVisibilityWhereByColumn(actor, "o.id")
}

func ownerVisibilityWhereByColumn(actor Actor, ownerColumn string) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "1 = 1", nil
	case RoleSupervisor:
		return `EXISTS (
			SELECT 1
			FROM customer_leads cl
			WHERE cl.owner_id = ` + ownerColumn + `
				AND cl.deleted_at IS NULL
				AND (cl.current_owner_user_id = ? OR cl.supervisor_id = ?)
		)`, []any{actor.ID, actor.ID}
	case RoleSales:
		return `EXISTS (
			SELECT 1
			FROM customer_leads cl
			WHERE cl.owner_id = ` + ownerColumn + `
				AND cl.deleted_at IS NULL
				AND cl.current_owner_role = 'SALES'
				AND cl.current_owner_user_id = ?
		)`, []any{actor.ID}
	default:
		return "1 = 0", nil
	}
}

func outletWhere(ownerID int64, params ListParams) (string, []any) {
	where := []string{"ot.owner_id = ?", scopeCondition("ot.deleted_at", params.Scope)}
	args := []any{ownerID}
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(ot.code LIKE ? OR ot.name LIKE ? OR ot.phone LIKE ? OR ot.city LIKE ? OR ot.province LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}
	if params.Code != "" {
		where = append(where, "ot.code LIKE ?")
		args = append(args, like(params.Code))
	}
	if params.Name != "" {
		where = append(where, "ot.name LIKE ?")
		args = append(args, like(params.Name))
	}
	if params.Phone != "" {
		where = append(where, "ot.phone LIKE ?")
		args = append(args, like(params.Phone))
	}
	if params.Province != "" {
		where = append(where, "ot.province LIKE ?")
		args = append(args, like(params.Province))
	}
	if params.City != "" {
		where = append(where, "ot.city LIKE ?")
		args = append(args, like(params.City))
	}
	if params.CreatedFrom != nil {
		where = append(where, "ot.created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "ot.created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	return strings.Join(where, " AND "), args
}

func globalOutletWhere(actor Actor, params ListParams) (string, []any) {
	where := []string{scopeCondition("ot.deleted_at", params.Scope)}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhereByColumn(actor, "ot.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(ot.code LIKE ? OR ot.name LIKE ? OR ot.phone LIKE ? OR ot.city LIKE ? OR ot.province LIKE ? OR o.code LIKE ? OR o.name LIKE ? OR o.brand_name LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern)
	}
	if params.OwnerID != nil {
		where = append(where, "ot.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.Code != "" {
		where = append(where, "ot.code LIKE ?")
		args = append(args, like(params.Code))
	}
	if params.Name != "" {
		where = append(where, "ot.name LIKE ?")
		args = append(args, like(params.Name))
	}
	if params.Phone != "" {
		where = append(where, "ot.phone LIKE ?")
		args = append(args, like(params.Phone))
	}
	if params.BrandName != "" {
		where = append(where, "o.brand_name LIKE ?")
		args = append(args, like(params.BrandName))
	}
	if params.Province != "" {
		where = append(where, "ot.province LIKE ?")
		args = append(args, like(params.Province))
	}
	if params.City != "" {
		where = append(where, "ot.city LIKE ?")
		args = append(args, like(params.City))
	}
	if params.CreatedFrom != nil {
		where = append(where, "ot.created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "ot.created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	if params.SubscriptionStatus != "" || params.SubscriptionMonth != "" {
		subQuery := []string{"s.outlet_id = ot.id", "s.deleted_at IS NULL"}
		subArgs := []any{}
		if params.SubscriptionStatus != "" {
			subQuery = append(subQuery, "s.status = ?")
			subArgs = append(subArgs, params.SubscriptionStatus)
		}
		if params.SubscriptionMonth != "" {
			subQuery = append(subQuery, "STR_TO_DATE(CONCAT(?, '-01'), '%Y-%m-%d') <= s.active_until")
			subQuery = append(subQuery, "LAST_DAY(STR_TO_DATE(CONCAT(?, '-01'), '%Y-%m-%d')) >= s.active_from")
			subArgs = append(subArgs, params.SubscriptionMonth, params.SubscriptionMonth)
		}
		where = append(where, "EXISTS (SELECT 1 FROM subscriptions s WHERE "+strings.Join(subQuery, " AND ")+")")
		args = append(args, subArgs...)
	}
	return strings.Join(where, " AND "), args
}

func scopeCondition(column, scope string) string {
	switch scope {
	case ScopeDeleted:
		return column + " IS NOT NULL"
	case ScopeAll:
		return "1 = 1"
	default:
		return column + " IS NULL"
	}
}

func outletOverviewSelect() string {
	return `
		SELECT
			ot.id,
			ot.owner_id,
			o.code,
			o.name,
			o.phone,
			o.email,
			o.brand_name,
			ot.code,
			ot.name,
			ot.phone,
			ot.province,
			ot.city,
			ot.district,
			ot.sub_district,
			ot.address,
			ot.status,
			(
				SELECT COUNT(*)
				FROM subscriptions s
				WHERE s.outlet_id = ot.id AND s.deleted_at IS NULL
			) AS subscription_count,
			(
				SELECT COUNT(*)
				FROM subscriptions s
				WHERE s.outlet_id = ot.id AND s.deleted_at IS NULL AND s.status = 'ACTIVE'
			) AS active_subscription_count,
			(
				SELECT s.status
				FROM subscriptions s
				WHERE s.outlet_id = ot.id AND s.deleted_at IS NULL
				ORDER BY s.active_until DESC, s.id DESC
				LIMIT 1
			) AS latest_subscription_status,
			(
				SELECT s.active_from
				FROM subscriptions s
				WHERE s.outlet_id = ot.id AND s.deleted_at IS NULL
				ORDER BY s.active_until DESC, s.id DESC
				LIMIT 1
			) AS latest_subscription_start,
			(
				SELECT s.active_until
				FROM subscriptions s
				WHERE s.outlet_id = ot.id AND s.deleted_at IS NULL
				ORDER BY s.active_until DESC, s.id DESC
				LIMIT 1
			) AS latest_subscription_until,
			ot.created_at,
			ot.updated_at
		FROM outlets ot
		LEFT JOIN owners o ON o.id = ot.owner_id`
}

// ownerOverviewSelect is the Owner-level analog of outletOverviewSelect: wallet balance and its
// rollups (transferred/topup/spent) live here now — they used to be duplicated per outlet via a
// JOIN on wa.owner_id = ot.owner_id, which showed the identical number on every outlet an owner
// had. That JOIN is gone from outletOverviewSelect; this is the only place wallet data is surfaced.
//
// total_topup filters on wp.status = 'ACCEPTED' (Sprint 15a §2's top-up lifecycle: a top-up only
// credits the wallet, and only counts toward this rollup, once ACCEPTED — PENDING/REJECTED/EXPIRED
// top-ups never touched balance in the first place).
func ownerOverviewSelect() string {
	return `
		SELECT
			o.id,
			o.code,
			o.name,
			o.phone,
			o.email,
			o.brand_name,
			o.province,
			o.city,
			o.district,
			o.sub_district,
			o.address,
			o.status,
			COALESCE(wa.account_code, CONCAT('WALLET-OWNER-', LPAD(o.id, 6, '0'))) AS account_code,
			COALESCE(wa.id, 0) AS wallet_id,
			COALESCE(CAST(wa.balance AS CHAR), '0.00') AS wallet_balance,
			COALESCE(CAST((
				SELECT COALESCE(SUM(CASE WHEN wt.direction = 'CREDIT' THEN wt.amount ELSE -wt.amount END), 0)
				FROM wallet_transactions wt
				WHERE wt.wallet_account_id = wa.id AND wt.deleted_at IS NULL
			) AS CHAR), '0.00') AS wallet_ledger_balance,
			COALESCE(wa.status, 'UNAVAILABLE') AS wallet_status,
			COALESCE(wa.created_at, o.created_at) AS wallet_created_at,
			COALESCE(wa.updated_at, o.updated_at) AS wallet_updated_at,
			COALESCE(CAST((
				SELECT COALESCE(SUM(ot2.amount), 0)
				FROM owner_transfers ot2
				WHERE ot2.owner_id = o.id AND ot2.match_status = 'MATCHED'
			) AS CHAR), '0.00') AS total_transferred,
			COALESCE(CAST((
				SELECT COALESCE(SUM(wp.amount), 0)
				FROM wallet_payments wp
				WHERE wp.owner_id = o.id AND wp.payment_type = 'TOPUP' AND wp.status = 'ACCEPTED'
			) AS CHAR), '0.00') AS total_topup,
			COALESCE(CAST((
				SELECT COALESCE(SUM(wt.amount), 0)
				FROM wallet_transactions wt
				WHERE wt.wallet_account_id = wa.id AND wt.direction = 'DEBIT' AND wt.deleted_at IS NULL
			) AS CHAR), '0.00') AS total_spent,
			(
				SELECT COUNT(*)
				FROM outlets ot3
				WHERE ot3.owner_id = o.id AND ot3.deleted_at IS NULL
			) AS outlet_count,
			(
				SELECT COUNT(DISTINCT s.outlet_id)
				FROM subscriptions s
				JOIN outlets ot4 ON ot4.id = s.outlet_id
				WHERE ot4.owner_id = o.id AND s.deleted_at IS NULL AND s.status = 'ACTIVE'
			) AS subscribed_outlet_count,
			o.created_at,
			o.updated_at
		FROM owners o
		LEFT JOIN wallet_accounts wa ON wa.owner_id = o.id AND wa.deleted_at IS NULL`
}

func ownerOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"created_at":   "o.created_at",
		"updated_at":   "o.updated_at",
		"code":         "o.code",
		"name":         "o.name",
		"brand_name":   "o.brand_name",
		"city":         "o.city",
		"province":     "o.province",
		"phone":        "o.phone",
		"status":       "CASE WHEN (SELECT COUNT(DISTINCT s.outlet_id) FROM subscriptions s JOIN outlets ot4 ON ot4.id = s.outlet_id WHERE ot4.owner_id = o.id AND s.deleted_at IS NULL) > 0 THEN 1 WHEN o.created_at >= DATE_SUB(NOW(), INTERVAL 14 DAY) THEN 2 ELSE 3 END",
		"outlet_count": "outlet_count",
		"wallet_balance": "CAST(wa.balance AS DECIMAL(20,2))",
	}, "o.created_at DESC, o.id DESC")
}

func outletOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"created_at": "ot.created_at",
		"updated_at": "ot.updated_at",
		"code":       "ot.code",
		"name":       "ot.name",
		"city":       "ot.city",
		"province":   "ot.province",
	}, "ot.created_at DESC, ot.id DESC")
}

func orderBy(sort string, allowed map[string]string, fallback string) (string, error) {
	sort = trim(sort)
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

func nullableString(value string) sql.NullString {
	value = trim(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func trim(value string) string {
	return strings.TrimSpace(value)
}

func like(value string) string {
	return "%" + trim(value) + "%"
}

func placeholders(count int) string {
	if count < 1 {
		return ""
	}
	items := make([]string, count)
	for index := range items {
		items[index] = "?"
	}
	return strings.Join(items, ",")
}

func int64sToAny(values []int64) []any {
	args := make([]any, len(values))
	for index, value := range values {
		args[index] = value
	}
	return args
}

func idArgs(ids []int64, deletedAt time.Time) []any {
	args := []any{StatusDeleted, deletedAt}
	args = append(args, int64sToAny(ids)...)
	return args
}

func ensureAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func ensureSingleDeleteResult(ctx context.Context, q queryExecutor, result sql.Result, table string, id int64) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return ensureRowExists(ctx, q, table, "id = ?", id)
}

func ensureNestedDeleteResult(ctx context.Context, q queryExecutor, result sql.Result, table string, id int64, parentColumn string, parentID int64) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return ensureRowExists(ctx, q, table, "id = ? AND "+parentColumn+" = ?", id, parentID)
}

func ensureRowExists(ctx context.Context, q queryExecutor, table, where string, args ...any) error {
	var exists int
	err := q.QueryRowContext(ctx, "SELECT 1 FROM "+table+" WHERE "+where+" LIMIT 1", args...).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

func mapDuplicateError(err error) error {
	message := err.Error()
	if strings.Contains(message, "Duplicate entry") || strings.Contains(message, "uq_owners_code") || strings.Contains(message, "uq_outlets_code") {
		return ErrCodeAlreadyUsed
	}
	return fmt.Errorf("database customer: %w", err)
}
