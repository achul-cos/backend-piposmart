package catalog

import "time"

type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

type CatalogAuditEntry struct {
	ActorUserID int64
	Action      string
	EntityType  string
	EntityID    int64
	Before      any
	After       any
	IPAddress   string
	UserAgent   string
	RequestID   string
}

type HistoryActorResponse struct {
	ID   *int64 `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type HistoryEntryResponse struct {
	ID         int64                `json:"id"`
	Action     string               `json:"action"`
	EntityType string               `json:"entity_type"`
	EntityID   int64                `json:"entity_id"`
	Actor      HistoryActorResponse `json:"actor"`
	RequestID  string               `json:"request_id,omitempty"`
	Before     any                  `json:"before,omitempty"`
	After      any                  `json:"after,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
}

type HistoryListResponse struct {
	Items []HistoryEntryResponse `json:"items"`
	Total int                    `json:"total"`
}

type PromotionAuditSnapshot struct {
	Promotion       PromotionResponse `json:"promotion"`
	EligiblePlanIDs []int64           `json:"eligible_plan_ids,omitempty"`
}

type AuditLogRecord struct {
	ID          int64
	Action      string
	EntityType  string
	EntityID    int64
	ActorUserID *int64
	ActorName   string
	RequestID   string
	Before      any
	After       any
	CreatedAt   time.Time
}

const (
	entityTypeCatalogPackage   = "catalog.package"
	entityTypeCatalogPlan      = "catalog.plan"
	entityTypeCatalogPromotion = "catalog.promotion"
)
