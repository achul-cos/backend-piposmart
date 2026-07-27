package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

func (r *Repository) Audit(ctx context.Context, entry CatalogAuditEntry) error {
	beforeJSON, err := marshalNullableJSON(entry.Before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalNullableJSON(entry.After)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs
			(actor_user_id, action, entity_type, entity_id, before_json, after_json, ip_address, user_agent, request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableInt64(entry.ActorUserID),
		entry.Action,
		entry.EntityType,
		nullableInt64(entry.EntityID),
		beforeJSON,
		afterJSON,
		nullableString(entry.IPAddress),
		nullableString(entry.UserAgent),
		nullableString(entry.RequestID),
	)
	return err
}

func (r *Repository) ListHistories(ctx context.Context, entityType string, entityID int64) ([]AuditLogRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.action, a.entity_type, a.entity_id, a.actor_user_id, u.name,
			CAST(a.before_json AS CHAR), CAST(a.after_json AS CHAR), a.request_id, a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE a.entity_type = ? AND a.entity_id = ?
		ORDER BY a.created_at DESC, a.id DESC`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AuditLogRecord, 0)
	for rows.Next() {
		var (
			item      AuditLogRecord
			actorID   sql.NullInt64
			actorName sql.NullString
			beforeRaw sql.NullString
			afterRaw  sql.NullString
			requestID sql.NullString
		)
		if err := rows.Scan(&item.ID, &item.Action, &item.EntityType, &item.EntityID, &actorID, &actorName, &beforeRaw, &afterRaw, &requestID, &item.CreatedAt); err != nil {
			return nil, err
		}
		if actorID.Valid {
			value := actorID.Int64
			item.ActorUserID = &value
		}
		item.ActorName = actorName.String
		item.RequestID = requestID.String
		if beforeRaw.Valid && strings.TrimSpace(beforeRaw.String) != "" {
			var before any
			if err := json.Unmarshal([]byte(beforeRaw.String), &before); err == nil {
				item.Before = before
			}
		}
		if afterRaw.Valid && strings.TrimSpace(afterRaw.String) != "" {
			var after any
			if err := json.Unmarshal([]byte(afterRaw.String), &after); err == nil {
				item.After = after
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) FindPackageByIDAny(ctx context.Context, id int64) (Package, error) {
	item, err := scanPackage(r.db.QueryRowContext(ctx, packageSelect()+" WHERE sp.id = ? LIMIT 1", id))
	if err == sql.ErrNoRows {
		return Package{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) FindPlanByIDAny(ctx context.Context, id int64) (Plan, error) {
	item, err := scanPlan(r.db.QueryRowContext(ctx, planSelect()+" WHERE spl.id = ? LIMIT 1", id))
	if err == sql.ErrNoRows {
		return Plan{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) FindPromotionByIDAny(ctx context.Context, id int64) (Promotion, error) {
	item, err := scanPromotion(r.db.QueryRowContext(ctx, promotionSelect()+" WHERE p.id = ? LIMIT 1", id))
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

func (r *Repository) ListEligiblePlanIDs(ctx context.Context, promotionID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT plan_id
		FROM promotion_plan_eligibilities
		WHERE promotion_id = ?
		ORDER BY plan_id ASC`, promotionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, rows.Err()
}

func nullableInt64(value int64) sql.NullInt64 {
	if value < 1 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

func marshalNullableJSON(value any) (sql.NullString, error) {
	if value == nil {
		return sql.NullString{}, nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(bytes), Valid: true}, nil
}

func historyResponses(records []AuditLogRecord) HistoryListResponse {
	items := make([]HistoryEntryResponse, 0, len(records))
	for _, record := range records {
		items = append(items, HistoryEntryResponse{
			ID:         record.ID,
			Action:     record.Action,
			EntityType: record.EntityType,
			EntityID:   record.EntityID,
			Actor: HistoryActorResponse{
				ID:   record.ActorUserID,
				Name: record.ActorName,
			},
			RequestID: record.RequestID,
			Before:    record.Before,
			After:     record.After,
			CreatedAt: record.CreatedAt,
		})
	}
	return HistoryListResponse{Items: items, Total: len(items)}
}

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
