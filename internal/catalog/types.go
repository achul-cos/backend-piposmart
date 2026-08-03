package catalog

import (
	"database/sql"
	"encoding/json"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"

	ChargeFree = "FREE"
	ChargePaid = "PAID"

	ScopeActive  = "ACTIVE"
	ScopeDeleted = "DELETED"
	ScopeAll     = "ALL"
)

type Package struct {
	ID          int64
	Code        string
	Name        string
	LevelOrder  int
	Description sql.NullString
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Plan struct {
	ID            int64
	PackageID     int64
	PackageCode   string
	PackageName   string
	Code          string
	Name          string
	TenureMonths  int
	DurationDays  int
	Price         string
	Currency      string
	EffectiveFrom time.Time
	EffectiveTo   sql.NullTime
	Active        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Promotion struct {
	ID               int64
	Code             string
	Name             string
	PromotionType    string
	ChargeType       string
	AdditionalCharge string
	Priority         int
	Description      sql.NullString
	EffectiveFrom    time.Time
	EffectiveTo      sql.NullTime
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Benefits         []Benefit
}

type Benefit struct {
	ID           int64
	PromotionID  int64
	BenefitType  string
	PackageID    sql.NullInt64
	PackageCode  sql.NullString
	PackageName  sql.NullString
	DurationDays sql.NullInt64
	Quantity     sql.NullInt64
	Description  sql.NullString
	MetadataJSON sql.NullString
	CreatedAt    time.Time
}

type PackageResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	LevelOrder  int       `json:"level_order"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PlanResponse struct {
	ID            int64      `json:"id"`
	Package       PackageRef `json:"package"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	TenureMonths  int        `json:"tenure_months"`
	DurationDays  int        `json:"duration_days"`
	Price         string     `json:"price"`
	Currency      string     `json:"currency"`
	EffectiveFrom string     `json:"effective_from"`
	EffectiveTo   string     `json:"effective_to,omitempty"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PackageRef struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type PromotionResponse struct {
	ID               int64             `json:"id"`
	Code             string            `json:"code"`
	Name             string            `json:"name"`
	PromotionType    string            `json:"promotion_type"`
	ChargeType       string            `json:"charge_type"`
	AdditionalCharge string            `json:"additional_charge"`
	Priority         int               `json:"priority"`
	Description      string            `json:"description,omitempty"`
	EffectiveFrom    string            `json:"effective_from"`
	EffectiveTo      string            `json:"effective_to,omitempty"`
	Active           bool              `json:"active"`
	Benefits         []BenefitResponse `json:"benefits,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type BenefitResponse struct {
	ID           int64       `json:"id"`
	PromotionID  int64       `json:"promotion_id"`
	BenefitType  string      `json:"benefit_type"`
	Package      *PackageRef `json:"package,omitempty"`
	DurationDays *int64      `json:"duration_days,omitempty"`
	Quantity     *int64      `json:"quantity,omitempty"`
	Description  string      `json:"description,omitempty"`
	Metadata     any         `json:"metadata_json,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type PackageListResponse struct {
	Items      []PackageResponse `json:"items"`
	Pagination PaginationMeta    `json:"pagination"`
}

type PlanListResponse struct {
	Items      []PlanResponse `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type PromotionListResponse struct {
	Items      []PromotionResponse `json:"items"`
	Pagination PaginationMeta      `json:"pagination"`
}

type EligiblePromotionResponse struct {
	Items       []PromotionResponse `json:"items"`
	Recommended *PromotionResponse  `json:"recommended_promotion,omitempty"`
}

type EligiblePlanResponse struct {
	Items []PlanResponse `json:"items"`
}

type BulkActionResponse struct {
	IDs      []int64 `json:"ids"`
	Affected int64   `json:"affected"`
}

type CreatePackageRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	LevelOrder  int    `json:"level_order" binding:"required"`
	Description string `json:"description"`
	Active      *bool  `json:"active"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

type UpdatePackageRequest struct {
	Code        *string `json:"code"`
	Name        *string `json:"name"`
	LevelOrder  *int    `json:"level_order"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
}

type CreatePlanRequest struct {
	PackageID     int64   `json:"package_id" binding:"required,min=1"`
	Code          string  `json:"code" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	TenureMonths  int     `json:"tenure_months" binding:"required,min=1"`
	Price         string  `json:"price" binding:"required"`
	Currency      string  `json:"currency"`
	EffectiveFrom string  `json:"effective_from" binding:"required"`
	EffectiveTo   *string `json:"effective_to"`
	Active        *bool   `json:"active"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
}

type UpdatePlanRequest struct {
	PackageID     *int64  `json:"package_id"`
	Code          *string `json:"code"`
	Name          *string `json:"name"`
	TenureMonths  *int    `json:"tenure_months"`
	Price         *string `json:"price"`
	Currency      *string `json:"currency"`
	EffectiveFrom *string `json:"effective_from"`
	EffectiveTo   *string `json:"effective_to"`
	Active        *bool   `json:"active"`
}

type CreatePromotionRequest struct {
	Code             string  `json:"code" binding:"required"`
	Name             string  `json:"name" binding:"required"`
	PromotionType    string  `json:"promotion_type" binding:"required"`
	ChargeType       string  `json:"charge_type" binding:"required"`
	AdditionalCharge string  `json:"additional_charge"`
	Priority         int     `json:"priority"`
	Description      string  `json:"description"`
	EffectiveFrom    string  `json:"effective_from" binding:"required"`
	EffectiveTo      *string `json:"effective_to"`
	Active           *bool   `json:"active"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
}

type UpdatePromotionRequest struct {
	Code             *string `json:"code"`
	Name             *string `json:"name"`
	PromotionType    *string `json:"promotion_type"`
	ChargeType       *string `json:"charge_type"`
	AdditionalCharge *string `json:"additional_charge"`
	Priority         *int    `json:"priority"`
	Description      *string `json:"description"`
	EffectiveFrom    *string `json:"effective_from"`
	EffectiveTo      *string `json:"effective_to"`
	Active           *bool   `json:"active"`
}

type CreateBenefitRequest struct {
	BenefitType  string          `json:"benefit_type" binding:"required"`
	PackageID    *int64          `json:"package_id"`
	DurationDays *int64          `json:"duration_days"`
	Quantity     *int64          `json:"quantity"`
	Description  string          `json:"description"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
}

type SetEligibilityRequest struct {
	PlanIDs []int64 `json:"plan_ids" binding:"required,min=1,dive,min=1"`
}

type BulkIDRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,dive,min=1"`
}

