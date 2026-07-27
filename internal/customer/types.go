package customer

import (
	"database/sql"
	"time"
)

const (
	StatusActive  = "ACTIVE"
	StatusDeleted = "DELETED"

	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	ScopeActive  = "ACTIVE"
	ScopeDeleted = "DELETED"
	ScopeAll     = "ALL"
)

type Actor struct {
	ID       int64
	RoleCode string
}

type Owner struct {
	ID          int64
	Code        string
	Name        string
	Phone       sql.NullString
	Email       sql.NullString
	BrandName   sql.NullString
	Province    sql.NullString
	City        sql.NullString
	Address     sql.NullString
	Status      string
	OutletCount int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Outlet struct {
	ID        int64
	OwnerID   sql.NullInt64
	Code      string
	Name      string
	Phone     sql.NullString
	Province  sql.NullString
	City      sql.NullString
	Address   sql.NullString
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OutletOverview struct {
	ID                       int64
	OwnerID                  sql.NullInt64
	OwnerCode                sql.NullString
	OwnerName                sql.NullString
	OwnerPhone               sql.NullString
	OwnerEmail               sql.NullString
	OwnerBrandName           sql.NullString
	AccountCode              string
	WalletID                 int64
	WalletBalance            string
	WalletLedgerBalance      string
	WalletStatus             string
	WalletCreatedAt          time.Time
	WalletUpdatedAt          time.Time
	Code                     string
	Name                     string
	Phone                    sql.NullString
	Province                 sql.NullString
	City                     sql.NullString
	Address                  sql.NullString
	Status                   string
	SubscriptionCount        int64
	ActiveSubscriptionCount  int64
	LatestSubscriptionStatus sql.NullString
	LatestSubscriptionStart  sql.NullTime
	LatestSubscriptionUntil  sql.NullTime
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type OwnerResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
	BrandName   string    `json:"brand_name,omitempty"`
	Province    string    `json:"province,omitempty"`
	City        string    `json:"city,omitempty"`
	Address     string    `json:"address,omitempty"`
	Status      string    `json:"status"`
	OutletCount int64     `json:"outlet_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OutletResponse struct {
	ID        int64     `json:"id"`
	OwnerID   *int64    `json:"owner_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	Province  string    `json:"province,omitempty"`
	City      string    `json:"city,omitempty"`
	Address   string    `json:"address,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OwnerBriefResponse struct {
	ID        *int64 `json:"id,omitempty"`
	Code      string `json:"code,omitempty"`
	Name      string `json:"name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Email     string `json:"email,omitempty"`
	BrandName string `json:"brand_name,omitempty"`
	Message   string `json:"message,omitempty"`
}

type WalletBriefResponse struct {
	ID            int64     `json:"id"`
	AccountCode   string    `json:"account_code"`
	Currency      string    `json:"currency"`
	Balance       string    `json:"balance"`
	LedgerBalance string    `json:"ledger_balance"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SubscriptionSummaryResponse struct {
	TotalSubscriptions       int64  `json:"total_subscriptions"`
	ActiveSubscriptions      int64  `json:"active_subscriptions"`
	LatestSubscriptionStatus string `json:"latest_subscription_status,omitempty"`
	LatestSubscriptionStart  string `json:"latest_subscription_start,omitempty"`
	LatestSubscriptionEnd    string `json:"latest_subscription_end,omitempty"`
}

type OutletOverviewResponse struct {
	ID                  int64                       `json:"id"`
	Owner               OwnerBriefResponse          `json:"owner"`
	Wallet              WalletBriefResponse         `json:"wallet"`
	Code                string                      `json:"code"`
	Name                string                      `json:"name"`
	Phone               string                      `json:"phone,omitempty"`
	Province            string                      `json:"province,omitempty"`
	City                string                      `json:"city,omitempty"`
	Address             string                      `json:"address,omitempty"`
	Status              string                      `json:"status"`
	SubscriptionSummary SubscriptionSummaryResponse `json:"subscription_summary"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

type OutletDetailResponse struct {
	ID                  int64                       `json:"id"`
	Owner               OwnerBriefResponse          `json:"owner"`
	Wallet              WalletBriefResponse         `json:"wallet"`
	Code                string                      `json:"code"`
	Name                string                      `json:"name"`
	Phone               string                      `json:"phone,omitempty"`
	Province            string                      `json:"province,omitempty"`
	City                string                      `json:"city,omitempty"`
	Address             string                      `json:"address,omitempty"`
	Status              string                      `json:"status"`
	SubscriptionSummary SubscriptionSummaryResponse `json:"subscription_summary"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

type CreateOwnerRequest struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	BrandName string `json:"brand_name"`
	Province  string `json:"province"`
	City      string `json:"city"`
	Address   string `json:"address"`
}

type UpdateOwnerRequest struct {
	Code      *string `json:"code"`
	Name      *string `json:"name"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	BrandName *string `json:"brand_name"`
	Province  *string `json:"province"`
	City      *string `json:"city"`
	Address   *string `json:"address"`
}

type CreateOutletRequest struct {
	Code     string `json:"code" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone"`
	Province string `json:"province"`
	City     string `json:"city"`
	Address  string `json:"address"`
}

type UpdateOutletRequest struct {
	Code     *string `json:"code"`
	Name     *string `json:"name"`
	Phone    *string `json:"phone"`
	Province *string `json:"province"`
	City     *string `json:"city"`
	Address  *string `json:"address"`
}

type BulkIDRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,dive,min=1"`
}

type BulkOwnerCreateRequest struct {
	Items []CreateOwnerRequest `json:"items" binding:"required,min=1,dive"`
}

type BulkOwnerUpdateRequest struct {
	Items []BulkOwnerUpdateItem `json:"items" binding:"required,min=1,dive"`
}

type BulkOwnerUpdateItem struct {
	ID        int64   `json:"id" binding:"required,min=1"`
	Code      *string `json:"code"`
	Name      *string `json:"name"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	BrandName *string `json:"brand_name"`
	Province  *string `json:"province"`
	City      *string `json:"city"`
	Address   *string `json:"address"`
}

type OwnerUpdateInput struct {
	ID              int64
	Request         UpdateOwnerRequest
	NormalizedPhone *string
}

type BulkOutletCreateRequest struct {
	Items []CreateOutletRequest `json:"items" binding:"required,min=1,dive"`
}

type BulkOutletUpdateRequest struct {
	Items []BulkOutletUpdateItem `json:"items" binding:"required,min=1,dive"`
}

type BulkOutletUpdateItem struct {
	ID       int64   `json:"id" binding:"required,min=1"`
	Code     *string `json:"code"`
	Name     *string `json:"name"`
	Phone    *string `json:"phone"`
	Province *string `json:"province"`
	City     *string `json:"city"`
	Address  *string `json:"address"`
}

type OutletUpdateInput struct {
	ID              int64
	Request         UpdateOutletRequest
	NormalizedPhone *string
}

type ListParams struct {
	Query              string
	Code               string
	Name               string
	Phone              string
	BrandName          string
	Province           string
	City               string
	OwnerID            *int64
	SubscriptionStatus string
	SubscriptionMonth  string
	Scope              string
	Page               int
	Limit              int
	Sort               string
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type OwnerListResponse struct {
	Items      []OwnerResponse `json:"items"`
	Pagination PaginationMeta  `json:"pagination"`
}

type OutletListResponse struct {
	Items      []OutletResponse `json:"items"`
	Pagination PaginationMeta   `json:"pagination"`
}

type OutletOverviewListResponse struct {
	Items      []OutletOverviewResponse `json:"items"`
	Pagination PaginationMeta           `json:"pagination"`
}

type OwnerBulkResponse struct {
	Items []OwnerResponse `json:"items"`
	Total int             `json:"total"`
}

type OutletBulkResponse struct {
	Items []OutletResponse `json:"items"`
	Total int              `json:"total"`
}

type BulkActionResponse struct {
	IDs      []int64 `json:"ids"`
	Affected int64   `json:"affected"`
}

func NewOwnerResponse(owner Owner) OwnerResponse {
	return OwnerResponse{
		ID:          owner.ID,
		Code:        owner.Code,
		Name:        owner.Name,
		Phone:       owner.Phone.String,
		Email:       owner.Email.String,
		BrandName:   owner.BrandName.String,
		Province:    owner.Province.String,
		City:        owner.City.String,
		Address:     owner.Address.String,
		Status:      owner.Status,
		OutletCount: owner.OutletCount,
		CreatedAt:   owner.CreatedAt,
		UpdatedAt:   owner.UpdatedAt,
	}
}

func NewOutletResponse(outlet Outlet) OutletResponse {
	var ownerID *int64
	if outlet.OwnerID.Valid {
		value := outlet.OwnerID.Int64
		ownerID = &value
	}
	return OutletResponse{
		ID:        outlet.ID,
		OwnerID:   ownerID,
		Code:      outlet.Code,
		Name:      outlet.Name,
		Phone:     outlet.Phone.String,
		Province:  outlet.Province.String,
		City:      outlet.City.String,
		Address:   outlet.Address.String,
		Status:    outlet.Status,
		CreatedAt: outlet.CreatedAt,
		UpdatedAt: outlet.UpdatedAt,
	}
}

func NewOutletOverviewResponse(item OutletOverview) OutletOverviewResponse {
	return OutletOverviewResponse{
		ID:    item.ID,
		Owner: newOwnerBriefResponse(item),
		Wallet: WalletBriefResponse{
			ID:            item.WalletID,
			AccountCode:   item.AccountCode,
			Currency:      "IDR",
			Balance:       item.WalletBalance,
			LedgerBalance: item.WalletLedgerBalance,
			Status:        item.WalletStatus,
			CreatedAt:     item.WalletCreatedAt,
			UpdatedAt:     item.WalletUpdatedAt,
		},
		Code:     item.Code,
		Name:     item.Name,
		Phone:    item.Phone.String,
		Province: item.Province.String,
		City:     item.City.String,
		Address:  item.Address.String,
		Status:   item.Status,
		SubscriptionSummary: SubscriptionSummaryResponse{
			TotalSubscriptions:       item.SubscriptionCount,
			ActiveSubscriptions:      item.ActiveSubscriptionCount,
			LatestSubscriptionStatus: item.LatestSubscriptionStatus.String,
			LatestSubscriptionStart:  formatNullDate(item.LatestSubscriptionStart),
			LatestSubscriptionEnd:    formatNullDate(item.LatestSubscriptionUntil),
		},
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func NewOutletDetailResponse(item OutletOverview) OutletDetailResponse {
	overview := NewOutletOverviewResponse(item)
	return OutletDetailResponse{
		ID:                  overview.ID,
		Owner:               overview.Owner,
		Wallet:              overview.Wallet,
		Code:                overview.Code,
		Name:                overview.Name,
		Phone:               overview.Phone,
		Province:            overview.Province,
		City:                overview.City,
		Address:             overview.Address,
		Status:              overview.Status,
		SubscriptionSummary: overview.SubscriptionSummary,
		CreatedAt:           overview.CreatedAt,
		UpdatedAt:           overview.UpdatedAt,
	}
}

func newOwnerBriefResponse(item OutletOverview) OwnerBriefResponse {
	if !item.OwnerID.Valid {
		return OwnerBriefResponse{Message: "Data owner tidak tersedia"}
	}
	id := item.OwnerID.Int64
	return OwnerBriefResponse{
		ID:        &id,
		Code:      item.OwnerCode.String,
		Name:      item.OwnerName.String,
		Phone:     item.OwnerPhone.String,
		Email:     item.OwnerEmail.String,
		BrandName: item.OwnerBrandName.String,
	}
}

func formatNullDate(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}
