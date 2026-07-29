package subscription

import (
	"database/sql"
	"encoding/json"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	OrderStatusPaid       = "PAID"
	OrderStatusReconciled = "RECONCILED"
	OrderStatusRejected   = "REJECTED"
	OrderStatusCanceled   = "CANCELED"

	SubscriptionStatusActive   = "ACTIVE"
	SubscriptionStatusExpired  = "EXPIRED"
	SubscriptionStatusCanceled = "CANCELED"

	ReconciliationStatusPending   = "PENDING"
	ReconciliationStatusConfirmed = "CONFIRMED"
	ReconciliationStatusRejected  = "REJECTED"

	ReconciliationMatchAuto   = "AUTO"
	ReconciliationMatchManual = "MANUAL"

	IssueStatusOpen      = "OPEN"
	IssueStatusResolved  = "RESOLVED"
	IssueStatusDismissed = "DISMISSED"

	IssueHangingOrder   = "HANGING_ORDER"
	IssueAmountMismatch = "AMOUNT_MISMATCH"
	IssueOwnerMismatch  = "OWNER_MISMATCH"

	WalletDirectionDebit          = "DEBIT"
	WalletTransactionDebit        = "DEBIT"
	WalletSourceSubscriptionOrder = "SUBSCRIPTION_ORDER"
)

