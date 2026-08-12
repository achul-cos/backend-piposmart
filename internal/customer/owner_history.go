package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// RequestMeta carries request-scoped metadata for audit logging, mirroring internal/partner's
// pattern for the same generic audit_logs table (Sprint 8 scraper update-sync addendum).
type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

type HistoryActor struct {
	ID   *int64 `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type HistoryEntry struct {
	ID         int64        `json:"id"`
	Action     string       `json:"action"`
	EntityType string       `json:"entity_type"`
	EntityID   int64        `json:"entity_id"`
	Actor      HistoryActor `json:"actor"`
	RequestID  string       `json:"request_id,omitempty"`
	Before     any          `json:"before,omitempty"`
	After      any          `json:"after,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

type HistoryListResponse struct {
	Items      []HistoryEntry `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type auditLogRecord struct {
	ID          int64
	Action      string
	EntityType  string
	EntityID    int64
	ActorUserID sql.NullInt64
	ActorName   sql.NullString
	BeforeRaw   sql.NullString
	AfterRaw    sql.NullString
	RequestID   sql.NullString
	CreatedAt   time.Time
}

const (
	entityTypeOwner  = "customer.owner"
	entityTypeOutlet = "customer.outlet"
)

func nullableAuditInt64(id int64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
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

// Audit writes one entry to the shared audit_logs table (already used by internal/partner and
// internal/catalog for their own entities). Owner/Outlet updates previously wrote nothing here at
// all - this closes that gap so both human-made and scraper-made changes are visible uniformly.
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

func (r *Repository) listEntityHistories(ctx context.Context, entityType string, entityID int64) ([]auditLogRecord, error) {
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

	items := make([]auditLogRecord, 0)
	for rows.Next() {
		var item auditLogRecord
		if err := rows.Scan(
			&item.ID, &item.Action, &item.EntityType, &item.EntityID, &item.ActorUserID, &item.ActorName,
			&item.BeforeRaw, &item.AfterRaw, &item.RequestID, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func historyRecordsToResponse(records []auditLogRecord) *HistoryListResponse {
	items := make([]HistoryEntry, 0, len(records))
	for _, record := range records {
		var actorID *int64
		if record.ActorUserID.Valid {
			value := record.ActorUserID.Int64
			actorID = &value
		}
		entry := HistoryEntry{
			ID:         record.ID,
			Action:     record.Action,
			EntityType: record.EntityType,
			EntityID:   record.EntityID,
			Actor:      HistoryActor{ID: actorID, Name: record.ActorName.String},
			RequestID:  record.RequestID.String,
			CreatedAt:  record.CreatedAt,
		}
		if record.BeforeRaw.Valid && record.BeforeRaw.String != "" {
			var before any
			if err := json.Unmarshal([]byte(record.BeforeRaw.String), &before); err == nil {
				entry.Before = before
			}
		}
		if record.AfterRaw.Valid && record.AfterRaw.String != "" {
			var after any
			if err := json.Unmarshal([]byte(record.AfterRaw.String), &after); err == nil {
				entry.After = after
			}
		}
		items = append(items, entry)
	}
	return &HistoryListResponse{
		Items:      items,
		Pagination: PaginationMeta{Page: 1, Limit: len(items), Total: int64(len(items))},
	}
}

// ListOwnerHistories returns the audit trail for one owner (both human-made changes via the CRM
// UI and scraper-pushed updates from the admin-dashboard sync pipeline appear here uniformly,
// since the scraper authenticates as a normal API client - see Sprint 8 scraper plan).
func (s *Service) ListOwnerHistories(ctx context.Context, actor Actor, id int64) (*HistoryListResponse, error) {
	if _, err := s.repo.FindOwnerByID(ctx, actor, id); err != nil {
		return nil, err
	}
	records, err := s.repo.listEntityHistories(ctx, entityTypeOwner, id)
	if err != nil {
		return nil, err
	}
	return historyRecordsToResponse(records), nil
}

// ListOutletHistories returns the audit trail for one outlet.
func (s *Service) ListOutletHistories(ctx context.Context, actor Actor, ownerID, outletID int64) (*HistoryListResponse, error) {
	if _, err := s.repo.FindOutletByID(ctx, actor, ownerID, outletID); err != nil {
		return nil, err
	}
	records, err := s.repo.listEntityHistories(ctx, entityTypeOutlet, outletID)
	if err != nil {
		return nil, err
	}
	return historyRecordsToResponse(records), nil
}
