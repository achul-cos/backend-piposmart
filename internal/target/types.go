package target

import (
	"database/sql"
	"time"
)

const (
	SourceBulk     = "BULK"
	SourceOverride = "OVERRIDE"

	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"
)

// SalesTarget is one (sales, metric, period) target row.
type SalesTarget struct {
	ID              int64
	SalesID         int64
	SalesName       string
	SalesCode       sql.NullString
	MetricCodeID    int64
	MetricCode      string
	MetricName      string
	MetricUnit      string
	PeriodYear      int
	PeriodMonth     int
	TargetValue     string
	Source          string
	CreatedByUserID sql.NullInt64
	CreatedByName   sql.NullString
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ActorRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type SalesTargetResponse struct {
	ID          int64     `json:"id"`
	SalesID     int64     `json:"sales_id"`
	SalesName   string    `json:"sales_name"`
	SalesCode   *string   `json:"sales_code,omitempty"`
	MetricCode  string    `json:"metric_code"`
	MetricName  string    `json:"metric_name"`
	MetricUnit  string    `json:"metric_unit"`
	PeriodYear  int       `json:"period_year"`
	PeriodMonth int       `json:"period_month"`
	TargetValue string    `json:"target_value"`
	Source      string    `json:"source"`
	CreatedBy   *ActorRef `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewSalesTargetResponse(t SalesTarget) SalesTargetResponse {
	resp := SalesTargetResponse{
		ID:          t.ID,
		SalesID:     t.SalesID,
		SalesName:   t.SalesName,
		MetricCode:  t.MetricCode,
		MetricName:  t.MetricName,
		MetricUnit:  t.MetricUnit,
		PeriodYear:  t.PeriodYear,
		PeriodMonth: t.PeriodMonth,
		TargetValue: t.TargetValue,
		Source:      t.Source,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if t.SalesCode.Valid {
		code := t.SalesCode.String
		resp.SalesCode = &code
	}
	if t.CreatedByUserID.Valid {
		resp.CreatedBy = &ActorRef{ID: t.CreatedByUserID.Int64, Name: t.CreatedByName.String}
	}
	return resp
}

type SalesTargetListResponse struct {
	Items []SalesTargetResponse `json:"items"`
	Meta  ListMeta              `json:"meta"`
}

type ListMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// BulkSetTargetRequest sets a default target for every eligible active Sales rep who does not
// already have a target row for this (metric, period) — never overwrites an existing row,
// whether it was set by an earlier bulk call or an override.
type BulkSetTargetRequest struct {
	PeriodYear  int    `json:"period_year" binding:"required"`
	PeriodMonth int    `json:"period_month" binding:"required,min=1,max=12"`
	MetricCode  string `json:"metric_code" binding:"required"`
	TargetValue string `json:"target_value" binding:"required"`
}

type BulkSetTargetResponse struct {
	MetricCode    string `json:"metric_code"`
	PeriodYear    int    `json:"period_year"`
	PeriodMonth   int    `json:"period_month"`
	TargetValue   string `json:"target_value"`
	EligibleSales int    `json:"eligible_sales"`
	Created       int    `json:"created"`
}

// OverrideTargetRequest always wins over a bulk-set value for the given sales rep.
type OverrideTargetRequest struct {
	PeriodYear  int    `json:"period_year" binding:"required"`
	PeriodMonth int    `json:"period_month" binding:"required,min=1,max=12"`
	MetricCode  string `json:"metric_code" binding:"required"`
	TargetValue string `json:"target_value" binding:"required"`
}

type ListTargetsParams struct {
	SalesID     *int64
	PeriodYear  *int
	PeriodMonth *int
	MetricCode  string
	Page        int
	Limit       int
}
