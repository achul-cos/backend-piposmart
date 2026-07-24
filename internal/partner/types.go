package partner

import (
	"database/sql"
	"time"
)

const (
	// Partner status
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"

	// Partner account mask length
	AccountMaskLength = 4

	// User roles (mirrors internal/identity role codes)
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	// Partner commission status
	CommissionStatusPending   = "PENDING"
	CommissionStatusApproved  = "APPROVED"
	CommissionStatusPaid      = "PAID"
	CommissionStatusCancelled = "CANCELLED"

	// Partner type commission mode
	CommissionModePercentage = "PERCENTAGE"
	CommissionModeFixed      = "FIXED"
)

// PartnerType represents the type of partner (e.g., Supplier, Distributor, Agent).
// CommissionMode/CommissionValue define how much a partner of this type earns when a
// lead it referred reaches a CONFIRMED closing: PERCENTAGE applies CommissionValue as a
// percent of the closing's final_amount, FIXED pays CommissionValue flat regardless of
// the closing's size.
type PartnerType struct {
	ID              int64     `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	CommissionMode  string    `json:"commission_mode"`
	CommissionValue string    `json:"commission_value"`
	Description     string    `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Partner represents a business partner.
type Partner struct {
	ID                         int64          `json:"id"`
	PartnerTypeID              int64          `json:"partner_type_id"`
	PartnerTypeCode            sql.NullString `json:"partner_type_code,omitempty"`
	PartnerTypeName            sql.NullString `json:"partner_type_name,omitempty"`
	PartnerTypeCommissionMode  sql.NullString `json:"-"`
	PartnerTypeCommissionValue sql.NullString `json:"-"`
	Code                       string         `json:"code"`
	Name                       string         `json:"name"`
	Phone                      sql.NullString `json:"phone,omitempty"`
	Email                      sql.NullString `json:"email,omitempty"`
	Address                    sql.NullString `json:"address,omitempty"`
	BankAccountEncrypted       []byte         `json:"-"`                            // encrypted account number, never exposed
	BankAccountLast4           sql.NullString `json:"bank_account_last4,omitempty"` // last 4 digits for masking
	Status                     string         `json:"status"`
	CreatedAt                  time.Time      `json:"created_at"`
	UpdatedAt                  time.Time      `json:"updated_at"`
}

// PartnerAssignment represents the assignment of a PIC (Person In Charge) to a partner.
type PartnerAssignment struct {
	ID             int64          `json:"id"`
	PartnerID      int64          `json:"partner_id"`
	UserID         int64          `json:"user_id"` // PIC user
	UserName       sql.NullString `json:"user_name,omitempty"`
	UserRole       string         `json:"user_role,omitempty"` // e.g., SALES, SUPERVISOR
	AssignedByID   sql.NullInt64  `json:"assigned_by_id,omitempty"`
	AssignedByName sql.NullString `json:"assigned_by_name,omitempty"`
	AssignedAt     time.Time      `json:"assigned_at"`
	UnassignedAt   sql.NullTime   `json:"unassigned_at,omitempty"`
	Active         bool           `json:"active"` // true if currently active assignment
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// PartnerInteraction records a call or chat interaction with a partner.
type PartnerInteraction struct {
	ID              int64          `json:"id"`
	PartnerID       int64          `json:"partner_id"`
	InteractionType string         `json:"interaction_type"` // CALL or CHAT
	InteractionAt   time.Time      `json:"interaction_at"`
	Note            sql.NullString `json:"note,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

// PartnerReferral represents a referral from a partner to a lead.
type PartnerReferral struct {
	ID           int64          `json:"id"`
	PartnerID    int64          `json:"partner_id"`
	LeadID       int64          `json:"lead_id"`
	ReferralDate time.Time      `json:"referral_date"`
	Notes        sql.NullString `json:"notes,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	// Ensure uniqueness: one referral per partner-lead pair
}

// Response structs for API responses

type PartnerTypeResponse struct {
	ID              int64     `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	CommissionMode  string    `json:"commission_mode"`
	CommissionValue string    `json:"commission_value"`
	Description     string    `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PartnerResponse struct {
	ID                int64               `json:"id"`
	PartnerType       PartnerTypeResponse `json:"partner_type"`
	Code              string              `json:"code"`
	Name              string              `json:"name"`
	Phone             *string             `json:"phone,omitempty"`
	Email             *string             `json:"email,omitempty"`
	Address           *string             `json:"address,omitempty"`
	BankAccountMasked *string             `json:"bank_account_masked,omitempty"` // e.g., ****1234
	Status            string              `json:"status"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type PartnerAssignmentResponse struct {
	ID             int64      `json:"id"`
	PartnerID      int64      `json:"partner_id"`
	UserID         int64      `json:"user_id"`
	UserName       *string    `json:"user_name,omitempty"`
	UserRole       string     `json:"user_role,omitempty"`
	AssignedByID   *int64     `json:"assigned_by_id,omitempty"`
	AssignedByName *string    `json:"assigned_by_name,omitempty"`
	AssignedAt     time.Time  `json:"assigned_at"`
	UnassignedAt   *time.Time `json:"unassigned_at,omitempty"`
	Active         bool       `json:"active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type PartnerInteractionResponse struct {
	ID              int64     `json:"id"`
	PartnerID       int64     `json:"partner_id"`
	InteractionType string    `json:"interaction_type"`
	InteractionAt   time.Time `json:"interaction_at"`
	Note            *string   `json:"note,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type PartnerReferralResponse struct {
	ID           int64     `json:"id"`
	PartnerID    int64     `json:"partner_id"`
	LeadID       int64     `json:"lead_id"`
	ReferralDate time.Time `json:"referral_date"`
	Notes        *string   `json:"note,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Pagination metadata

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// List responses

type PartnerTypeListResponse struct {
	Items      []PartnerTypeResponse `json:"items"`
	Pagination PaginationMeta        `json:"pagination"`
}

type PartnerListResponse struct {
	Items      []PartnerResponse `json:"items"`
	Pagination PaginationMeta    `json:"pagination"`
}

type PartnerAssignmentListResponse struct {
	Items      []PartnerAssignmentResponse `json:"items"`
	Pagination PaginationMeta              `json:"pagination"`
}

type PartnerInteractionListResponse struct {
	Items      []PartnerInteractionResponse `json:"items"`
	Pagination PaginationMeta               `json:"pagination"`
}

type PartnerReferralListResponse struct {
	Items      []PartnerReferralResponse `json:"items"`
	Pagination PaginationMeta            `json:"pagination"`
}

// Request structs for creating/updating

type CreatePartnerTypeRequest struct {
	Code            string `json:"code" binding:"required,min=3"`
	Name            string `json:"name" binding:"required,min=3"`
	CommissionMode  string `json:"commission_mode" binding:"required,oneof=PERCENTAGE FIXED"`
	CommissionValue string `json:"commission_value" binding:"required"`
	Description     string `json:"description,omitempty"`
}

type UpdatePartnerTypeRequest struct {
	Name            string `json:"name,omitempty"`
	CommissionMode  string `json:"commission_mode,omitempty"`
	CommissionValue string `json:"commission_value,omitempty"`
	Description     string `json:"description,omitempty"`
}

type CreatePartnerRequest struct {
	PartnerTypeID int64   `json:"partner_type_id" binding:"required,min=1"`
	Code          string  `json:"code" binding:"required,min=3"`
	Name          string  `json:"name" binding:"required,min=3"`
	Phone         *string `json:"phone,omitempty"`
	Email         *string `json:"email,omitempty"`
	Address       *string `json:"address,omitempty"`
	BankAccount   *string `json:"bank_account,omitempty"` // plain account number, will be encrypted
	Status        string  `json:"status,omitempty"`       // default ACTIVE
}

type UpdatePartnerRequest struct {
	Name        *string `json:"name,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Email       *string `json:"email,omitempty"`
	Address     *string `json:"address,omitempty"`
	BankAccount *string `json:"bank_account,omitempty"` // if provided, re-encrypt
	Status      *string `json:"status,omitempty"`
}

type CreatePartnerAssignmentRequest struct {
	UserID       int64  `json:"user_id" binding:"required,min=1"` // PIC user
	AssignedByID *int64 `json:"assigned_by_id,omitempty"`         // who made assignment (optional, can be from auth)
	// Assumed active by default
}

type CreatePartnerInteractionRequest struct {
	InteractionType string     `json:"interaction_type" binding:"required,oneof=CALL CHAT"`
	InteractionAt   *time.Time `json:"interaction_at,omitempty"` // default now
	Note            *string    `json:"note,omitempty"`
}

type CreatePartnerReferralRequest struct {
	LeadID       int64      `json:"lead_id" binding:"required,min=1"`
	ReferralDate *time.Time `json:"referral_date,omitempty"` // default now
	Notes        *string    `json:"notes,omitempty"`
}

// Helper functions to build responses

func NewPartnerTypeResponse(pt PartnerType) PartnerTypeResponse {
	return PartnerTypeResponse{
		ID:              pt.ID,
		Code:            pt.Code,
		Name:            pt.Name,
		CommissionMode:  pt.CommissionMode,
		CommissionValue: pt.CommissionValue,
		Description:     pt.Description,
		CreatedAt:       pt.CreatedAt,
		UpdatedAt:       pt.UpdatedAt,
	}
}

func maskedAccountPtr(last4 sql.NullString) *string {
	if !last4.Valid || len(last4.String) == 0 {
		return nil
	}
	m := "****" + last4.String
	return &m
}

func NewPartnerResponse(p Partner) PartnerResponse {
	return PartnerResponse{
		ID: p.ID,
		PartnerType: NewPartnerTypeResponse(PartnerType{
			ID:              p.PartnerTypeID,
			Code:            p.PartnerTypeCode.String,
			Name:            p.PartnerTypeName.String,
			CommissionMode:  p.PartnerTypeCommissionMode.String,
			CommissionValue: p.PartnerTypeCommissionValue.String,
		}),
		Code:              p.Code,
		Name:              p.Name,
		Phone:             nullStringToPtr(p.Phone),
		Email:             nullStringToPtr(p.Email),
		Address:           nullStringToPtr(p.Address),
		BankAccountMasked: maskedAccountPtr(p.BankAccountLast4),
		Status:            p.Status,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

func NewPartnerAssignmentResponse(a PartnerAssignment) PartnerAssignmentResponse {
	var assignedByID *int64
	var assignedByName *string
	if a.AssignedByID.Valid {
		v := a.AssignedByID.Int64
		assignedByID = &v
	}
	if a.AssignedByName.Valid {
		v := a.AssignedByName.String
		assignedByName = &v
	}
	var unassignedAt *time.Time
	if a.UnassignedAt.Valid {
		u := a.UnassignedAt.Time
		unassignedAt = &u
	}
	var userName *string
	if a.UserName.Valid {
		u := a.UserName.String
		userName = &u
	}
	return PartnerAssignmentResponse{
		ID:             a.ID,
		PartnerID:      a.PartnerID,
		UserID:         a.UserID,
		UserName:       userName,
		UserRole:       a.UserRole,
		AssignedByID:   assignedByID,
		AssignedByName: assignedByName,
		AssignedAt:     a.AssignedAt,
		UnassignedAt:   unassignedAt,
		Active:         a.Active,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

func NewPartnerInteractionResponse(i PartnerInteraction) PartnerInteractionResponse {
	var note *string
	if i.Note.Valid {
		n := i.Note.String
		note = &n
	}
	return PartnerInteractionResponse{
		ID:              i.ID,
		PartnerID:       i.PartnerID,
		InteractionType: i.InteractionType,
		InteractionAt:   i.InteractionAt,
		Note:            note,
		CreatedAt:       i.CreatedAt,
	}
}

func NewPartnerReferralResponse(r PartnerReferral) PartnerReferralResponse {
	var notes *string
	if r.Notes.Valid {
		n := r.Notes.String
		notes = &n
	}
	return PartnerReferralResponse{
		ID:           r.ID,
		PartnerID:    r.PartnerID,
		LeadID:       r.LeadID,
		ReferralDate: r.ReferralDate,
		Notes:        notes,
		CreatedAt:    r.CreatedAt,
	}
}

// Helper to convert *string to string pointer (identity)
func ptrToString(s *string) *string {
	if s == nil {
		return nil
	}
	return s
}

/* ---------- PartnerCommission ---------- */

// UserBrief is a lightweight reference to a user (approver/payer of a commission).
type UserBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

// PartnerCommission is earned by a partner when a lead it referred reaches a
// CONFIRMED closing. One commission row per closing (see uq_partner_commissions_closing).
// CommissionMode/CommissionValue snapshot the partner_type's rate at calculation time, so
// later changes to the partner_type's rate don't retroactively alter past commissions.
type PartnerCommission struct {
	ID               int64
	Code             string
	PartnerID        int64
	PartnerCode      sql.NullString
	PartnerName      sql.NullString
	ReferralID       int64
	ClosingID        int64
	ClosingCode      sql.NullString
	CommissionMode   string
	CommissionValue  string
	BaseAmount       string // snapshot of closing.final_amount at calculation time
	CommissionAmount string
	Currency         string
	Status           string // PENDING, APPROVED, PAID, CANCELLED
	Note             sql.NullString
	ApprovedByUserID sql.NullInt64
	ApprovedByName   sql.NullString
	ApprovedAt       sql.NullTime
	PaidByUserID     sql.NullInt64
	PaidByName       sql.NullString
	PaidAt           sql.NullTime
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type PartnerCommissionResponse struct {
	ID               int64      `json:"id"`
	Code             string     `json:"code"`
	PartnerID        int64      `json:"partner_id"`
	PartnerCode      *string    `json:"partner_code,omitempty"`
	PartnerName      *string    `json:"partner_name,omitempty"`
	ReferralID       int64      `json:"referral_id"`
	ClosingID        int64      `json:"closing_id"`
	ClosingCode      *string    `json:"closing_code,omitempty"`
	CommissionMode   string     `json:"commission_mode"`
	CommissionValue  string     `json:"commission_value"`
	BaseAmount       string     `json:"base_amount"`
	CommissionAmount string     `json:"commission_amount"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	Note             string     `json:"note,omitempty"`
	ApprovedBy       *UserBrief `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	PaidBy           *UserBrief `json:"paid_by,omitempty"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type PartnerCommissionListResponse struct {
	Items      []PartnerCommissionResponse `json:"items"`
	Pagination PaginationMeta              `json:"pagination"`
}

type SyncCommissionsResponse struct {
	Created int64                       `json:"created"`
	Items   []PartnerCommissionResponse `json:"items"`
}

type CommissionActionRequest struct {
	Note string `json:"note,omitempty"`
}

func NewPartnerCommissionResponse(pc PartnerCommission) PartnerCommissionResponse {
	return PartnerCommissionResponse{
		ID:               pc.ID,
		Code:             pc.Code,
		PartnerID:        pc.PartnerID,
		PartnerCode:      nullStringToPtr(pc.PartnerCode),
		PartnerName:      nullStringToPtr(pc.PartnerName),
		ReferralID:       pc.ReferralID,
		ClosingID:        pc.ClosingID,
		ClosingCode:      nullStringToPtr(pc.ClosingCode),
		CommissionMode:   pc.CommissionMode,
		CommissionValue:  pc.CommissionValue,
		BaseAmount:       pc.BaseAmount,
		CommissionAmount: pc.CommissionAmount,
		Currency:         pc.Currency,
		Status:           pc.Status,
		Note:             pc.Note.String,
		ApprovedBy:       nullableUserBrief(pc.ApprovedByUserID, pc.ApprovedByName),
		ApprovedAt:       nullableTimePtr(pc.ApprovedAt),
		PaidBy:           nullableUserBrief(pc.PaidByUserID, pc.PaidByName),
		PaidAt:           nullableTimePtr(pc.PaidAt),
		CreatedAt:        pc.CreatedAt,
		UpdatedAt:        pc.UpdatedAt,
	}
}

func nullableUserBrief(id sql.NullInt64, name sql.NullString) *UserBrief {
	if !id.Valid {
		return nil
	}
	return &UserBrief{ID: id.Int64, Name: name.String}
}

func nullableTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
