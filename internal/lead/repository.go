package lead

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type Repository struct {
	db *sql.DB
}

type lockedLead struct {
	ID                 int64
	OwnerID            sql.NullInt64
	CurrentOwnerUserID sql.NullInt64
	CurrentOwnerRole   string
	SupervisorID       sql.NullInt64
	ActiveSalesID      sql.NullInt64
	Stage              string
	Status             string
	CurrentScore       sql.NullInt64
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateLead(ctx context.Context, actor identity.User, req CreateLeadRequest) (Lead, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Lead{}, err
	}
	defer tx.Rollback()

	if err := ensureOwnerExists(ctx, tx, req.OwnerID); err != nil {
		return Lead{}, err
	}
	if req.OutletID != nil {
		if err := ensureOutletExists(ctx, tx, req.OwnerID, *req.OutletID); err != nil {
			return Lead{}, err
		}
	}

	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = "MANUAL"
	}
	query := `
		INSERT INTO customer_leads
			(code, owner_id, outlet_id, source_type, source_reference, stage, status, current_score, current_owner_user_id, current_owner_role`
	args := []any{
		fmt.Sprintf("LEAD-%06d", req.OwnerID),
		req.OwnerID,
		nullableInt64Ptr(req.OutletID),
		sourceType,
		nullableString(req.SourceReference),
		actor.ID,
	}
	values := `) VALUES (?, ?, ?, ?, ?, 'NEW', 'OPEN', 1, ?, 'ADMIN'`
	if req.CreatedAt != nil {
		query += `, created_at`
		values += `, ?`
		args = append(args, req.CreatedAt.UTC())
	}
	query += values + `)`
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return Lead{}, mapCreateLeadError(err)
	}
	leadID, err := result.LastInsertId()
	if err != nil {
		return Lead{}, err
	}
	if err := r.insertAssignment(ctx, tx, assignmentInput{
		LeadID:           leadID,
		OwnerID:          sql.NullInt64{Int64: req.OwnerID, Valid: true},
		ToUserID:         sql.NullInt64{Int64: actor.ID, Valid: true},
		ToRole:           RoleAdmin,
		AssignedByUserID: sql.NullInt64{Int64: actor.ID, Valid: true},
		Action:           ActionCreatedExistingOwner,
		Score:            sql.NullInt64{Int64: 1, Valid: true},
		Active:           true,
		StartedAt:        time.Now().UTC(),
	}); err != nil {
		return Lead{}, err
	}
	if err := tx.Commit(); err != nil {
		return Lead{}, err
	}
	return r.FindLeadByID(ctx, actor, leadID)
}

func (r *Repository) ListLeads(ctx context.Context, actor identity.User, params ListParams) ([]Lead, int64, error) {
	where, args := leadWhere(actor, params)

	countArgs := make([]any, len(args))
	copy(countArgs, args)

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM outlets ot
		JOIN owners o ON o.id = ot.owner_id AND o.deleted_at IS NULL
		LEFT JOIN customer_leads cl ON (cl.outlet_id = ot.id OR (cl.owner_id = ot.owner_id AND cl.outlet_id IS NULL)) AND cl.deleted_at IS NULL
		WHERE `+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := leadOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}

	selectArgs := make([]any, len(args))
	copy(selectArgs, args)

	query := leadSelect() + `
		WHERE ` + where + `
		ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		selectArgs = append(selectArgs, params.Limit, offset)
		query += `
		LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanLeads(rows, total)
}

func (r *Repository) FindLeadByID(ctx context.Context, actor identity.User, id int64) (Lead, error) {
	where, args := leadWhere(actor, ListParams{})
	args = append([]any{id}, args...)
	lead, err := scanLead(r.db.QueryRowContext(ctx, leadSelect()+`
		WHERE cl.id = ? AND `+where+`
		LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return Lead{}, ErrNotFound
	}
	return lead, err
}

