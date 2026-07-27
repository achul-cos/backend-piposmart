package partner

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

type PartnerTypeHistoryActor struct {
	ID   *int64 `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type PartnerTypeHistoryEntry struct {
	ID         int64                   `json:"id"`
	Action     string                  `json:"action"`
	EntityType string                  `json:"entity_type"`
	EntityID   int64                   `json:"entity_id"`
	Actor      PartnerTypeHistoryActor `json:"actor"`
	RequestID  string                  `json:"request_id,omitempty"`
	Before     any                     `json:"before,omitempty"`
	After      any                     `json:"after,omitempty"`
	CreatedAt  time.Time               `json:"created_at"`
}

type PartnerTypeHistoryListResponse struct {
	Items      []PartnerTypeHistoryEntry `json:"items"`
	Pagination PaginationMeta            `json:"pagination"`
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

const entityTypePartnerType = "partner.type"

func canManagePartnerType(actor identity.User) bool {
	return actor.RoleCode == identity.RoleAdmin
}

func (s *Service) CreatePartnerTypeWithMeta(ctx context.Context, actor identity.User, req CreatePartnerTypeRequest, meta RequestMeta) (*PartnerTypeResponse, error) {
	if !canManagePartnerType(actor) {
		return nil, ErrForbidden
	}
	resp, err := s.CreatePartnerType(ctx, req)
	if err != nil {
		return nil, err
	}
	_ = s.repo.Audit(ctx, actor.ID, "partner_type.create", entityTypePartnerType, resp.ID, nil, resp, meta)
	return resp, nil
}

func (s *Service) UpdatePartnerTypeWithMeta(ctx context.Context, actor identity.User, id int64, req UpdatePartnerTypeRequest, meta RequestMeta) (*PartnerTypeResponse, error) {
	if !canManagePartnerType(actor) {
		return nil, ErrForbidden
	}
	before, err := s.GetPartnerTypeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	after, err := s.UpdatePartnerType(ctx, id, req)
	if err != nil {
		return nil, err
	}
	_ = s.repo.Audit(ctx, actor.ID, "partner_type.update", entityTypePartnerType, id, before, after, meta)
	return after, nil
}

func (s *Service) ListPartnerTypeHistories(ctx context.Context, id int64) (*PartnerTypeHistoryListResponse, error) {
	if _, err := s.repo.GetPartnerTypeByID(ctx, id); err != nil {
		return nil, err
	}
	records, err := s.repo.ListPartnerTypeHistories(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]PartnerTypeHistoryEntry, 0, len(records))
	for _, record := range records {
		var actorID *int64
		if record.ActorUserID.Valid {
			value := record.ActorUserID.Int64
			actorID = &value
		}
		entry := PartnerTypeHistoryEntry{
			ID:         record.ID,
			Action:     record.Action,
			EntityType: record.EntityType,
			EntityID:   record.EntityID,
			Actor:      PartnerTypeHistoryActor{ID: actorID, Name: record.ActorName.String},
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
	return &PartnerTypeHistoryListResponse{
		Items:      items,
		Pagination: PaginationMeta{Page: 1, Limit: len(items), Total: int64(len(items))},
	}, nil
}
