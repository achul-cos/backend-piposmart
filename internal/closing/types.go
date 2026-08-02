package closing

import (
	"database/sql"
	"encoding/json"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	StatusPending   = "PENDING_RECONCILIATION"
	StatusConfirmed = "CONFIRMED"
	StatusRejected  = "REJECTED"

	StageClosing = "CLOSING"
	StatusOpen   = "OPEN"

	InteractionCall = "CALL"
	InteractionChat = "CHAT"

	ScopeActive  = "ACTIVE"
	ScopeDeleted = "DELETED"
	ScopeAll     = "ALL"
)

type LeadState struct {
	ID                 int64
	OwnerID            sql.NullInt64
	OutletID           sql.NullInt64
	ActiveSalesID      sql.NullInt64
	CurrentOwnerUserID sql.NullInt64
	CurrentOwnerRole   string
	SupervisorID       sql.NullInt64
	Stage              string
	Status             string
	CurrentScore       sql.NullInt64
}

type RemarkReason struct {
	ID    int64
	Score int64
	Code  string
	Label string
}

type PlanSnapshot struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	TenureMonths  int    `json:"tenure_months"`
	DurationDays  int    `json:"duration_days"`
	Price         string `json:"price"`
	Currency      string `json:"currency"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
}

type PackageSnapshot struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	LevelOrder int    `json:"level_order"`
}

type PromotionSnapshot struct {
	ID               int64             `json:"id"`
	Code             string            `json:"code"`
	Name             string            `json:"name"`
	PromotionType    string            `json:"promotion_type"`
	ChargeType       string            `json:"charge_type"`
	AdditionalCharge string            `json:"additional_charge"`
	Priority         int               `json:"priority"`
	EffectiveFrom    string            `json:"effective_from"`
	EffectiveTo      string            `json:"effective_to,omitempty"`
	Benefits         []BenefitSnapshot `json:"benefits,omitempty"`
}

type BenefitSnapshot struct {
	ID           int64  `json:"id"`
	BenefitType  string `json:"benefit_type"`
	PackageID    *int64 `json:"package_id,omitempty"`
	PackageCode  string `json:"package_code,omitempty"`
	PackageName  string `json:"package_name,omitempty"`
	DurationDays *int64 `json:"duration_days,omitempty"`
	Quantity     *int64 `json:"quantity,omitempty"`
	Description  string `json:"description,omitempty"`
	MetadataJSON any    `json:"metadata_json,omitempty"`
}

type Closing struct {
	ID                    int64
	Code                  string
	LeadID                sql.NullInt64
	LeadCode              sql.NullString
	OwnerID               sql.NullInt64
	OwnerCode             sql.NullString
	OwnerName             sql.NullString
	OutletID              sql.NullInt64
	SalesID               sql.NullInt64
	SalesName             sql.NullString
	SupervisorID          sql.NullInt64
	SupervisorName        sql.NullString
	PackageID             sql.NullInt64
	PackageCode           sql.NullString
	PackageName           sql.NullString
	PlanID                sql.NullInt64
	PlanCode              sql.NullString
	PlanName              sql.NullString
	PromotionID           sql.NullInt64
	PromotionCode         sql.NullString
	PromotionName         sql.NullString
	PackageSnapshotJSON   string
	PlanSnapshotJSON      string
	PromotionSnapshotJSON sql.NullString
	TenureMonths          int
	DurationDays          int
	BasePrice             string
	DiscountAmount        string
	AdditionalCharge      string
	UniqueTransferCode    int
	FinalAmount           string
	Currency              string
	Status                string
	Note                  sql.NullString
	RejectionReason       sql.NullString
	ClosedAt              time.Time
	ConfirmedAt           sql.NullTime
	RejectedAt            sql.NullTime
	CreatedByUserID       sql.NullInt64
	CreatedByName         sql.NullString
	UpdatedByUserID       sql.NullInt64
	UpdatedByName         sql.NullString
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type UserBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

type EntityRef struct {
	ID   int64  `json:"id"`
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

type ClosingResponse struct {
	ID                 int64              `json:"id"`
	Code               string             `json:"code"`
	Lead               *EntityRef         `json:"lead,omitempty"`
	Owner              *EntityRef         `json:"owner,omitempty"`
	OutletID           *int64             `json:"outlet_id,omitempty"`
	Sales              *UserBrief         `json:"sales,omitempty"`
	Supervisor         *UserBrief         `json:"supervisor,omitempty"`
	Package            *EntityRef         `json:"package,omitempty"`
	Plan               *EntityRef         `json:"plan,omitempty"`
	// Promotion/PromotionSnapshot are the FIRST applied promotion, kept for any old consumer of
	// this single field. Promotions is the full stacked list (Sprint 15a §4b) — always use this
	// one for anything beyond simple display, since a closing can now carry more than one.
	Promotion          *EntityRef         `json:"promotion,omitempty"`
	Promotions         []EntityRef        `json:"promotions,omitempty"`
	PackageSnapshot    PackageSnapshot    `json:"package_snapshot"`
	PlanSnapshot       PlanSnapshot       `json:"plan_snapshot"`
	PromotionSnapshot  *PromotionSnapshot `json:"promotion_snapshot,omitempty"`
	TenureMonths       int                `json:"tenure_months"`
	DurationDays       int                `json:"duration_days"`
	BasePrice          string             `json:"base_price"`
	DiscountAmount     string             `json:"discount_amount"`
	AdditionalCharge   string             `json:"additional_charge"`
	UniqueTransferCode int                `json:"unique_transfer_code"`
	FinalAmount        string             `json:"final_amount"`
	Currency           string             `json:"currency"`
	Status             string             `json:"status"`
	Note               string             `json:"note,omitempty"`
	RejectionReason    string             `json:"rejection_reason,omitempty"`
	ClosedAt           time.Time          `json:"closed_at"`
	ConfirmedAt        *time.Time         `json:"confirmed_at,omitempty"`
	RejectedAt         *time.Time         `json:"rejected_at,omitempty"`
	CreatedBy          *UserBrief         `json:"created_by,omitempty"`
	UpdatedBy          *UserBrief         `json:"updated_by,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type ClosingListResponse struct {
	Items      []ClosingResponse `json:"items"`
	Pagination PaginationMeta    `json:"pagination"`
}

