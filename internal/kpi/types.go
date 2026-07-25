package kpi

import (
	"database/sql"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	ClassificationAchieved     = "ACHIEVED"
	ClassificationNearAchieved = "NEAR_ACHIEVED"
	ClassificationNotAchieved  = "NOT_ACHIEVED"

	JobTypeRecompute = "KPI_RECOMPUTE"
)

// KpiDefinition is one metric's weight/threshold configuration for a period.
type KpiDefinition struct {
	ID                int64
	MetricCodeID      int64
	MetricCode        string
	MetricName        string
	MetricUnit        string
	PeriodYear        int
	PeriodMonth       int
	Weight            string
	ThresholdAchieved string
	ThresholdNear     string
	Active            bool
	CreatedByUserID   sql.NullInt64
	CreatedByName     sql.NullString
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ActorRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type KpiDefinitionResponse struct {
	ID                int64     `json:"id"`
	MetricCode        string    `json:"metric_code"`
	MetricName        string    `json:"metric_name"`
	MetricUnit        string    `json:"metric_unit"`
	PeriodYear        int       `json:"period_year"`
	PeriodMonth       int       `json:"period_month"`
	Weight            string    `json:"weight"`
	ThresholdAchieved string    `json:"threshold_achieved"`
	ThresholdNear     string    `json:"threshold_near"`
	Active            bool      `json:"active"`
	CreatedBy         *ActorRef `json:"created_by,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewKpiDefinitionResponse(d KpiDefinition) KpiDefinitionResponse {
	resp := KpiDefinitionResponse{
		ID:                d.ID,
		MetricCode:        d.MetricCode,
		MetricName:        d.MetricName,
		MetricUnit:        d.MetricUnit,
		PeriodYear:        d.PeriodYear,
		PeriodMonth:       d.PeriodMonth,
		Weight:            d.Weight,
		ThresholdAchieved: d.ThresholdAchieved,
		ThresholdNear:     d.ThresholdNear,
		Active:            d.Active,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
	if d.CreatedByUserID.Valid {
		resp.CreatedBy = &ActorRef{ID: d.CreatedByUserID.Int64, Name: d.CreatedByName.String}
	}
	return resp
}

type CreateKpiDefinitionRequest struct {
	MetricCode        string `json:"metric_code" binding:"required"`
	PeriodYear        int    `json:"period_year" binding:"required"`
	PeriodMonth       int    `json:"period_month" binding:"required,min=1,max=12"`
	Weight            string `json:"weight" binding:"required"`
	ThresholdAchieved string `json:"threshold_achieved"`
	ThresholdNear     string `json:"threshold_near"`
}

type KpiDefinitionListResponse struct {
	Items []KpiDefinitionResponse `json:"items"`
}

// SalesKpiMetricResult is one metric's contribution within a recompute run.
type SalesKpiMetricResult struct {
	ID              int64
	SalesID         int64
	KpiDefinitionID int64
	MetricCode      string
	PeriodYear      int
	PeriodMonth     int
	TargetValue     string
	ActualValue     string
	AchievementPct  string
	WeightedScore   string
}

type SalesKpiMetricResultResponse struct {
	MetricCode     string `json:"metric_code"`
	TargetValue    string `json:"target_value"`
	ActualValue    string `json:"actual_value"`
	AchievementPct string `json:"achievement_pct"`
	WeightedScore  string `json:"weighted_score"`
}

// SalesKpiResult is the overall per-sales-per-period summary produced by a recompute run.
type SalesKpiResult struct {
	ID             int64
	SalesID        int64
	SalesName      string
	SalesCode      sql.NullString
	PeriodYear     int
	PeriodMonth    int
	TotalScore     string
	Classification string
	RankPosition   sql.NullInt64
	ComputedAt     time.Time
	JobID          sql.NullInt64
	Metrics        []SalesKpiMetricResultResponse
}

type SalesKpiResultResponse struct {
	SalesID        int64                          `json:"sales_id"`
	SalesName      string                         `json:"sales_name"`
	SalesCode      *string                        `json:"sales_code,omitempty"`
	PeriodYear     int                            `json:"period_year"`
	PeriodMonth    int                            `json:"period_month"`
	TotalScore     string                         `json:"total_score"`
	Classification string                         `json:"classification"`
	RankPosition   *int64                         `json:"rank_position,omitempty"`
	ComputedAt     time.Time                      `json:"computed_at"`
	Metrics        []SalesKpiMetricResultResponse `json:"metrics,omitempty"`
}

func NewSalesKpiResultResponse(r SalesKpiResult) SalesKpiResultResponse {
	resp := SalesKpiResultResponse{
		SalesID:        r.SalesID,
		SalesName:      r.SalesName,
		PeriodYear:     r.PeriodYear,
		PeriodMonth:    r.PeriodMonth,
		TotalScore:     r.TotalScore,
		Classification: r.Classification,
		ComputedAt:     r.ComputedAt,
		Metrics:        r.Metrics,
	}
	if r.SalesCode.Valid {
		code := r.SalesCode.String
		resp.SalesCode = &code
	}
	if r.RankPosition.Valid {
		rank := r.RankPosition.Int64
		resp.RankPosition = &rank
	}
	return resp
}

type SalesKpiResultListResponse struct {
	Items []SalesKpiResultResponse `json:"items"`
}

type RecomputeRequest struct {
	PeriodYear  int `json:"period_year" binding:"required"`
	PeriodMonth int `json:"period_month" binding:"required,min=1,max=12"`
}

type RecomputeJobPayload struct {
	PeriodYear  int `json:"period_year"`
	PeriodMonth int `json:"period_month"`
}

type JobResponse struct {
	ID          int64      `json:"id"`
	JobType     string     `json:"job_type"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	LastError   *string    `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type ListResultsParams struct {
	SalesID     *int64
	PeriodYear  *int
	PeriodMonth *int
}
