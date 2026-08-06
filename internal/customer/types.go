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
	District    sql.NullString
	SubDistrict sql.NullString
	Address     sql.NullString
	Status                string
	OutletCount           int64
	SubscribedOutletCount int64
	SubscriptionStatus    string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Outlet struct {
	ID        int64
	OwnerID   sql.NullInt64
	Code      string
	Name      string
	Phone     sql.NullString
	Province  sql.NullString
	City      sql.NullString
	District    sql.NullString
	SubDistrict sql.NullString
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
	OwnerCreatedAt           sql.NullTime
	Code                     string
	Name                     string
	Phone                    sql.NullString
	Province                 sql.NullString
	City                     sql.NullString
	District                 sql.NullString
	SubDistrict              sql.NullString
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

// OwnerAge/OwnerSubscriptionStatus classify an owner for the Owner overview endpoint. Wallet
// balance is owner-scoped (see internal/wallet) — this is the only place it's surfaced now that
// OutletOverview no longer carries a redundant per-outlet copy of the same owner wallet.
const (
	OwnerAgeNew = "NEW"
	OwnerAgeOld = "OLD"

	OwnerSubscriptionStatusSubscribed    = "SUBSCRIBE"
	OwnerSubscriptionStatusNotSubscribed = "NOT_SUBSCRIBE"
	OwnerSubscriptionStatusTrial         = "TRIAL"
)

func CalculateOwnerSubscriptionStatus(createdAt time.Time, subscribedOutletCount int64) string {
	if subscribedOutletCount > 0 {
		return OwnerSubscriptionStatusSubscribed
	}
	if time.Since(createdAt) <= 14*24*time.Hour {
		return OwnerSubscriptionStatusTrial
	}
	return OwnerSubscriptionStatusNotSubscribed
}

type OwnerOverview struct {
	ID                    int64
	Code                  string
	Name                  string
	Phone                 sql.NullString
	Email                 sql.NullString
	BrandName             sql.NullString
	Province              sql.NullString
	City                  sql.NullString
	District              sql.NullString
	SubDistrict           sql.NullString
	Address               sql.NullString
	Status                string
	AccountCode           string
	WalletID              int64
	WalletBalance         string
	WalletLedgerBalance   string
	WalletStatus          string
	WalletCreatedAt       time.Time
	WalletUpdatedAt       time.Time
	TotalTransferred      string
	TotalTopup            string
	TotalSpent            string
	AgeStatus             string
	SubscriptionStatus    string
	SubscribedOutletCount int64
	OutletCount           int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
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
	District    string    `json:"district,omitempty"`
	SubDistrict string    `json:"sub_district,omitempty"`
	Address     string    `json:"address,omitempty"`
	Status                string    `json:"status"`
	SubscriptionStatus    string    `json:"subscription_status,omitempty"`
	SubscribedOutletCount int64     `json:"subscribed_outlet_count,omitempty"`
	OutletCount           int64     `json:"outlet_count,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type OutletResponse struct {
	ID        int64     `json:"id"`
	OwnerID   *int64    `json:"owner_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	Province  string    `json:"province,omitempty"`
	City      string    `json:"city,omitempty"`
	District    string    `json:"district,omitempty"`
	SubDistrict string    `json:"sub_district,omitempty"`
	Address   string    `json:"address,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OwnerBriefResponse struct {
	ID        *int64     `json:"id,omitempty"`
	Code      string     `json:"code,omitempty"`
	Name      string     `json:"name,omitempty"`
	Phone     string     `json:"phone,omitempty"`
	Email     string     `json:"email,omitempty"`
	BrandName string     `json:"brand_name,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	Message   string     `json:"message,omitempty"`
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
	Code                string                      `json:"code"`
	Name                string                      `json:"name"`
	Phone               string                      `json:"phone,omitempty"`
	Province            string                      `json:"province,omitempty"`
	City                string                      `json:"city,omitempty"`
	District            string                      `json:"district,omitempty"`
	SubDistrict         string                      `json:"sub_district,omitempty"`
	Address             string                      `json:"address,omitempty"`
	Status              string                      `json:"status"`
	SubscriptionSummary SubscriptionSummaryResponse `json:"subscription_summary"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

type OutletDetailResponse struct {
	ID                  int64                       `json:"id"`
	Owner               OwnerBriefResponse          `json:"owner"`
	Code                string                      `json:"code"`
	Name                string                      `json:"name"`
	Phone               string                      `json:"phone,omitempty"`
	Province            string                      `json:"province,omitempty"`
	City                string                      `json:"city,omitempty"`
	District            string                      `json:"district,omitempty"`
	SubDistrict         string                      `json:"sub_district,omitempty"`
	Address             string                      `json:"address,omitempty"`
	Status              string                      `json:"status"`
	SubscriptionSummary SubscriptionSummaryResponse `json:"subscription_summary"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

// OwnerBalanceResponse surfaces the owner-scoped wallet plus the rollups the client asked for:
// how much has been transferred in, topped up, and spent across every outlet this owner owns.
type OwnerBalanceResponse struct {
	Wallet           WalletBriefResponse `json:"wallet"`
	TotalTransferred string              `json:"total_transferred"`
	TotalTopup       string              `json:"total_topup"`
	TotalSpent       string              `json:"total_spent"`
}

type OwnerStatusResponse struct {
	AgeStatus             string `json:"age_status"`
	SubscriptionStatus    string `json:"subscription_status"`
	SubscribedOutletCount int64  `json:"subscribed_outlet_count"`
	OutletCount           int64  `json:"outlet_count"`
}

type OwnerOverviewResponse struct {
	ID          int64                `json:"id"`
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Phone       string               `json:"phone,omitempty"`
	Email       string               `json:"email,omitempty"`
	BrandName   string               `json:"brand_name,omitempty"`
	Province    string               `json:"province,omitempty"`
	City        string               `json:"city,omitempty"`
	District    string               `json:"district,omitempty"`
	SubDistrict string               `json:"sub_district,omitempty"`
	Address     string               `json:"address,omitempty"`
	Status      string               `json:"status"`
	Balance     OwnerBalanceResponse `json:"balance"`
	OwnerStatus OwnerStatusResponse  `json:"owner_status"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

type CreateOwnerRequest struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	BrandName string `json:"brand_name"`
	Province  string `json:"province"`
	City      string `json:"city"`
	District    string `json:"district"`
	SubDistrict string `json:"sub_district"`
	Address   string `json:"address"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type UpdateOwnerRequest struct {
	Code      *string `json:"code"`
	Name      *string `json:"name"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	BrandName *string `json:"brand_name"`
	Province  *string `json:"province"`
	City      *string `json:"city"`
	District    *string `json:"district"`
	SubDistrict *string `json:"sub_district"`
	Address   *string `json:"address"`
}

type CreateOutletRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone"`
	Province string `json:"province"`
	City     string `json:"city"`
	District    string `json:"district"`
	SubDistrict string `json:"sub_district"`
	Address  string `json:"address"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type UpdateOutletRequest struct {
	Code     *string `json:"code"`
	Name     *string `json:"name"`
	Phone    *string `json:"phone"`
	Province *string `json:"province"`
	City     *string `json:"city"`
	District    *string `json:"district"`
	SubDistrict *string `json:"sub_district"`
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
	CreatedFrom        *time.Time
	CreatedTo          *time.Time
	OwnerID            *int64
	SubscriptionStatus string
	SubscriptionMonth  string
	Scope              string
	All                bool
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
	subStatus := owner.SubscriptionStatus
	if subStatus == "" {
		subStatus = CalculateOwnerSubscriptionStatus(owner.CreatedAt, owner.SubscribedOutletCount)
	}
	return OwnerResponse{
		ID:                    owner.ID,
		Code:                  owner.Code,
		Name:                  owner.Name,
		Phone:                 owner.Phone.String,
		Email:                 owner.Email.String,
		BrandName:             owner.BrandName.String,
		Province:              owner.Province.String,
		City:                  owner.City.String,
		District:              owner.District.String,
		SubDistrict:           owner.SubDistrict.String,
		Address:               owner.Address.String,
		Status:                owner.Status,
		SubscriptionStatus:    subStatus,
		SubscribedOutletCount: owner.SubscribedOutletCount,
		OutletCount:           owner.OutletCount,
		CreatedAt:             owner.CreatedAt,
		UpdatedAt:             owner.UpdatedAt,
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
		District:  outlet.District.String,
		SubDistrict: outlet.SubDistrict.String,
		Address:   outlet.Address.String,
		Status:    outlet.Status,
		CreatedAt: outlet.CreatedAt,
		UpdatedAt: outlet.UpdatedAt,
	}
}

func NewOutletOverviewResponse(item OutletOverview) OutletOverviewResponse {
	return OutletOverviewResponse{
		ID:       item.ID,
		Owner:    newOwnerBriefResponse(item),
		Code:     item.Code,
		Name:     item.Name,
		Phone:    item.Phone.String,
		Province: item.Province.String,
		City:     item.City.String,
		District: item.District.String,
		SubDistrict: item.SubDistrict.String,
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
		Code:                overview.Code,
		Name:                overview.Name,
		Phone:               overview.Phone,
		Province:            overview.Province,
		City:                overview.City,
		District:            overview.District,
		SubDistrict:         overview.SubDistrict,
		Address:             overview.Address,
		Status:              overview.Status,
		SubscriptionSummary: overview.SubscriptionSummary,
		CreatedAt:           overview.CreatedAt,
		UpdatedAt:           overview.UpdatedAt,
	}
}

func NewOwnerOverviewResponse(item OwnerOverview) OwnerOverviewResponse {
	return OwnerOverviewResponse{
		ID:        item.ID,
		Code:      item.Code,
		Name:      item.Name,
		Phone:     item.Phone.String,
		Email:     item.Email.String,
		BrandName: item.BrandName.String,
		Province:  item.Province.String,
		City:      item.City.String,
		District:  item.District.String,
		SubDistrict: item.SubDistrict.String,
		Address:   item.Address.String,
		Status:    item.Status,
		Balance: OwnerBalanceResponse{
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
			TotalTransferred: item.TotalTransferred,
			TotalTopup:       item.TotalTopup,
			TotalSpent:       item.TotalSpent,
		},
		OwnerStatus: OwnerStatusResponse{
			AgeStatus:             item.AgeStatus,
			SubscriptionStatus:    item.SubscriptionStatus,
			SubscribedOutletCount: item.SubscribedOutletCount,
			OutletCount:           item.OutletCount,
		},
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func newOwnerBriefResponse(item OutletOverview) OwnerBriefResponse {
	if !item.OwnerID.Valid {
		return OwnerBriefResponse{Message: "Data owner tidak tersedia"}
	}
	id := item.OwnerID.Int64
	resp := OwnerBriefResponse{
		ID:        &id,
		Code:      item.OwnerCode.String,
		Name:      item.OwnerName.String,
		Phone:     item.OwnerPhone.String,
		Email:     item.OwnerEmail.String,
		BrandName: item.OwnerBrandName.String,
	}
	if item.OwnerCreatedAt.Valid {
		t := item.OwnerCreatedAt.Time
		resp.CreatedAt = &t
	}
	return resp
}

func formatNullDate(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}