func (r *Repository) findLeadByIDRaw(ctx context.Context, id int64) (Lead, error) {
	lead, err := scanLead(r.db.QueryRowContext(ctx, leadSelect()+`
		WHERE cl.id = ? AND cl.deleted_at IS NULL
		LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return Lead{}, ErrNotFound
	}
	return lead, err
}

func (r *Repository) AssignmentHistory(ctx context.Context, actor identity.User, leadID int64) ([]AssignmentHistory, error) {
	if _, err := r.FindLeadByID(ctx, actor, leadID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			la.id, la.lead_id, la.owner_id,
			la.from_user_id, fu.name, la.from_role,
			la.to_user_id, tu.name, la.to_role,
			la.supervisor_id, su.name,
			la.assigned_by_user_id, au.name,
			la.action, la.reason, la.score, la.active, la.started_at, la.ended_at, la.created_at
		FROM lead_assignments la
		LEFT JOIN users fu ON fu.id = la.from_user_id
		LEFT JOIN users tu ON tu.id = la.to_user_id
		LEFT JOIN users su ON su.id = la.supervisor_id
		LEFT JOIN users au ON au.id = la.assigned_by_user_id
		WHERE la.lead_id = ? AND la.deleted_at IS NULL
		ORDER BY la.started_at DESC, la.id DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []AssignmentHistory{}
	for rows.Next() {
		item, err := scanAssignmentHistory(rows)
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

func (r *Repository) AssignSupervisor(ctx context.Context, actor identity.User, leadIDs []int64, supervisorID int64, reason string) ([]Lead, error) {
	if err := r.ensureActiveUserRole(ctx, supervisorID, RoleSupervisor); err != nil {
		return nil, err
	}
	return r.applyBulk(ctx, actor, leadIDs, func(ctx context.Context, tx *sql.Tx, current lockedLead) error {
		score := current.CurrentScore
		if !score.Valid {
			score = sql.NullInt64{Int64: 1, Valid: true}
		}
		return r.moveLead(ctx, tx, current, moveInput{
			ActorID:       actor.ID,
			ToUserID:      supervisorID,
			ToRole:        RoleSupervisor,
			SupervisorID:  sql.NullInt64{Int64: supervisorID, Valid: true},
			ActiveSalesID: sql.NullInt64{},
			Action:        ActionAssignedSupervisor,
			Reason:        reason,
			Stage:         current.Stage,
			Status:        StatusOpen,
			Score:         score,
		})
	})
}

func (r *Repository) AssignSales(ctx context.Context, actor identity.User, leadIDs []int64, salesID int64, reason string) ([]Lead, error) {
	if err := r.ensureActiveUserRole(ctx, salesID, RoleSales); err != nil {
		return nil, err
	}
	return r.applyBulk(ctx, actor, leadIDs, func(ctx context.Context, tx *sql.Tx, current lockedLead) error {
		if actor.RoleCode != RoleSupervisor {
			return ErrForbidden
		}
		if !currentBelongsToSupervisor(current, actor.ID) {
			return ErrForbidden
		}
		score := current.CurrentScore
		stage := current.Stage
		if !score.Valid || score.Int64 == 0 {
			score = sql.NullInt64{Int64: 1, Valid: true}
			stage = StagePossible
		}
		action := ActionAssignedSales
		if current.CurrentOwnerRole == RoleSales {
			action = ActionRedistributedSales
		}
		return r.moveLead(ctx, tx, current, moveInput{
			ActorID:       actor.ID,
			ToUserID:      salesID,
			ToRole:        RoleSales,
			SupervisorID:  sql.NullInt64{Int64: actor.ID, Valid: true},
			ActiveSalesID: sql.NullInt64{Int64: salesID, Valid: true},
			Action:        action,
			Reason:        reason,
			Stage:         stage,
			Status:        StatusOpen,
			Score:         score,
		})
	})
}

func (r *Repository) ReleaseToAdmin(ctx context.Context, actor identity.User, leadIDs []int64, adminID sql.NullInt64, reason string) ([]Lead, error) {
	targetAdminID := adminID.Int64
	if !adminID.Valid {
		if actor.RoleCode == RoleAdmin {
			targetAdminID = actor.ID
		} else {
			id, err := r.firstActiveAdminID(ctx)
			if err != nil {
				return nil, err
			}
			targetAdminID = id
		}
	}
	if err := r.ensureActiveUserRole(ctx, targetAdminID, RoleAdmin); err != nil {
		return nil, err
	}
	return r.applyBulk(ctx, actor, leadIDs, func(ctx context.Context, tx *sql.Tx, current lockedLead) error {
		if actor.RoleCode == RoleSupervisor && !currentBelongsToSupervisor(current, actor.ID) {
			return ErrForbidden
		}
		score := current.CurrentScore
		if !score.Valid {
			score = sql.NullInt64{Int64: 1, Valid: true}
		}
		return r.moveLead(ctx, tx, current, moveInput{
			ActorID:       actor.ID,
			ToUserID:      targetAdminID,
			ToRole:        RoleAdmin,
			SupervisorID:  sql.NullInt64{},
			ActiveSalesID: sql.NullInt64{},
			Action:        ActionReleasedAdmin,
			Reason:        reason,
			Stage:         StageNew,
			Status:        StatusOpen,
			Score:         score,
		})
	})
}

func (r *Repository) MarkInvalid(ctx context.Context, actor identity.User, leadID int64, reason string) (Lead, error) {
	items, err := r.applyBulk(ctx, actor, []int64{leadID}, func(ctx context.Context, tx *sql.Tx, current lockedLead) error {
		if actor.RoleCode != RoleSales ||
			current.CurrentOwnerRole != RoleSales ||
			!current.CurrentOwnerUserID.Valid ||
			current.CurrentOwnerUserID.Int64 != actor.ID ||
			!current.SupervisorID.Valid {
			return ErrForbidden
		}
		return r.moveLead(ctx, tx, current, moveInput{
			ActorID:              actor.ID,
			ToUserID:             current.SupervisorID.Int64,
			ToRole:               RoleSupervisor,
			SupervisorID:         current.SupervisorID,
			ActiveSalesID:        sql.NullInt64{},
			Action:               ActionInvalidatedBySales,
			Reason:               reason,
			Stage:                StageInvalid,
			Status:               StatusInvalid,
			Score:                sql.NullInt64{Int64: 0, Valid: true},
			InvalidatedAt:        sql.NullTime{Time: time.Now().UTC(), Valid: true},
			InvalidatedBySalesID: sql.NullInt64{Int64: actor.ID, Valid: true},
		})
	})
	if err != nil {
		return Lead{}, err
	}
	if len(items) == 0 {
		return Lead{}, ErrNotFound
	}
	return items[0], nil
}

type bulkMutation func(context.Context, *sql.Tx, lockedLead) error

func (r *Repository) applyBulk(ctx context.Context, actor identity.User, leadIDs []int64, mutate bulkMutation) ([]Lead, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for _, id := range leadIDs {
		current, err := r.lockLead(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if !canSeeLockedLead(actor, current) {
			return nil, ErrForbidden
		}
		if err := mutate(ctx, tx, current); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	items := make([]Lead, 0, len(leadIDs))
	for _, id := range leadIDs {
		item, err := r.findLeadByIDRaw(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

type moveInput struct {
	ActorID              int64
	ToUserID             int64
	ToRole               string
	SupervisorID         sql.NullInt64
	ActiveSalesID        sql.NullInt64
	Action               string
	Reason               string
	Stage                string
	Status               string
	Score                sql.NullInt64
	InvalidatedAt        sql.NullTime
	InvalidatedBySalesID sql.NullInt64
}

func (r *Repository) moveLead(ctx context.Context, tx *sql.Tx, current lockedLead, input moveInput) error {
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		UPDATE lead_assignments
		SET active = FALSE, ended_at = ?
		WHERE lead_id = ? AND active = TRUE`, now, current.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE customer_leads
		SET
			current_owner_user_id = ?,
			current_owner_role = ?,
			supervisor_id = ?,
			active_sales_id = ?,
			stage = ?,
			status = ?,
			current_score = ?,
			invalidated_at = ?,
			invalidated_by_sales_id = ?
		WHERE id = ?`,
		input.ToUserID,
		input.ToRole,
		input.SupervisorID,
		input.ActiveSalesID,
		input.Stage,
		input.Status,
		input.Score,
		input.InvalidatedAt,
		input.InvalidatedBySalesID,
		current.ID,
	); err != nil {
		return err
	}
	return r.insertAssignment(ctx, tx, assignmentInput{
		LeadID:           current.ID,
		OwnerID:          current.OwnerID,
		FromUserID:       current.CurrentOwnerUserID,
		FromRole:         sql.NullString{String: current.CurrentOwnerRole, Valid: current.CurrentOwnerRole != ""},
		ToUserID:         sql.NullInt64{Int64: input.ToUserID, Valid: true},
		ToRole:           input.ToRole,
		SupervisorID:     input.SupervisorID,
		AssignedByUserID: sql.NullInt64{Int64: input.ActorID, Valid: true},
		Action:           input.Action,
		Reason:           nullableString(input.Reason),
		Score:            input.Score,
		Active:           true,
		StartedAt:        now,
	})
}

type assignmentInput struct {
	LeadID           int64
	OwnerID          sql.NullInt64
	FromUserID       sql.NullInt64
	FromRole         sql.NullString
	ToUserID         sql.NullInt64
	ToRole           string
	SupervisorID     sql.NullInt64
	AssignedByUserID sql.NullInt64
	Action           string
	Reason           sql.NullString
	Score            sql.NullInt64
	Active           bool
	StartedAt        time.Time
}

func (r *Repository) insertAssignment(ctx context.Context, tx *sql.Tx, input assignmentInput) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO lead_assignments
			(lead_id, owner_id, from_user_id, from_role, to_user_id, to_role, supervisor_id, assigned_by_user_id, action, reason, score, active, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.LeadID,
		input.OwnerID,
		input.FromUserID,
		input.FromRole,
		input.ToUserID,
		input.ToRole,
		input.SupervisorID,
		input.AssignedByUserID,
		input.Action,
		input.Reason,
		input.Score,
		input.Active,
		input.StartedAt,
	)
	return err
}

func (r *Repository) lockLead(ctx context.Context, tx *sql.Tx, id int64) (lockedLead, error) {
	var item lockedLead
	err := tx.QueryRowContext(ctx, `
		SELECT id, owner_id, current_owner_user_id, current_owner_role, supervisor_id,
			active_sales_id, stage, status, current_score
		FROM customer_leads
		WHERE id = ? AND deleted_at IS NULL
		FOR UPDATE`, id).
		Scan(
			&item.ID,
			&item.OwnerID,
			&item.CurrentOwnerUserID,
			&item.CurrentOwnerRole,
			&item.SupervisorID,
			&item.ActiveSalesID,
			&item.Stage,
			&item.Status,
			&item.CurrentScore,
		)
	if err == sql.ErrNoRows {
		return lockedLead{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ensureActiveUserRole(ctx context.Context, userID int64, roleCode string) error {
	var exists int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = ? AND r.code = ? AND u.status = 'ACTIVE' AND u.deleted_at IS NULL
		LIMIT 1`, userID, roleCode).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrUserNotValid
	}
	return err
}

