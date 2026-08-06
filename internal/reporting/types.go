package reporting

import (
	"database/sql"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	PermissionReadAll = "reports.read_all"
	PermissionReadOwn = "reports.read_own"

	ReportOwnersOutlets = "owners_outlets"
	ReportActivities    = "activities"
	ReportTopups        = "topups"
	ReportClosings      = "closings"
	ReportSubscriptions = "subscriptions"
	ReportPartners      = "partners"
	ReportTargetsKPI    = "targets_kpi"
	ReportAdminOwner    = "admin_owner"
	ReportAdminOwnerOutlet = "admin_owner_outlet"
	ReportAdminNewSubscribe = "admin_new_subscribe"
	ReportAdminNasabahProvinsi = "admin_nasabah_baru_provinsi"

	ExportFormatCSV  = "CSV"
	ExportFormatXLSX = "XLSX"
	ExportFormatPDF  = "PDF"

	ExportStatusPending    = "PENDING"
	ExportStatusProcessing = "PROCESSING"
	ExportStatusCompleted  = "COMPLETED"
	ExportStatusFailed     = "FAILED"

	JobTypeGenerateExport = "REPORT_EXPORT_GENERATE"
)

type Actor struct {
	ID          int64
	RoleCode    string
	Permissions []string
}

type DashboardParams struct {
	DateFrom string
	DateTo   string
}

type DashboardCard struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type DashboardResponse struct {
	Role     string          `json:"role"`
	DateFrom string          `json:"date_from"`
	DateTo   string          `json:"date_to"`
	Cards    []DashboardCard `json:"cards"`
}

type ReportColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type ListReportsParams struct {
	DateFrom     string
	DateTo       string
	CreatedFrom  string
	CreatedTo    string
	Status       string
	Query        string
	SalesID      *int64
	SupervisorID *int64
	Province     string
	City         string
	Page         int
	Limit        int
	All          bool
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type ReportListResponse struct {
	ReportKey  string              `json:"report_key"`
	Columns    []ReportColumn      `json:"columns"`
	Items      []map[string]any    `json:"items"`
	Pagination PaginationMeta      `json:"pagination"`
	Insight    map[string]any      `json:"insight,omitempty"`
}

type CreateExportRequest struct {
	ReportKey string            `json:"report_key" binding:"required"`
	Format    string            `json:"format" binding:"required"`
	Filters   map[string]string `json:"filters"`
}

type ReportExport struct {
	ID                int64
	Code              string
	ReportKey         string
	Format            string
	Status            string
	FiltersJSON       sql.NullString
	RequestedByUserID sql.NullInt64
	RequestedByName   sql.NullString
	RequestedByRole   sql.NullString
	JobID             sql.NullInt64
	FilePath          sql.NullString
	FileName          sql.NullString
	MimeType          sql.NullString
	StorageDisk       string
	FileBlob          []byte
	RowCount          int64
	LastError         sql.NullString
	CompletedAt       sql.NullTime
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type UserBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ReportExportResponse struct {
	ID          int64             `json:"id"`
	Code        string            `json:"code"`
	ReportKey   string            `json:"report_key"`
	Format      string            `json:"format"`
	Status      string            `json:"status"`
	Filters     map[string]string `json:"filters,omitempty"`
	RequestedBy *UserBrief        `json:"requested_by,omitempty"`
	JobID       *int64            `json:"job_id,omitempty"`
	FileName    string            `json:"file_name,omitempty"`
	MimeType    string            `json:"mime_type,omitempty"`
	RowCount    int64             `json:"row_count"`
	LastError   string            `json:"last_error,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DownloadURL string            `json:"download_url,omitempty"`
}

type ReportExportListResponse struct {
	Items      []ReportExportResponse `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
}

type GenerateExportJobPayload struct {
	ExportID int64 `json:"export_id"`
}