type EntityRef struct {
	ID   int64  `json:"id"`
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

type UserBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

type PackageSnapshot struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	LevelOrder int    `json:"level_order"`
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

type SubscriptionOrder struct {
	ID                    int64
	Code                  string
	OwnerID               sql.NullInt64
	OwnerCode             sql.NullString
	OwnerName             sql.NullString
	OutletID              sql.NullInt64
	ClosingID             sql.NullInt64
	ClosingCode           sql.NullString
	SalesID               sql.NullInt64
	SalesName             sql.NullString
	SupervisorID          sql.NullInt64
	SupervisorName        sql.NullString
	WalletAccountID       sql.NullInt64
	WalletTransactionID   sql.NullInt64
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
	FinalAmount           string
	Currency              string
	Status                string
	IdempotencyKey        string
	ExternalReference     sql.NullString
	PurchasedAt           time.Time
	SubscriptionStartDate time.Time
	Note                  sql.NullString
	CreatedByUserID       sql.NullInt64
	CreatedByName         sql.NullString
	UpdatedByUserID       sql.NullInt64
	UpdatedByName         sql.NullString
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Subscription struct {
	ID                int64
	Code              string
	OwnerID           sql.NullInt64
	OwnerCode         sql.NullString
	OwnerName         sql.NullString
	OutletID          sql.NullInt64
	OrderID           sql.NullInt64
	OrderCode         sql.NullString
	PackageID         sql.NullInt64
	PackageCode       sql.NullString
	PackageName       sql.NullString
	PlanID            sql.NullInt64
	PlanCode          sql.NullString
	PlanName          sql.NullString
	Status            string
	ActiveFrom        time.Time
	ActiveUntil       time.Time
	TotalDurationDays int
	SourceType        string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SubscriptionPeriod struct {
	ID             int64
	SubscriptionID sql.NullInt64
	OrderID        sql.NullInt64
	OwnerID        sql.NullInt64
	PeriodIndex    int
	StartDate      time.Time
	EndDate        time.Time
	DurationDays   int
	PackageID      sql.NullInt64
	PlanID         sql.NullInt64
	PromotionID    sql.NullInt64
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Reconciliation struct {
	ID               int64
	Code             string
	OrderID          sql.NullInt64
	OrderCode        sql.NullString
	ClosingID        sql.NullInt64
	ClosingCode      sql.NullString
	OwnerID          sql.NullInt64
	OwnerCode        sql.NullString
	OwnerName        sql.NullString
	Status           string
	MatchType        string
	IssueCode        sql.NullString
	AmountDifference string
	Note             sql.NullString
	Reason           sql.NullString
	ConfirmedAt      sql.NullTime
	RejectedAt       sql.NullTime
	CreatedByUserID  sql.NullInt64
	CreatedByName    sql.NullString
	UpdatedByUserID  sql.NullInt64
	UpdatedByName    sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ReconciliationIssue struct {
	ID              int64
	Code            string
	OrderID         sql.NullInt64
	OrderCode       sql.NullString
	ClosingID       sql.NullInt64
	ClosingCode     sql.NullString
	OwnerID         sql.NullInt64
	OwnerCode       sql.NullString
	OwnerName       sql.NullString
	IssueType       string
	Status          string
	Description     sql.NullString
	DetectedAt      time.Time
	ResolvedAt      sql.NullTime
	CreatedByUserID sql.NullInt64
	CreatedByName   sql.NullString
	UpdatedByUserID sql.NullInt64
	UpdatedByName   sql.NullString
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateOrderRequest struct {
	PlanID                int64      `json:"plan_id"`
	OutletID              *int64     `json:"outlet_id"`
	PromotionID           *int64     `json:"promotion_id"`
	ClosingID             *int64     `json:"closing_id"`
	DiscountAmount        string     `json:"discount_amount"`
	IdempotencyKey        string     `json:"idempotency_key"`
	ExternalReference     string     `json:"external_reference"`
	PurchasedAt           *time.Time `json:"purchased_at"`
	SubscriptionStartDate string     `json:"subscription_start_date"`
	Note                  string     `json:"note"`
}

type ManualReconcileRequest struct {
	ClosingID int64  `json:"closing_id" binding:"required,min=1"`
	Action    string `json:"action" binding:"required"`
	Note      string `json:"note"`
	Reason    string `json:"reason"`
}

type ListParams struct {
	Query         string
	Status        string
	OwnerID       *int64
	OutletID      *int64
	OrderID       *int64
	ClosingID     *int64
	SalesID       *int64
	SupervisorID  *int64
	PlanID        *int64
	IssueType     string
	PurchasedFrom *time.Time
	PurchasedTo   *time.Time
	ActiveFrom    *time.Time
	ActiveTo      *time.Time
	All           bool
	Page          int
	Limit         int
	Sort          string
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type SubscriptionOrderResponse struct {
	ID                    int64              `json:"id"`
	Code                  string             `json:"code"`
	Owner                 *EntityRef         `json:"owner,omitempty"`
	OutletID              *int64             `json:"outlet_id,omitempty"`
	Closing               *EntityRef         `json:"closing,omitempty"`
	Sales                 *UserBrief         `json:"sales,omitempty"`
	Supervisor            *UserBrief         `json:"supervisor,omitempty"`
	WalletAccountID       *int64             `json:"wallet_account_id,omitempty"`
	WalletTransactionID   *int64             `json:"wallet_transaction_id,omitempty"`
	Package               *EntityRef         `json:"package,omitempty"`
	Plan                  *EntityRef         `json:"plan,omitempty"`
	Promotion             *EntityRef         `json:"promotion,omitempty"`
	PackageSnapshot       PackageSnapshot    `json:"package_snapshot"`
	PlanSnapshot          PlanSnapshot       `json:"plan_snapshot"`
	PromotionSnapshot     *PromotionSnapshot `json:"promotion_snapshot,omitempty"`
	TenureMonths          int                `json:"tenure_months"`
	DurationDays          int                `json:"duration_days"`
	BasePrice             string             `json:"base_price"`
	DiscountAmount        string             `json:"discount_amount"`
	AdditionalCharge      string             `json:"additional_charge"`
	FinalAmount           string             `json:"final_amount"`
	Currency              string             `json:"currency"`
	Status                string             `json:"status"`
	IdempotencyKey        string             `json:"idempotency_key"`
	ExternalReference     string             `json:"external_reference,omitempty"`
	PurchasedAt           time.Time          `json:"purchased_at"`
	SubscriptionStartDate string             `json:"subscription_start_date"`
	Note                  string             `json:"note,omitempty"`
	CreatedBy             *UserBrief         `json:"created_by,omitempty"`
	UpdatedBy             *UserBrief         `json:"updated_by,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
}

type SubscriptionResponse struct {
	ID                int64      `json:"id"`
	Code              string     `json:"code"`
	Owner             *EntityRef `json:"owner,omitempty"`
	OutletID          *int64     `json:"outlet_id,omitempty"`
	Order             *EntityRef `json:"order,omitempty"`
	Package           *EntityRef `json:"package,omitempty"`
	Plan              *EntityRef `json:"plan,omitempty"`
	Status            string     `json:"status"`
	ActiveFrom        string     `json:"active_from"`
	ActiveUntil       string     `json:"active_until"`
	TotalDurationDays int        `json:"total_duration_days"`
	SourceType        string     `json:"source_type"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type SubscriptionPeriodResponse struct {
	ID             int64  `json:"id"`
	SubscriptionID *int64 `json:"subscription_id,omitempty"`
	OrderID        *int64 `json:"order_id,omitempty"`
	OwnerID        *int64 `json:"owner_id,omitempty"`
	PeriodIndex    int    `json:"period_index"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	DurationDays   int    `json:"duration_days"`
	PackageID      *int64 `json:"package_id,omitempty"`
	PlanID         *int64 `json:"plan_id,omitempty"`
	PromotionID    *int64 `json:"promotion_id,omitempty"`
	Status         string `json:"status"`
}

type ReconciliationResponse struct {
	ID               int64      `json:"id"`
	Code             string     `json:"code"`
	Order            *EntityRef `json:"order,omitempty"`
	Closing          *EntityRef `json:"closing,omitempty"`
	Owner            *EntityRef `json:"owner,omitempty"`
	Status           string     `json:"status"`
	MatchType        string     `json:"match_type"`
	IssueCode        string     `json:"issue_code,omitempty"`
	AmountDifference string     `json:"amount_difference"`
	Note             string     `json:"note,omitempty"`
	Reason           string     `json:"reason,omitempty"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty"`
	RejectedAt       *time.Time `json:"rejected_at,omitempty"`
	CreatedBy        *UserBrief `json:"created_by,omitempty"`
	UpdatedBy        *UserBrief `json:"updated_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ReconciliationIssueResponse struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	Order       *EntityRef `json:"order,omitempty"`
	Closing     *EntityRef `json:"closing,omitempty"`
	Owner       *EntityRef `json:"owner,omitempty"`
	IssueType   string     `json:"issue_type"`
	Status      string     `json:"status"`
	Description string     `json:"description,omitempty"`
	DetectedAt  time.Time  `json:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	CreatedBy   *UserBrief `json:"created_by,omitempty"`
	UpdatedBy   *UserBrief `json:"updated_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type OrderCreateResponse struct {
	Order          SubscriptionOrderResponse    `json:"order"`
	Subscription   SubscriptionResponse         `json:"subscription"`
	Period         SubscriptionPeriodResponse   `json:"period"`
	Reconciliation *ReconciliationResponse      `json:"reconciliation,omitempty"`
	Issue          *ReconciliationIssueResponse `json:"issue,omitempty"`
	Idempotent     bool                         `json:"idempotent"`
}

type ReconciliationActionResponse struct {
	Order          SubscriptionOrderResponse    `json:"order"`
	Reconciliation ReconciliationResponse       `json:"reconciliation"`
	Issue          *ReconciliationIssueResponse `json:"issue,omitempty"`
}
type OrderListResponse struct {
	Items      []SubscriptionOrderResponse `json:"items"`
	Pagination PaginationMeta              `json:"pagination"`
}

type SubscriptionListResponse struct {
	Items      []SubscriptionResponse `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
}

type ReconciliationListResponse struct {
	Items      []ReconciliationResponse `json:"items"`
	Pagination PaginationMeta           `json:"pagination"`
}

type ReconciliationIssueListResponse struct {
	Items      []ReconciliationIssueResponse `json:"items"`
	Pagination PaginationMeta                `json:"pagination"`
}

func NewOrderResponse(item SubscriptionOrder) SubscriptionOrderResponse {
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
	return SubscriptionOrderResponse{
		ID:                    item.ID,
		Code:                  item.Code,
		Owner:                 nullableEntity(item.OwnerID, item.OwnerCode, item.OwnerName),
		OutletID:              nullableInt64Ptr(item.OutletID),
		Closing:               nullableEntity(item.ClosingID, item.ClosingCode, sql.NullString{}),
		Sales:                 nullableUser(item.SalesID, item.SalesName, RoleSales),
		Supervisor:            nullableUser(item.SupervisorID, item.SupervisorName, RoleSupervisor),
		WalletAccountID:       nullableInt64Ptr(item.WalletAccountID),
		WalletTransactionID:   nullableInt64Ptr(item.WalletTransactionID),
		Package:               nullableEntity(item.PackageID, item.PackageCode, item.PackageName),
		Plan:                  nullableEntity(item.PlanID, item.PlanCode, item.PlanName),
		Promotion:             nullableEntity(item.PromotionID, item.PromotionCode, item.PromotionName),
		PackageSnapshot:       packageSnapshot,
		PlanSnapshot:          planSnapshot,
		PromotionSnapshot:     promotionSnapshot,
		TenureMonths:          item.TenureMonths,
		DurationDays:          item.DurationDays,
		BasePrice:             item.BasePrice,
		DiscountAmount:        item.DiscountAmount,
		AdditionalCharge:      item.AdditionalCharge,
		FinalAmount:           item.FinalAmount,
		Currency:              item.Currency,
		Status:                item.Status,
		IdempotencyKey:        item.IdempotencyKey,
		ExternalReference:     item.ExternalReference.String,
		PurchasedAt:           item.PurchasedAt,
		SubscriptionStartDate: formatDate(item.SubscriptionStartDate),
		Note:                  item.Note.String,
		CreatedBy:             nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		UpdatedBy:             nullableUser(item.UpdatedByUserID, item.UpdatedByName, ""),
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func NewSubscriptionResponse(item Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:                item.ID,
		Code:              item.Code,
		Owner:             nullableEntity(item.OwnerID, item.OwnerCode, item.OwnerName),
		OutletID:          nullableInt64Ptr(item.OutletID),
		Order:             nullableEntity(item.OrderID, item.OrderCode, sql.NullString{}),
		Package:           nullableEntity(item.PackageID, item.PackageCode, item.PackageName),
		Plan:              nullableEntity(item.PlanID, item.PlanCode, item.PlanName),
		Status:            item.Status,
		ActiveFrom:        formatDate(item.ActiveFrom),
		ActiveUntil:       formatDate(item.ActiveUntil),
		TotalDurationDays: item.TotalDurationDays,
		SourceType:        item.SourceType,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func NewPeriodResponse(item SubscriptionPeriod) SubscriptionPeriodResponse {
	return SubscriptionPeriodResponse{
		ID:             item.ID,
		SubscriptionID: nullableInt64Ptr(item.SubscriptionID),
		OrderID:        nullableInt64Ptr(item.OrderID),
		OwnerID:        nullableInt64Ptr(item.OwnerID),
		PeriodIndex:    item.PeriodIndex,
		StartDate:      formatDate(item.StartDate),
		EndDate:        formatDate(item.EndDate),
		DurationDays:   item.DurationDays,
		PackageID:      nullableInt64Ptr(item.PackageID),
		PlanID:         nullableInt64Ptr(item.PlanID),
		PromotionID:    nullableInt64Ptr(item.PromotionID),
		Status:         item.Status,
	}
}

func NewReconciliationResponse(item Reconciliation) ReconciliationResponse {
	return ReconciliationResponse{
		ID:               item.ID,
		Code:             item.Code,
		Order:            nullableEntity(item.OrderID, item.OrderCode, sql.NullString{}),
		Closing:          nullableEntity(item.ClosingID, item.ClosingCode, sql.NullString{}),
		Owner:            nullableEntity(item.OwnerID, item.OwnerCode, item.OwnerName),
		Status:           item.Status,
		MatchType:        item.MatchType,
		IssueCode:        item.IssueCode.String,
		AmountDifference: item.AmountDifference,
		Note:             item.Note.String,
		Reason:           item.Reason.String,
		ConfirmedAt:      nullableTimePtr(item.ConfirmedAt),
		RejectedAt:       nullableTimePtr(item.RejectedAt),
		CreatedBy:        nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		UpdatedBy:        nullableUser(item.UpdatedByUserID, item.UpdatedByName, ""),
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func NewIssueResponse(item ReconciliationIssue) ReconciliationIssueResponse {
	return ReconciliationIssueResponse{
		ID:          item.ID,
		Code:        item.Code,
		Order:       nullableEntity(item.OrderID, item.OrderCode, sql.NullString{}),
		Closing:     nullableEntity(item.ClosingID, item.ClosingCode, sql.NullString{}),
		Owner:       nullableEntity(item.OwnerID, item.OwnerCode, item.OwnerName),
		IssueType:   item.IssueType,
		Status:      item.Status,
		Description: item.Description.String,
		DetectedAt:  item.DetectedAt,
		ResolvedAt:  nullableTimePtr(item.ResolvedAt),
		CreatedBy:   nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		UpdatedBy:   nullableUser(item.UpdatedByUserID, item.UpdatedByName, ""),
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
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

func formatDate(value time.Time) string {
	return value.Format("2006-01-02")
}