type ListParams struct {
	Query      string
	PackageID  *int64
	Active     *bool
	ChargeType string
	Scope      string
	AsOf       *time.Time
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	All        bool
	Page       int
	Limit      int
	Sort       string
}

func NewPackageResponse(item Package) PackageResponse {
	return PackageResponse{
		ID:          item.ID,
		Code:        item.Code,
		Name:        item.Name,
		LevelOrder:  item.LevelOrder,
		Description: item.Description.String,
		Active:      item.Active,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func NewPlanResponse(item Plan) PlanResponse {
	return PlanResponse{
		ID:            item.ID,
		Package:       PackageRef{ID: item.PackageID, Code: item.PackageCode, Name: item.PackageName},
		Code:          item.Code,
		Name:          item.Name,
		TenureMonths:  item.TenureMonths,
		DurationDays:  item.DurationDays,
		Price:         item.Price,
		Currency:      item.Currency,
		EffectiveFrom: formatDate(item.EffectiveFrom),
		EffectiveTo:   formatNullDate(item.EffectiveTo),
		Active:        item.Active,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func NewPromotionResponse(item Promotion) PromotionResponse {
	benefits := make([]BenefitResponse, 0, len(item.Benefits))
	for _, benefit := range item.Benefits {
		benefits = append(benefits, NewBenefitResponse(benefit))
	}
	return PromotionResponse{
		ID:               item.ID,
		Code:             item.Code,
		Name:             item.Name,
		PromotionType:    item.PromotionType,
		ChargeType:       item.ChargeType,
		AdditionalCharge: item.AdditionalCharge,
		Priority:         item.Priority,
		Description:      item.Description.String,
		EffectiveFrom:    formatDate(item.EffectiveFrom),
		EffectiveTo:      formatNullDate(item.EffectiveTo),
		Active:           item.Active,
		Benefits:         benefits,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func NewBenefitResponse(item Benefit) BenefitResponse {
	var pkg *PackageRef
	if item.PackageID.Valid {
		pkg = &PackageRef{ID: item.PackageID.Int64, Code: item.PackageCode.String, Name: item.PackageName.String}
	}
	var metadata any
	if item.MetadataJSON.Valid && item.MetadataJSON.String != "" {
		_ = json.Unmarshal([]byte(item.MetadataJSON.String), &metadata)
	}
	return BenefitResponse{
		ID:           item.ID,
		PromotionID:  item.PromotionID,
		BenefitType:  item.BenefitType,
		Package:      pkg,
		DurationDays: nullableInt64Ptr(item.DurationDays),
		Quantity:     nullableInt64Ptr(item.Quantity),
		Description:  item.Description.String,
		Metadata:     metadata,
		CreatedAt:    item.CreatedAt,
	}
}

func formatDate(value time.Time) string {
	return value.Format("2006-01-02")
}

func formatNullDate(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	out := value.Int64
	return &out
}
