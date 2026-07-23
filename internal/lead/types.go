package lead

import (
	"database/sql"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	StageNew       = "NEW"
	StagePossible  = "POSSIBLE"
	StagePotential = "POTENTIAL"
	StageInvalid   = "INVALID"

	StatusOpen    = "OPEN"
	StatusInvalid = "INVALID"

	ActionCreated              = "CREATED_BY_ADMIN"
	ActionAssignedSupervisor   = "ASSIGNED_TO_SUPERVISOR"
	ActionAssignedSales        = "ASSIGNED_TO_SALES"
	ActionRedistributedSales   = "REDISTRIBUTED_TO_SALES"
	ActionReleasedAdmin        = "RELEASED_TO_ADMIN"
	ActionInvalidatedBySales   = "INVALIDATED_BY_SALES"
	ActionCreatedExistingOwner = "CREATED_FROM_EXISTING_OWNER"
)

type Lead struct {
	ID                   int64
	Code                 string
	OwnerID              sql.NullInt64
	OwnerCode            sql.NullString
	OwnerName            sql.NullString
	OwnerPhone           sql.NullString
	OwnerBrandName       sql.NullString
	OwnerProvince        sql.NullString
	OwnerCity            sql.NullString
	OutletID             sql.NullInt64
	ActiveSalesID        sql.NullInt64
	ActiveSalesName      sql.NullString
	CurrentOwnerUserID   sql.NullInt64
	CurrentOwnerUserName sql.NullString
	CurrentOwnerRole     string
	SupervisorID         sql.NullInt64
	SupervisorName       sql.NullString
	SourceType           string
	SourceReference      sql.NullString
	Stage                string
	Status               string
	CurrentScore         sql.NullInt64
	LastInteractionAt    sql.NullTime
	NextFollowUpAt       sql.NullTime
	InvalidatedAt        sql.NullTime
	InvalidatedBySalesID sql.NullInt64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type AssignmentHistory struct {
	ID             int64
	LeadID         int64
	OwnerID        sql.NullInt64
	FromUserID     sql.NullInt64
	FromUserName   sql.NullString
	FromRole       sql.NullString
	ToUserID       sql.NullInt64
	ToUserName     sql.NullString
	ToRole         string
	SupervisorID   sql.NullInt64
	SupervisorName sql.NullString
	AssignedByID   sql.NullInt64
	AssignedByName sql.NullString
	Action         string
	Reason         sql.NullString
	Score          sql.NullInt64
	Active         bool
	StartedAt      time.Time
	EndedAt        sql.NullTime
	CreatedAt      time.Time
}

type UserSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type LeadOwnerResponse struct {
	Available bool   `json:"available"`
	ID        *int64 `json:"id,omitempty"`
	Code      string `json:"code,omitempty"`
	Name      string `json:"name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	BrandName string `json:"brand_name,omitempty"`
	Province  string `json:"province,omitempty"`
	City      string `json:"city,omitempty"`
	Message   string `json:"message,omitempty"`
}

type LeadResponse struct {
	ID                int64             `json:"id"`
	Code              string            `json:"code"`
	Owner             LeadOwnerResponse `json:"owner"`
	OutletID          *int64            `json:"outlet_id,omitempty"`
	CurrentOwner      *UserSummary      `json:"current_owner,omitempty"`
	CurrentOwnerRole  string            `json:"current_owner_role"`
	Supervisor        *UserSummary      `json:"supervisor,omitempty"`
	ActiveSales       *UserSummary      `json:"active_sales,omitempty"`
	SourceType        string            `json:"source_type"`
	SourceReference   string            `json:"source_reference,omitempty"`
	Stage             string            `json:"stage"`
	Status            string            `json:"status"`
	CurrentScore      *int64            `json:"current_score,omitempty"`
	LastInteractionAt *time.Time        `json:"last_interaction_at,omitempty"`
	NextFollowUpAt    *time.Time        `json:"next_follow_up_at,omitempty"`
	InvalidatedAt     *time.Time        `json:"invalidated_at,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type AssignmentHistoryResponse struct {
	ID         int64        `json:"id"`
	LeadID     int64        `json:"lead_id"`
	OwnerID    *int64       `json:"owner_id,omitempty"`
	FromUser   *UserSummary `json:"from_user,omitempty"`
	ToUser     *UserSummary `json:"to_user,omitempty"`
	Supervisor *UserSummary `json:"supervisor,omitempty"`
	AssignedBy *UserSummary `json:"assigned_by,omitempty"`
	Action     string       `json:"action"`
	Reason     string       `json:"reason,omitempty"`
	Score      *int64       `json:"score,omitempty"`
	Active     bool         `json:"active"`
	StartedAt  time.Time    `json:"started_at"`
	EndedAt    *time.Time   `json:"ended_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type LeadListResponse struct {
	Items      []LeadResponse `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type LeadBulkResponse struct {
	Items []LeadResponse `json:"items"`
	Total int            `json:"total"`
}

type AssignmentHistoryListResponse struct {
	Items []AssignmentHistoryResponse `json:"items"`
	Total int                         `json:"total"`
}

type CreateLeadRequest struct {
	OwnerID         int64  `json:"owner_id" binding:"required,min=1"`
	OutletID        *int64 `json:"outlet_id"`
	SourceType      string `json:"source_type"`
	SourceReference string `json:"source_reference"`
}

type AssignSupervisorRequest struct {
	SupervisorID int64  `json:"supervisor_id" binding:"required,min=1"`
	Reason       string `json:"reason"`
}

type BulkAssignSupervisorRequest struct {
	LeadIDs      []int64 `json:"lead_ids" binding:"required,min=1,dive,min=1"`
	SupervisorID int64   `json:"supervisor_id" binding:"required,min=1"`
	Reason       string  `json:"reason"`
}

type AssignSalesRequest struct {
	SalesID int64  `json:"sales_id" binding:"required,min=1"`
	Reason  string `json:"reason"`
}

type BulkAssignSalesRequest struct {
	LeadIDs []int64 `json:"lead_ids" binding:"required,min=1,dive,min=1"`
	SalesID int64   `json:"sales_id" binding:"required,min=1"`
	Reason  string  `json:"reason"`
}

type ReleaseRequest struct {
	AdminID *int64 `json:"admin_id"`
	Reason  string `json:"reason"`
}

type BulkReleaseRequest struct {
	LeadIDs []int64 `json:"lead_ids" binding:"required,min=1,dive,min=1"`
	AdminID *int64  `json:"admin_id"`
	Reason  string  `json:"reason"`
}

type MarkInvalidRequest struct {
	Reason string `json:"reason"`
}

type ListParams struct {
	Query        string
	Ownership    string
	Stage        string
	Status       string
	Score        *int64
	SupervisorID *int64
	SalesID      *int64
	FollowUpFrom *time.Time
	FollowUpTo   *time.Time
	Page         int
	Limit        int
	Sort         string
}

func NewLeadResponse(item Lead) LeadResponse {
	var ownerID *int64
	if item.OwnerID.Valid {
		value := item.OwnerID.Int64
		ownerID = &value
	}
	owner := LeadOwnerResponse{
		Available: item.OwnerID.Valid,
		ID:        ownerID,
		Code:      item.OwnerCode.String,
		Name:      item.OwnerName.String,
		Phone:     item.OwnerPhone.String,
		BrandName: item.OwnerBrandName.String,
		Province:  item.OwnerProvince.String,
		City:      item.OwnerCity.String,
	}
	if !owner.Available {
		owner.Message = "data owner tidak tersedia"
	}
	var outletID *int64
	if item.OutletID.Valid {
		value := item.OutletID.Int64
		outletID = &value
	}
	var score *int64
	if item.CurrentScore.Valid {
		value := item.CurrentScore.Int64
		score = &value
	}
	return LeadResponse{
		ID:                item.ID,
		Code:              item.Code,
		Owner:             owner,
		OutletID:          outletID,
		CurrentOwner:      nullableUser(item.CurrentOwnerUserID, item.CurrentOwnerUserName, item.CurrentOwnerRole),
		CurrentOwnerRole:  item.CurrentOwnerRole,
		Supervisor:        nullableUser(item.SupervisorID, item.SupervisorName, RoleSupervisor),
		ActiveSales:       nullableUser(item.ActiveSalesID, item.ActiveSalesName, RoleSales),
		SourceType:        item.SourceType,
		SourceReference:   item.SourceReference.String,
		Stage:             item.Stage,
		Status:            item.Status,
		CurrentScore:      score,
		LastInteractionAt: nullableTimePtr(item.LastInteractionAt),
		NextFollowUpAt:    nullableTimePtr(item.NextFollowUpAt),
		InvalidatedAt:     nullableTimePtr(item.InvalidatedAt),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func NewAssignmentHistoryResponse(item AssignmentHistory) AssignmentHistoryResponse {
	var ownerID *int64
	if item.OwnerID.Valid {
		value := item.OwnerID.Int64
		ownerID = &value
	}
	var score *int64
	if item.Score.Valid {
		value := item.Score.Int64
		score = &value
	}
	return AssignmentHistoryResponse{
		ID:         item.ID,
		LeadID:     item.LeadID,
		OwnerID:    ownerID,
		FromUser:   nullableUser(item.FromUserID, item.FromUserName, item.FromRole.String),
		ToUser:     nullableUser(item.ToUserID, item.ToUserName, item.ToRole),
		Supervisor: nullableUser(item.SupervisorID, item.SupervisorName, RoleSupervisor),
		AssignedBy: nullableUser(item.AssignedByID, item.AssignedByName, ""),
		Action:     item.Action,
		Reason:     item.Reason.String,
		Score:      score,
		Active:     item.Active,
		StartedAt:  item.StartedAt,
		EndedAt:    nullableTimePtr(item.EndedAt),
		CreatedAt:  item.CreatedAt,
	}
}

func nullableUser(id sql.NullInt64, name sql.NullString, role string) *UserSummary {
	if !id.Valid {
		return nil
	}
	return &UserSummary{ID: id.Int64, Name: name.String, Role: role}
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
