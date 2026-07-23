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

func (r *Repository) CreateOwner(ctx context.Context, req CreateOwnerRequest, normalizedPhone string) (Owner, error) {
	return r.createOwner(ctx, r.db, req, normalizedPhone)
}

func (r *Repository) CreateOwners(ctx context.Context, requests []CreateOwnerRequest, normalizedPhones []string) ([]Owner, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	owners := make([]Owner, 0, len(requests))
	for index, req := range requests {
		owner, err := r.createOwner(ctx, tx, req, normalizedPhones[index])
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

func (r *Repository) createOwner(ctx context.Context, q queryExecutor, req CreateOwnerRequest, normalizedPhone string) (Owner, error) {
	result, err := q.ExecContext(ctx, `
		INSERT INTO owners (code, name, phone, email, brand_name, province, city, address, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE')`,
		trim(req.Code),
		trim(req.Name),
		nullableString(normalizedPhone),
		nullableString(req.Email),
		nullableString(req.BrandName),
		nullableString(req.Province),
		nullableString(req.City),
		nullableString(req.Address),
	)
	if err != nil {
		return Owner{}, mapDuplicateError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Owner{}, err
	}
	return r.findOwnerByID(ctx, q, id, false)
}

func (r *Repository) FindOwnerByID(ctx context.Context, id int64) (Owner, error) {
	return r.findOwnerByID(ctx, r.db, id, false)
}

func (r *Repository) findOwnerByID(ctx context.Context, q queryExecutor, id int64, includeDeleted bool) (Owner, error) {
	deletedClause := "AND o.deleted_at IS NULL"
	if includeDeleted {
		deletedClause = ""
	}
	owner, err := scanOwner(q.QueryRowContext(ctx, `
		SELECT
			o.id, o.code, o.name, o.phone, o.email, o.brand_name, o.province, o.city,
			o.address, o.status, COUNT(ot.id) AS outlet_count, o.created_at, o.updated_at
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		WHERE o.id = ? `+deletedClause+`
		GROUP BY o.id`, id))
	if err == sql.ErrNoRows {
		return Owner{}, ErrNotFound
	}
	return owner, err
}

func (r *Repository) ListOwners(ctx context.Context, params ListParams) ([]Owner, int64, error) {
	where, args := ownerWhere(params)
	countQuery := "SELECT COUNT(*) FROM owners o WHERE " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := ownerOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			o.id, o.code, o.name, o.phone, o.email, o.brand_name, o.province, o.city,
			o.address, o.status, COUNT(ot.id) AS outlet_count, o.created_at, o.updated_at
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		WHERE `+where+`
		GROUP BY o.id
		ORDER BY `+orderBy+`
		LIMIT ? OFFSET ?`, args...)
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

	_, err = q.ExecContext(ctx, `
		UPDATE owners
		SET code = ?, name = ?, phone = ?, email = ?, brand_name = ?, province = ?, city = ?, address = ?
		WHERE id = ? AND deleted_at IS NULL`,
		code, name, phone, email, brandName, province, city, address, input.ID,
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
	return r.FindOwnerByID(ctx, id)
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
	if _, err := r.FindOwnerByID(ctx, ownerID); err != nil {
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
	result, err := q.ExecContext(ctx, `
		INSERT INTO outlets (owner_id, code, name, phone, province, city, address, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE')`,
		ownerID,
		trim(req.Code),
		trim(req.Name),
		nullableString(normalizedPhone),
		nullableString(req.Province),
		nullableString(req.City),
		nullableString(req.Address),
	)
	if err != nil {
		return Outlet{}, mapDuplicateError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Outlet{}, err
	}
	return r.findOutletByID(ctx, q, ownerID, id, false)
}

func (r *Repository) FindOutletByID(ctx context.Context, ownerID, outletID int64) (Outlet, error) {
	return r.findOutletByID(ctx, r.db, ownerID, outletID, false)
}

func (r *Repository) findOutletByID(ctx context.Context, q queryExecutor, ownerID, outletID int64, includeDeleted bool) (Outlet, error) {
	deletedClause := "AND deleted_at IS NULL"
	if includeDeleted {
		deletedClause = ""
	}
	outlet, err := scanOutlet(q.QueryRowContext(ctx, `
		SELECT id, owner_id, code, name, phone, province, city, address, status, created_at, updated_at
		FROM outlets
		WHERE id = ? AND owner_id = ? `+deletedClause, outletID, ownerID))
	if err == sql.ErrNoRows {
		return Outlet{}, ErrNotFound
	}
	return outlet, err
}

func (r *Repository) ListOutlets(ctx context.Context, ownerID int64, params ListParams) ([]Outlet, int64, error) {
	if _, err := r.FindOwnerByID(ctx, ownerID); err != nil {
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
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, owner_id, code, name, phone, province, city, address, status, created_at, updated_at
		FROM outlets ot
		WHERE `+where+`
		ORDER BY `+orderBy+`
		LIMIT ? OFFSET ?`, args...)
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

	_, err = q.ExecContext(ctx, `
		UPDATE outlets
		SET code = ?, name = ?, phone = ?, province = ?, city = ?, address = ?
		WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
		code, name, phone, province, city, address, input.ID, ownerID,
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
	return r.FindOutletByID(ctx, ownerID, outletID)
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
		&owner.Address,
		&owner.Status,
		&owner.OutletCount,
		&owner.CreatedAt,
		&owner.UpdatedAt,
	)
	return owner, err
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
		&outlet.Address,
		&outlet.Status,
		&outlet.CreatedAt,
		&outlet.UpdatedAt,
	)
	return outlet, err
}

func ownerWhere(params ListParams) (string, []any) {
	where := []string{"o.deleted_at IS NULL"}
	var args []any
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
	return strings.Join(where, " AND "), args
}

func outletWhere(ownerID int64, params ListParams) (string, []any) {
	where := []string{"ot.owner_id = ?", "ot.deleted_at IS NULL"}
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
	return strings.Join(where, " AND "), args
}

func ownerOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"created_at": "o.created_at",
		"updated_at": "o.updated_at",
		"code":       "o.code",
		"name":       "o.name",
		"brand_name": "o.brand_name",
		"city":       "o.city",
		"province":   "o.province",
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