func (r *Repository) firstActiveAdminID(ctx context.Context) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.code = 'ADMIN' AND u.status = 'ACTIVE' AND u.deleted_at IS NULL
		ORDER BY u.id
		LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrUserNotValid
	}
	return id, err
}

func leadSelect() string {
	return `
		SELECT
			COALESCE(cl.id, ot.id) AS id,
			COALESCE(cl.code, CONCAT('LEAD-', ot.code)) AS code,
			ot.owner_id,
			o.code, o.name, o.phone, o.brand_name, o.province, o.city,
			ot.id AS outlet_id,
			ot.code AS outlet_code,
			ot.name AS outlet_name,
			ot.phone AS outlet_phone,
			cl.active_sales_id, sales.name,
			COALESCE(cl.current_owner_user_id, o.id) AS current_owner_user_id,
			COALESCE(current_owner.name, o.name) AS current_owner_user_name,
			COALESCE(cl.current_owner_role, 'ADMIN') AS current_owner_role,
			cl.supervisor_id, supervisor.name,
			COALESCE(cl.source_type, 'MANUAL') AS source_type,
			COALESCE(cl.source_reference, '') AS source_reference,
			COALESCE(cl.stage, 'NEW') AS stage,
			COALESCE(cl.status, 'OPEN') AS status,
			COALESCE(cl.current_score, 1) AS current_score,
			cl.last_interaction_at, cl.next_follow_up_at, cl.invalidated_at, cl.invalidated_by_sales_id,
			COALESCE(cl.created_at, ot.created_at) AS created_at,
			COALESCE(cl.updated_at, ot.updated_at) AS updated_at
		FROM outlets ot
		JOIN owners o ON o.id = ot.owner_id AND o.deleted_at IS NULL
		LEFT JOIN customer_leads cl ON (cl.outlet_id = ot.id OR (cl.owner_id = ot.owner_id AND cl.outlet_id IS NULL)) AND cl.deleted_at IS NULL
		LEFT JOIN users current_owner ON current_owner.id = cl.current_owner_user_id
		LEFT JOIN users supervisor ON supervisor.id = cl.supervisor_id
		LEFT JOIN users sales ON sales.id = cl.active_sales_id
	`
}

func leadWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"ot.deleted_at IS NULL", "o.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := leadVisibilityWhere(actor)
	where = append(where, visibility)
	args = append(args, visibilityArgs...)

	if params.Query != "" {
		pattern := wordBoundaryRegexp(params.Query)
		where = append(where, "(COALESCE(cl.code, ot.code) REGEXP ? OR o.code REGEXP ? OR o.name REGEXP ? OR o.phone REGEXP ? OR o.brand_name REGEXP ? OR o.city REGEXP ? OR o.province REGEXP ? OR ot.name REGEXP ? OR ot.code REGEXP ?)")
		args = append(args, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern)
	}
	if params.Ownership != "" {
		where = append(where, "COALESCE(cl.current_owner_role, 'ADMIN') = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(params.Ownership)))
	}
	if params.Stage != "" {
		where = append(where, "COALESCE(cl.stage, 'NEW') = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(params.Stage)))
	}
	if params.Status != "" {
		where = append(where, "COALESCE(cl.status, 'OPEN') = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(params.Status)))
	}
	if params.Score != nil {
		where = append(where, "COALESCE(cl.current_score, 1) = ?")
		args = append(args, *params.Score)
	}
	if params.SupervisorID != nil {
		where = append(where, "cl.supervisor_id = ?")
		args = append(args, *params.SupervisorID)
	}
	if params.SalesID != nil {
		where = append(where, "cl.active_sales_id = ?")
		args = append(args, *params.SalesID)
	}
	if params.CreatedFrom != nil {
		where = append(where, "COALESCE(cl.created_at, ot.created_at) >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "COALESCE(cl.created_at, ot.created_at) < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	if params.FollowUpFrom != nil {
		where = append(where, "cl.next_follow_up_at >= ?")
		args = append(args, *params.FollowUpFrom)
	}
	if params.FollowUpTo != nil {
		where = append(where, "cl.next_follow_up_at <= ?")
		args = append(args, *params.FollowUpTo)
	}
	return strings.Join(where, " AND "), args
}

func leadVisibilityWhere(actor identity.User) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "1 = 1", nil
	case RoleSupervisor:
		return "(cl.current_owner_user_id = ? OR cl.supervisor_id = ?)", []any{actor.ID, actor.ID}
	case RoleSales:
		return "(cl.current_owner_role = 'SALES' AND cl.current_owner_user_id = ?)", []any{actor.ID}
	default:
		return "1 = 0", nil
	}
}

func leadOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"created_at":        "COALESCE(cl.created_at, ot.created_at)",
		"updated_at":        "COALESCE(cl.updated_at, ot.updated_at)",
		"code":              "COALESCE(cl.code, ot.code)",
		"stage":             "COALESCE(cl.stage, 'NEW')",
		"status":            "COALESCE(cl.status, 'OPEN')",
		"score":             "COALESCE(cl.current_score, 1)",
		"next_follow_up_at": "cl.next_follow_up_at",
		"owner_name":        "o.name",
		"outlet_name":       "ot.name",
	}, "COALESCE(cl.created_at, ot.created_at) DESC, ot.id DESC")
}

