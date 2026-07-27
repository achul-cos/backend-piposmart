package partner

import (
	"context"
	"database/sql"
	"encoding/json"
)

func (r *Repository) Audit(ctx context.Context, actorUserID int64, action string, entityType string, entityID int64, before any, after any, meta RequestMeta) error {
	beforeJSON, err := marshalNullableJSON(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalNullableJSON(after)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs
			(actor_user_id, action, entity_type, entity_id, before_json, after_json, ip_address, user_agent, request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableAuditInt64(actorUserID),
		action,
		entityType,
		nullableAuditInt64(entityID),
		beforeJSON,
		afterJSON,
		nullableString(meta.IPAddress),
		nullableString(meta.UserAgent),
		nullableString(meta.RequestID),
	)
	return err
}

func (r *Repository) ListPartnerTypeHistories(ctx context.Context, entityID int64) ([]auditLogRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.action, a.entity_type, a.entity_id, a.actor_user_id, u.name,
			CAST(a.before_json AS CHAR), CAST(a.after_json AS CHAR), a.request_id, a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE a.entity_type = ? AND a.entity_id = ?
		ORDER BY a.created_at DESC, a.id DESC`, entityTypePartnerType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]auditLogRecord, 0)
	for rows.Next() {
		var item auditLogRecord
		if err := rows.Scan(
			&item.ID,
			&item.Action,
			&item.EntityType,
			&item.EntityID,
			&item.ActorUserID,
			&item.ActorName,
			&item.BeforeRaw,
			&item.AfterRaw,
			&item.RequestID,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func nullableAuditInt64(value int64) sql.NullInt64 {
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
