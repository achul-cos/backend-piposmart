package activity

import (
	"context"
	"database/sql"
	"errors"
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

func (r *Repository) ListInteractions(ctx context.Context, actor identity.User, params InteractionListParams) ([]CustomerInteraction, int64, error) {
	where, args := interactionWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM customer_interactions ci
		LEFT JOIN customer_leads cl ON cl.id = ci.lead_id AND cl.deleted_at IS NULL
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := interactionOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := interactionSelect() + `
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
	return scanInteractions(rows, total)
}

func (r *Repository) CreateInteraction(ctx context.Context, actor identity.User, leadID int64, req CreateInteractionRequest) (CustomerInteraction, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CustomerInteraction{}, err
	}
	defer tx.Rollback()

	current, err := r.lockLead(ctx, tx, leadID)
	if err != nil {
		return CustomerInteraction{}, err
	}
	if !canWriteLeadActivity(actor, current) {
		return CustomerInteraction{}, ErrForbidden
	}

	interactionType, callStatus, chatStatus, err := resolveInteractionChannels(req.Type, req.CallStatus, req.ChatStatus)
	if err != nil {
		return CustomerInteraction{}, err
	}
	interactionAt := time.Now().UTC()
	if req.InteractionAt != nil {
		interactionAt = req.InteractionAt.UTC()
	}

	var reason sql.NullString
	var remarkReason RemarkReason
	hasRemark := req.RemarkScore != nil || req.RemarkReasonID != nil
	var next remarkPolicyResult
	if hasRemark {
		remarkReason, err = r.resolveRemarkReason(ctx, tx, req.RemarkReasonID, req.RemarkScore)
		if err != nil {
			return CustomerInteraction{}, err
		}
		if remarkReason.Score == 3 {
			return CustomerInteraction{}, ErrInvalidTransition
		}
		if remarkReason.Score == 0 && actor.RoleCode != RoleSales {
			return CustomerInteraction{}, ErrForbidden
		}
		next, err = applyRemarkPolicy(current, remarkReason.Score)
		if err != nil {
			return CustomerInteraction{}, err
		}
		reason = nullableString(remarkReason.Label)
	} else {
		next = remarkPolicyResult{Stage: current.Stage, Status: current.Status, Score: current.CurrentScore}
	}

	followUpAt := nullableTimePtrInput(req.FollowUpAt)
	if hasRemark && !followUpAt.Valid && remarkReason.DefaultFollowUpDays.Valid {
		followUpAt = sql.NullTime{
			Time:  interactionAt.AddDate(0, 0, int(remarkReason.DefaultFollowUpDays.Int64)),
			Valid: true,
		}
	}
	if strings.TrimSpace(req.Note) != "" {
		reason = nullableString(req.Note)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO customer_interactions
			(lead_id, owner_id, outlet_id, sales_id, supervisor_id, interaction_type, call_status, chat_status, interaction_at,
			 contact_name, contact_phone, duration_seconds, remark_reason_id, remark_score, remark_code, remark_label,
			 note, customer_response, follow_up_at, follow_up_note, stage_before, stage_after, status_before,
			 status_after, score_before, score_after, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		leadID,
		current.OwnerID,
		current.OutletID,
		salesIDForInteraction(actor, current),
		current.SupervisorID,
		interactionType,
		callStatus,
		chatStatus,
		interactionAt,
		nullableString(req.ContactName),
		nullableString(req.ContactPhone),
		nullableInt64PtrInput(req.DurationSeconds),
		nullableIDFromRemark(hasRemark, remarkReason.ID),
		nullableScoreFromRemark(hasRemark, remarkReason.Score),
		nullableStringFromRemark(hasRemark, remarkReason.Code),
		nullableStringFromRemark(hasRemark, remarkReason.Label),
		nullableString(req.Note),
		nullableString(req.CustomerResponse),
		followUpAt,
		nullableString(req.FollowUpNote),
		nullableString(current.Stage),
		nullableString(next.Stage),
		nullableString(current.Status),
		nullableString(next.Status),
		current.CurrentScore,
		next.Score,
		actor.ID,
	)
	if err != nil {
		return CustomerInteraction{}, err
	}
	interactionID, err := result.LastInsertId()
	if err != nil {
		return CustomerInteraction{}, err
	}

	if hasRemark {
		if err := r.applyLeadRemark(ctx, tx, actor, current, next, followUpAt, interactionAt, interactionID, reason); err != nil {
			return CustomerInteraction{}, err
		}
	} else if followUpAt.Valid {
		if _, err := tx.ExecContext(ctx, `
			UPDATE customer_leads
			SET last_interaction_at = ?, next_follow_up_at = ?
			WHERE id = ?`, interactionAt, followUpAt, leadID); err != nil {
			return CustomerInteraction{}, err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			UPDATE customer_leads
			SET last_interaction_at = ?
			WHERE id = ?`, interactionAt, leadID); err != nil {
			return CustomerInteraction{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return CustomerInteraction{}, err
	}
	return r.findInteractionByIDRaw(ctx, interactionID)
}

func (r *Repository) StageHistory(ctx context.Context, actor identity.User, leadID int64) ([]LeadStageHistory, error) {
	if _, err := r.findLeadForActor(ctx, actor, leadID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, stageHistorySelect()+`
		WHERE lsh.lead_id = ?
		ORDER BY lsh.created_at DESC, lsh.id DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []LeadStageHistory{}
	for rows.Next() {
		item, err := scanStageHistory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListTrainings(ctx context.Context, actor identity.User, params TrainingListParams) ([]TrainingReport, int64, error) {
	where, args := trainingWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM training_reports tr
		LEFT JOIN customer_leads cl ON cl.id = tr.lead_id AND cl.deleted_at IS NULL
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := trainingOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := trainingSelect() + `
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
	return scanTrainings(rows, total)
}

func (r *Repository) ScheduleTraining(ctx context.Context, actor identity.User, leadID int64, req ScheduleTrainingRequest) (TrainingReport, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TrainingReport{}, err
	}
	defer tx.Rollback()

	current, err := r.lockLead(ctx, tx, leadID)
	if err != nil {
		return TrainingReport{}, err
	}
	if !canWriteLeadActivity(actor, current) {
		return TrainingReport{}, ErrForbidden
	}
	trainingType, err := normalizeTrainingType(req.TrainingType)
	if err != nil {
		return TrainingReport{}, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO training_reports
			(lead_id, owner_id, outlet_id, sales_id, supervisor_id, training_type, status, scheduled_at,
			 location, meeting_url, trainer_name, participant_name, note, created_by_user_id, updated_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		leadID,
		current.OwnerID,
		current.OutletID,
		salesIDForTraining(actor, current),
		current.SupervisorID,
		trainingType,
		TrainingScheduled,
		req.ScheduledAt.UTC(),
		nullableString(req.Location),
		nullableString(req.MeetingURL),
		nil,
		nil,
		nullableString(req.Note),
		actor.ID,
		actor.ID,
	)
	if err != nil {
		return TrainingReport{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return TrainingReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return TrainingReport{}, err
	}
	return r.findTrainingByIDRaw(ctx, id)
}

func (r *Repository) RescheduleTraining(ctx context.Context, actor identity.User, trainingID int64, req RescheduleTrainingRequest) (TrainingReport, error) {
	return r.updateTrainingStatus(ctx, actor, trainingID, func(ctx context.Context, tx *sql.Tx, current TrainingReport) error {
		if current.Status != TrainingScheduled {
			return ErrInvalidTransition
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE training_reports
			SET scheduled_at = ?, rescheduled_at = ?, note = ?, updated_by_user_id = ?
			WHERE id = ?`,
			req.ScheduledAt.UTC(),
			time.Now().UTC(),
			nullableString(req.Note),
			actor.ID,
			trainingID,
		)
		return err
	})
}

func (r *Repository) CompleteTraining(ctx context.Context, actor identity.User, trainingID int64, req CompleteTrainingRequest) (TrainingReport, error) {
	return r.updateTrainingStatus(ctx, actor, trainingID, func(ctx context.Context, tx *sql.Tx, current TrainingReport) error {
		if current.Status != TrainingScheduled {
			return ErrInvalidTransition
		}
		completedAt := time.Now().UTC()
		if req.CompletedAt != nil {
			completedAt = req.CompletedAt.UTC()
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE training_reports
			SET status = ?, completed_at = ?, result_note = ?, updated_by_user_id = ?
			WHERE id = ?`,
			TrainingCompleted,
			completedAt,
			nullableString(req.ResultNote),
			actor.ID,
			trainingID,
		)
		return err
	})
}

func (r *Repository) CancelTraining(ctx context.Context, actor identity.User, trainingID int64, req CancelTrainingRequest) (TrainingReport, error) {
	return r.updateTrainingStatus(ctx, actor, trainingID, func(ctx context.Context, tx *sql.Tx, current TrainingReport) error {
		if current.Status != TrainingScheduled {
			return ErrInvalidTransition
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE training_reports
			SET status = ?, canceled_at = ?, cancel_reason = ?, updated_by_user_id = ?
			WHERE id = ?`,
			TrainingCanceled,
			time.Now().UTC(),
			nullableString(req.CancelReason),
			actor.ID,
			trainingID,
		)
		return err
	})
}

type trainingMutation func(context.Context, *sql.Tx, TrainingReport) error

func (r *Repository) updateTrainingStatus(ctx context.Context, actor identity.User, trainingID int64, mutate trainingMutation) (TrainingReport, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TrainingReport{}, err
	}
	defer tx.Rollback()

	current, err := r.lockTraining(ctx, tx, trainingID)
	if err != nil {
		return TrainingReport{}, err
	}
	if current.LeadID.Valid {
		leadState, err := r.lockLead(ctx, tx, current.LeadID.Int64)
		if err != nil {
			return TrainingReport{}, err
		}
		if !canWriteLeadActivity(actor, leadState) {
			return TrainingReport{}, ErrForbidden
		}
	} else if actor.RoleCode != RoleAdmin {
		return TrainingReport{}, ErrForbidden
	}

	if err := mutate(ctx, tx, current); err != nil {
		return TrainingReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return TrainingReport{}, err
	}
	return r.findTrainingByIDRaw(ctx, trainingID)
}

func (r *Repository) applyLeadRemark(
	ctx context.Context,
	tx *sql.Tx,
	actor identity.User,
	current LeadState,
	next remarkPolicyResult,
	followUpAt sql.NullTime,
	interactionAt time.Time,
	interactionID int64,
	reason sql.NullString,
) error {
	if leadStateChanged(current, next) {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO lead_stage_histories
				(lead_id, owner_id, from_stage, to_stage, from_status, to_status, from_score, to_score,
				 changed_by_user_id, source_type, source_id, reason)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'INTERACTION', ?, ?)`,
			current.ID,
			current.OwnerID,
			nullableString(current.Stage),
			next.Stage,
			nullableString(current.Status),
			next.Status,
			current.CurrentScore,
			next.Score,
			actor.ID,
			interactionID,
			reason,
		); err != nil {
			return err
		}
	}

	if next.Score.Valid && next.Score.Int64 == 0 {
		if !current.SupervisorID.Valid {
			return ErrLeadHasNoPIC
		}
		now := time.Now().UTC()
		if _, err := tx.ExecContext(ctx, `
			UPDATE lead_assignments
			SET active = FALSE, ended_at = ?
			WHERE lead_id = ? AND active = TRUE`, now, current.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO lead_assignments
				(lead_id, owner_id, from_user_id, from_role, to_user_id, to_role, supervisor_id,
				 assigned_by_user_id, action, reason, score, active, started_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, TRUE, ?)`,
			current.ID,
			current.OwnerID,
			current.CurrentOwnerUserID,
			nullableString(current.CurrentOwnerRole),
			current.SupervisorID,
			RoleSupervisor,
			current.SupervisorID,
			actor.ID,
			AssignmentInvalidatedBySales,
			reason,
			now,
		); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE customer_leads
			SET
				current_owner_user_id = ?,
				current_owner_role = 'SUPERVISOR',
				active_sales_id = NULL,
				stage = ?,
				status = ?,
				current_score = ?,
				last_interaction_at = ?,
				next_follow_up_at = ?,
				invalidated_at = ?,
				invalidated_by_sales_id = ?
			WHERE id = ?`,
			current.SupervisorID,
			next.Stage,
			next.Status,
			next.Score,
			interactionAt,
			followUpAt,
			now,
			actor.ID,
			current.ID,
		)
		return err
	}

	_, err := tx.ExecContext(ctx, `
		UPDATE customer_leads
		SET
			stage = ?,
			status = ?,
			current_score = ?,
			last_interaction_at = ?,
			next_follow_up_at = ?,
			invalidated_at = NULL,
			invalidated_by_sales_id = NULL
		WHERE id = ?`,
		next.Stage,
		next.Status,
		next.Score,
		interactionAt,
		followUpAt,
		current.ID,
	)
	return err
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

func (r *Repository) findLeadForActor(ctx context.Context, actor identity.User, id int64) (LeadState, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return LeadState{}, err
	}
	defer tx.Rollback()
	item, err := r.lockLead(ctx, tx, id)
	if err != nil {
		return LeadState{}, err
	}
	if !canReadLeadActivity(actor, item) {
		return LeadState{}, ErrNotFound
	}
	return item, tx.Commit()
}

func (r *Repository) resolveRemarkReason(ctx context.Context, tx *sql.Tx, reasonID *int64, score *int64) (RemarkReason, error) {
	if score != nil && (*score < 0 || *score > 3) {
		return RemarkReason{}, ErrInvalidScore
	}
	query := `
		SELECT id, score, code, label, default_follow_up_days, releases_assignment
		FROM remark_reasons
		WHERE active = TRUE`
	args := []any{}
	if reasonID != nil {
		query += " AND id = ?"
		args = append(args, *reasonID)
	}
	if score != nil {
		query += " AND score = ?"
		args = append(args, *score)
	}
	query += " ORDER BY id LIMIT 1"

	var item RemarkReason
	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&item.ID,
		&item.Score,
		&item.Code,
		&item.Label,
		&item.DefaultFollowUpDays,
		&item.ReleasesAssignment,
	)
	if err == sql.ErrNoRows {
		return RemarkReason{}, ErrRemarkNotFound
	}
	return item, err
}

func (r *Repository) findInteractionByIDRaw(ctx context.Context, id int64) (CustomerInteraction, error) {
	item, err := scanInteraction(r.db.QueryRowContext(ctx, interactionSelect()+`
		WHERE ci.id = ? AND ci.deleted_at IS NULL
		LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return CustomerInteraction{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findTrainingByIDRaw(ctx context.Context, id int64) (TrainingReport, error) {
	item, err := scanTraining(r.db.QueryRowContext(ctx, trainingSelect()+`
		WHERE tr.id = ? AND tr.deleted_at IS NULL
		LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return TrainingReport{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findTrainingByIDVisible(ctx context.Context, actor identity.User, id int64) (TrainingReport, error) {
	where, args := trainingWhere(actor, TrainingListParams{})
	args = append(args, id)

	item, err := scanTraining(r.db.QueryRowContext(ctx, trainingSelect()+`
		WHERE `+where+` AND tr.id = ?
		LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return TrainingReport{}, ErrForbidden
	}
	return item, err
}

func (r *Repository) GetTraining(ctx context.Context, actor identity.User, id int64) (TrainingReport, error) {
	item, err := r.findTrainingByIDVisible(ctx, actor, id)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, ErrForbidden) {
		return TrainingReport{}, err
	}
	if _, rawErr := r.findTrainingByIDRaw(ctx, id); rawErr != nil {
		return TrainingReport{}, rawErr
	}
	return TrainingReport{}, ErrForbidden
}

func (r *Repository) lockTraining(ctx context.Context, tx *sql.Tx, id int64) (TrainingReport, error) {
	item, err := scanTraining(tx.QueryRowContext(ctx, trainingSelect()+`
		WHERE tr.id = ? AND tr.deleted_at IS NULL
		FOR UPDATE`, id))
	if err == sql.ErrNoRows {
		return TrainingReport{}, ErrNotFound
	}
	return item, err
}

func interactionSelect() string {
	return `
		SELECT
			ci.id, ci.lead_id, ci.owner_id, ci.outlet_id,
			ci.sales_id, sales.name,
			ci.supervisor_id, supervisor.name,
			ci.interaction_type, ci.call_status, ci.chat_status, ci.interaction_at, ci.contact_name, ci.contact_phone,
			ci.duration_seconds, ci.remark_reason_id, ci.remark_score, ci.remark_code,
			ci.remark_label, ci.note, ci.customer_response, ci.follow_up_at, ci.follow_up_note,
			ci.stage_before, ci.stage_after, ci.status_before, ci.status_after,
			ci.score_before, ci.score_after, ci.created_by_user_id, created_by.name,
			ci.created_at, ci.updated_at
		FROM customer_interactions ci
		LEFT JOIN customer_leads cl ON cl.id = ci.lead_id AND cl.deleted_at IS NULL
		LEFT JOIN users sales ON sales.id = ci.sales_id
		LEFT JOIN users supervisor ON supervisor.id = ci.supervisor_id
		LEFT JOIN users created_by ON created_by.id = ci.created_by_user_id
	`
}

func stageHistorySelect() string {
	return `
		SELECT
			lsh.id, lsh.lead_id, lsh.owner_id, lsh.from_stage, lsh.to_stage,
			lsh.from_status, lsh.to_status, lsh.from_score, lsh.to_score,
			lsh.changed_by_user_id, changed_by.name, lsh.source_type, lsh.source_id,
			lsh.reason, lsh.created_at
		FROM lead_stage_histories lsh
		LEFT JOIN users changed_by ON changed_by.id = lsh.changed_by_user_id
	`
}

func trainingSelect() string {
	return `
		SELECT
			tr.id, tr.lead_id, cl.code,
			tr.owner_id, o.code, o.name,
			tr.outlet_id,
			tr.sales_id, sales.name,
			tr.supervisor_id, supervisor.name,
			tr.training_type, tr.status, tr.scheduled_at, tr.completed_at, tr.canceled_at,
			tr.rescheduled_at, tr.location, tr.meeting_url,
			tr.note, tr.result_note, tr.cancel_reason,
			tr.created_by_user_id, created_by.name,
			tr.updated_by_user_id, updated_by.name,
			tr.created_at, tr.updated_at
		FROM training_reports tr
		LEFT JOIN customer_leads cl ON cl.id = tr.lead_id AND cl.deleted_at IS NULL
		LEFT JOIN owners o ON o.id = tr.owner_id AND o.deleted_at IS NULL
		LEFT JOIN users sales ON sales.id = tr.sales_id
		LEFT JOIN users supervisor ON supervisor.id = tr.supervisor_id
		LEFT JOIN users created_by ON created_by.id = tr.created_by_user_id
		LEFT JOIN users updated_by ON updated_by.id = tr.updated_by_user_id
	`
}

func interactionWhere(actor identity.User, params InteractionListParams) (string, []any) {
	where := []string{"ci.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := activityVisibilityWhere(actor, "ci")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.LeadID != nil {
		where = append(where, "ci.lead_id = ?")
		args = append(args, *params.LeadID)
	}
	if params.Type != "" {
		switch strings.ToUpper(strings.TrimSpace(params.Type)) {
		case InteractionCall:
			where = append(where, "((ci.call_status IS NOT NULL AND ci.call_status <> '') OR ci.interaction_type = 'CALL')")
		case InteractionChat:
			where = append(where, "((ci.chat_status IS NOT NULL AND ci.chat_status <> '') OR ci.interaction_type = 'CHAT')")
		case InteractionCallChat:
			where = append(where, "(ci.call_status IS NOT NULL AND ci.call_status <> '') AND (ci.chat_status IS NOT NULL AND ci.chat_status <> '')")
		default:
			where = append(where, "ci.interaction_type = ?")
			args = append(args, strings.ToUpper(strings.TrimSpace(params.Type)))
		}
	}
	if params.Score != nil {
		where = append(where, "ci.remark_score = ?")
		args = append(args, *params.Score)
	}
	if params.SalesID != nil {
		where = append(where, "ci.sales_id = ?")
		args = append(args, *params.SalesID)
	}
	if params.CreatedFrom != nil {
		where = append(where, "ci.created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "ci.created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	if params.InteractionFrom != nil {
		where = append(where, "ci.interaction_at >= ?")
		args = append(args, *params.InteractionFrom)
	}
	if params.InteractionTo != nil {
		where = append(where, "ci.interaction_at <= ?")
		args = append(args, *params.InteractionTo)
	}
	if params.FollowUpFrom != nil {
		where = append(where, "ci.follow_up_at >= ?")
		args = append(args, *params.FollowUpFrom)
	}
	if params.FollowUpTo != nil {
		where = append(where, "ci.follow_up_at <= ?")
		args = append(args, *params.FollowUpTo)
	}
	if params.OnlyFollowUps {
		where = append(where, "ci.follow_up_at IS NOT NULL")
	}
	return strings.Join(where, " AND "), args
}

func trainingWhere(actor identity.User, params TrainingListParams) (string, []any) {
	where := []string{"tr.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := activityVisibilityWhere(actor, "tr")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.LeadID != nil {
		where = append(where, "tr.lead_id = ?")
		args = append(args, *params.LeadID)
	}
	if params.Status != "" {
		where = append(where, "tr.status = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(params.Status)))
	}
	if params.TrainingType != "" {
		where = append(where, "tr.training_type = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(params.TrainingType)))
	}
	if params.SalesID != nil {
		where = append(where, "tr.sales_id = ?")
		args = append(args, *params.SalesID)
	}
	if params.CreatedFrom != nil {
		where = append(where, "tr.created_at >= ?")
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		where = append(where, "tr.created_at < ?")
		args = append(args, params.CreatedTo.AddDate(0, 0, 1))
	}
	if params.ScheduledFrom != nil {
		where = append(where, "tr.scheduled_at >= ?")
		args = append(args, *params.ScheduledFrom)
	}
	if params.ScheduledTo != nil {
		where = append(where, "tr.scheduled_at <= ?")
		args = append(args, *params.ScheduledTo)
	}
	return strings.Join(where, " AND "), args
}

func activityVisibilityWhere(actor identity.User, activityAlias string) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "1 = 1", nil
	case RoleSupervisor:
		return fmt.Sprintf("(%s.supervisor_id = ? OR cl.current_owner_user_id = ? OR cl.supervisor_id = ?)", activityAlias), []any{actor.ID, actor.ID, actor.ID}
	case RoleSales:
		return fmt.Sprintf("(%s.sales_id = ? OR (cl.current_owner_role = 'SALES' AND cl.current_owner_user_id = ?))", activityAlias), []any{actor.ID, actor.ID}
	default:
		return "1 = 0", nil
	}
}

func interactionOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"interaction_at": "ci.interaction_at",
		"created_at":     "ci.created_at",
		"follow_up_at":   "ci.follow_up_at",
		"score":          "ci.remark_score",
		"type":           "ci.interaction_type",
	}, "ci.interaction_at DESC, ci.id DESC")
}

func trainingOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"scheduled_at":  "tr.scheduled_at",
		"created_at":    "tr.created_at",
		"updated_at":    "tr.updated_at",
		"status":        "tr.status",
		"training_type": "tr.training_type",
	}, "tr.scheduled_at DESC, tr.id DESC")
}

func scanInteractions(rows *sql.Rows, total int64) ([]CustomerInteraction, int64, error) {
	items := []CustomerInteraction{}
	for rows.Next() {
		item, err := scanInteraction(rows)
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

func scanInteraction(row scanner) (CustomerInteraction, error) {
	var item CustomerInteraction
	err := row.Scan(
		&item.ID,
		&item.LeadID,
		&item.OwnerID,
		&item.OutletID,
		&item.SalesID,
		&item.SalesName,
		&item.SupervisorID,
		&item.SupervisorName,
		&item.InteractionType,
		&item.CallStatus,
		&item.ChatStatus,
		&item.InteractionAt,
		&item.ContactName,
		&item.ContactPhone,
		&item.DurationSeconds,
		&item.RemarkReasonID,
		&item.RemarkScore,
		&item.RemarkCode,
		&item.RemarkLabel,
		&item.Note,
		&item.CustomerResponse,
		&item.FollowUpAt,
		&item.FollowUpNote,
		&item.StageBefore,
		&item.StageAfter,
		&item.StatusBefore,
		&item.StatusAfter,
		&item.ScoreBefore,
		&item.ScoreAfter,
		&item.CreatedByUserID,
		&item.CreatedByName,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func scanStageHistory(row scanner) (LeadStageHistory, error) {
	var item LeadStageHistory
	err := row.Scan(
		&item.ID,
		&item.LeadID,
		&item.OwnerID,
		&item.FromStage,
		&item.ToStage,
		&item.FromStatus,
		&item.ToStatus,
		&item.FromScore,
		&item.ToScore,
		&item.ChangedByUserID,
		&item.ChangedByName,
		&item.SourceType,
		&item.SourceID,
		&item.Reason,
		&item.CreatedAt,
	)
	return item, err
}

func scanTrainings(rows *sql.Rows, total int64) ([]TrainingReport, int64, error) {
	items := []TrainingReport{}
	for rows.Next() {
		item, err := scanTraining(rows)
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

func scanTraining(row scanner) (TrainingReport, error) {
	var item TrainingReport
	err := row.Scan(
		&item.ID,
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
		&item.TrainingType,
		&item.Status,
		&item.ScheduledAt,
		&item.CompletedAt,
		&item.CanceledAt,
		&item.RescheduledAt,
		&item.Location,
		&item.MeetingURL,
		&item.Note,
		&item.ResultNote,
		&item.CancelReason,
		&item.CreatedByUserID,
		&item.CreatedByName,
		&item.UpdatedByUserID,
		&item.UpdatedByName,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func canReadLeadActivity(actor identity.User, item LeadState) bool {
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

func canWriteLeadActivity(actor identity.User, item LeadState) bool {
	return canReadLeadActivity(actor, item)
}

func currentBelongsToSupervisor(item LeadState, supervisorID int64) bool {
	return (item.CurrentOwnerRole == RoleSupervisor && item.CurrentOwnerUserID.Valid && item.CurrentOwnerUserID.Int64 == supervisorID) ||
		(item.SupervisorID.Valid && item.SupervisorID.Int64 == supervisorID)
}

func salesIDForInteraction(actor identity.User, current LeadState) sql.NullInt64 {
	if actor.RoleCode == RoleSales {
		return sql.NullInt64{Int64: actor.ID, Valid: true}
	}
	if current.ActiveSalesID.Valid {
		return current.ActiveSalesID
	}
	return sql.NullInt64{}
}

func salesIDForTraining(actor identity.User, current LeadState) sql.NullInt64 {
	return salesIDForInteraction(actor, current)
}

func normalizeInteractionType(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case InteractionCall, InteractionChat, InteractionCallChat:
		return value, nil
	default:
		return "", ErrInvalidType
	}
}

func resolveInteractionChannels(legacyType string, callStatus string, chatStatus string) (string, sql.NullString, sql.NullString, error) {
	call := nullableString(callStatus)
	chat := nullableString(chatStatus)

	if call.Valid && chat.Valid {
		return InteractionCallChat, call, chat, nil
	}
	if call.Valid {
		return InteractionCall, call, chat, nil
	}
	if chat.Valid {
		return InteractionChat, call, chat, nil
	}

	if strings.TrimSpace(legacyType) == "" {
		return "", sql.NullString{}, sql.NullString{}, ErrInteractionEmpty
	}
	normalized, err := normalizeInteractionType(legacyType)
	if err != nil {
		return "", sql.NullString{}, sql.NullString{}, err
	}
	switch normalized {
	case InteractionCall:
		call = nullableString("RECORDED")
	case InteractionChat:
		chat = nullableString("RECORDED")
	case InteractionCallChat:
		call = nullableString("RECORDED")
		chat = nullableString("RECORDED")
	}
	return normalized, call, chat, nil
}

func normalizeTrainingType(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case TrainingOnline, TrainingOffline:
		return value, nil
	default:
		return "", ErrInvalidType
	}
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

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableTimePtrInput(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func nullableInt64PtrInput(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullableIDFromRemark(valid bool, id int64) sql.NullInt64 {
	if !valid {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
}

func nullableScoreFromRemark(valid bool, score int64) sql.NullInt64 {
	if !valid {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: score, Valid: true}
}

func nullableStringFromRemark(valid bool, value string) sql.NullString {
	if !valid {
		return sql.NullString{}
	}
	return nullableString(value)
}