func scanLeads(rows *sql.Rows, total int64) ([]Lead, int64, error) {
	items := []Lead{}
	for rows.Next() {
		item, err := scanLead(rows)
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

func scanLead(row scanner) (Lead, error) {
	var item Lead
	err := row.Scan(
		&item.ID,
		&item.Code,
		&item.OwnerID,
		&item.OwnerCode,
		&item.OwnerName,
		&item.OwnerPhone,
		&item.OwnerBrandName,
		&item.OwnerProvince,
		&item.OwnerCity,
		&item.OutletID,
		&item.OutletCode,
		&item.OutletName,
		&item.OutletPhone,
		&item.ActiveSalesID,
		&item.ActiveSalesName,
		&item.CurrentOwnerUserID,
		&item.CurrentOwnerUserName,
		&item.CurrentOwnerRole,
		&item.SupervisorID,
		&item.SupervisorName,
		&item.SourceType,
		&item.SourceReference,
		&item.Stage,
		&item.Status,
		&item.CurrentScore,
		&item.LastInteractionAt,
		&item.NextFollowUpAt,
		&item.InvalidatedAt,
		&item.InvalidatedBySalesID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func scanAssignmentHistory(row scanner) (AssignmentHistory, error) {
	var item AssignmentHistory
	err := row.Scan(
		&item.ID,
		&item.LeadID,
		&item.OwnerID,
		&item.FromUserID,
		&item.FromUserName,
		&item.FromRole,
		&item.ToUserID,
		&item.ToUserName,
		&item.ToRole,
		&item.SupervisorID,
		&item.SupervisorName,
		&item.AssignedByID,
		&item.AssignedByName,
		&item.Action,
		&item.Reason,
		&item.Score,
		&item.Active,
		&item.StartedAt,
		&item.EndedAt,
		&item.CreatedAt,
	)
	return item, err
}

func canSeeLockedLead(actor identity.User, item lockedLead) bool {
	switch actor.RoleCode {
	case RoleAdmin:
		return true
	case RoleSupervisor:
		return currentBelongsToSupervisor(item, actor.ID)
	case RoleSales:
		return item.CurrentOwnerRole == RoleSales && item.CurrentOwnerUserID.Valid && item.CurrentOwnerUserID.Int64 == actor.ID
	default:
		return false
	}
}

func currentBelongsToSupervisor(item lockedLead, supervisorID int64) bool {
	return (item.CurrentOwnerRole == RoleSupervisor && item.CurrentOwnerUserID.Valid && item.CurrentOwnerUserID.Int64 == supervisorID) ||
		(item.SupervisorID.Valid && item.SupervisorID.Int64 == supervisorID)
}

func ensureOwnerExists(ctx context.Context, tx *sql.Tx, ownerID int64) error {
	var exists int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM owners WHERE id = ? AND deleted_at IS NULL LIMIT 1", ownerID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

func ensureOutletExists(ctx context.Context, tx *sql.Tx, ownerID, outletID int64) error {
	var exists int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM outlets WHERE id = ? AND owner_id = ? AND deleted_at IS NULL LIMIT 1", outletID, ownerID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

func normalizeIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, ErrEmptyBulk
	}
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			return nil, ErrEmptyBulk
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrEmptyBulk
	}
	return out, nil
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableInt64Ptr(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func like(value string) string {
	return "%" + strings.TrimSpace(value) + "%"
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

func mapCreateLeadError(err error) error {
	message := err.Error()
	if strings.Contains(message, "Duplicate entry") && strings.Contains(message, "uq_customer_leads_owner_id") {
		return ErrLeadAlreadyExists
	}
	return err
}

func wordBoundaryRegexp(value string) string {
	val := strings.TrimSpace(value)
	if val == "" {
		return ""
	}
	words := strings.Fields(val)
	var parts []string
	for _, w := range words {
		parts = append(parts, "\\b"+regexp.QuoteMeta(w)+"\\b")
	}
	return strings.Join(parts, ".*")
}