type CreateClosingRequest struct {
	PlanID      int64  `json:"plan_id" binding:"required,min=1"`
	PromotionID *int64 `json:"promotion_id"`
	// PromotionIDs stacks multiple promotions on the same plan (Sprint 15a §4b) — if set, it's
	// used instead of PromotionID (which stays for backward compat with older single-promotion
	// clients). Every ID must be eligible for PlanID or the whole request is rejected.
	PromotionIDs       []int64    `json:"promotion_ids"`
	DiscountAmount     string     `json:"discount_amount"`
	UniqueTransferCode *int       `json:"unique_transfer_code"`
	ClosedAt           *time.Time `json:"closed_at"`
	InteractionType    string     `json:"interaction_type"`
	ContactName        string     `json:"contact_name"`
	ContactPhone       string     `json:"contact_phone"`
	CustomerResponse   string     `json:"customer_response"`
	Note               string     `json:"note"`
}

type UpdateClosingStatusRequest struct {
	Note   string `json:"note"`
	Reason string `json:"reason"`
}

type BulkIDRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,dive,min=1"`
}

type BulkActionResponse struct {
	IDs      []int64 `json:"ids"`
	Affected int64   `json:"affected"`
}

type ListParams struct {
	Query        string
	Status       string
	LeadID       *int64
	OwnerID      *int64
	SalesID      *int64
	SupervisorID *int64
	PlanID       *int64
	Scope        string
	ClosedFrom   *time.Time
	ClosedTo     *time.Time
	All          bool
	Page         int
	Limit        int
	Sort         string
}

func NewClosingResponse(item Closing) ClosingResponse {
	var packageSnapshot PackageSnapshot
	_ = json.Unmarshal([]byte(item.PackageSnapshotJSON), &packageSnapshot)
	var planSnapshot PlanSnapshot
	_ = json.Unmarshal([]byte(item.PlanSnapshotJSON), &planSnapshot)
	var promotionSnapshot *PromotionSnapshot
	if item.PromotionSnapshotJSON.Valid && item.PromotionSnapshotJSON.String != "" {
		var value PromotionSnapshot
		if err := json.Unmarshal([]byte(item.PromotionSnapshotJSON.String), &value); err == nil {
			promotionSnapshot = &value
		}
	}
	return ClosingResponse{
		ID:                 item.ID,
		Code:               item.Code,
		Lead:               nullableEntity(item.LeadID, item.LeadCode, sql.NullString{}),
		Owner:              nullableEntity(item.OwnerID, item.OwnerCode, item.OwnerName),
		OutletID:           nullableInt64Ptr(item.OutletID),
		Sales:              nullableUser(item.SalesID, item.SalesName, RoleSales),
		Supervisor:         nullableUser(item.SupervisorID, item.SupervisorName, RoleSupervisor),
		Package:            nullableEntity(item.PackageID, item.PackageCode, item.PackageName),
		Plan:               nullableEntity(item.PlanID, item.PlanCode, item.PlanName),
		Promotion:          nullableEntity(item.PromotionID, item.PromotionCode, item.PromotionName),
		PackageSnapshot:    packageSnapshot,
		PlanSnapshot:       planSnapshot,
		PromotionSnapshot:  promotionSnapshot,
		TenureMonths:       item.TenureMonths,
		DurationDays:       item.DurationDays,
		BasePrice:          item.BasePrice,
		DiscountAmount:     item.DiscountAmount,
		AdditionalCharge:   item.AdditionalCharge,
		UniqueTransferCode: item.UniqueTransferCode,
		FinalAmount:        item.FinalAmount,
		Currency:           item.Currency,
		Status:             item.Status,
		Note:               item.Note.String,
		RejectionReason:    item.RejectionReason.String,
		ClosedAt:           item.ClosedAt,
		ConfirmedAt:        nullableTimePtr(item.ConfirmedAt),
		RejectedAt:         nullableTimePtr(item.RejectedAt),
		CreatedBy:          nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		UpdatedBy:          nullableUser(item.UpdatedByUserID, item.UpdatedByName, ""),
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func nullableEntity(id sql.NullInt64, code sql.NullString, name sql.NullString) *EntityRef {
	if !id.Valid {
		return nil
	}
	return &EntityRef{ID: id.Int64, Code: code.String, Name: name.String}
}

func nullableUser(id sql.NullInt64, name sql.NullString, role string) *UserBrief {
	if !id.Valid {
		return nil
	}
	return &UserBrief{ID: id.Int64, Name: name.String, Role: role}
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	out := value.Int64
	return &out
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time
	return &out
}
