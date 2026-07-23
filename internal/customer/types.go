package customer

import (
	"database/sql"
	"time"
)

const (
	StatusActive  = "ACTIVE"
	StatusDeleted = "DELETED"
)

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
	Query     string
	Code      string
	Name      string
	Phone     string
	BrandName string
	Province  string
	City      string
	Page      int
	Limit     int
	Sort      string
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
