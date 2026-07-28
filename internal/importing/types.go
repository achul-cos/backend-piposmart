// Package importing implements the Excel import framework: upload -> async validate -> preview
// -> async commit. Named "importing" (not "import") because import is a reserved Go keyword.
package importing

import (
	"database/sql"
	"encoding/json"
	"time"
)

const (
	ProfileOwnerOutlet      = "OWNER_OUTLET"
	ProfileNonRegister      = "NON_REGISTER"
	ProfilePendingDetection = "PENDING_DETECTION" // profile not yet known; resolved by the IMPORT_VALIDATE job

	BatchStatusUploaded         = "UPLOADED"
	BatchStatusValidating       = "VALIDATING"
	BatchStatusValidated        = "VALIDATED"
	BatchStatusValidationFailed = "VALIDATION_FAILED"
	BatchStatusCommitting       = "COMMITTING"
	BatchStatusCommitted        = "COMMITTED"
	BatchStatusCommitFailed     = "COMMIT_FAILED"

	RowStatusPending      = "PENDING"
	RowStatusValid        = "VALID"
	RowStatusInvalid      = "INVALID"
	RowStatusCommitted    = "COMMITTED"
	RowStatusCommitFailed = "COMMIT_FAILED"

	JobTypeValidate = "IMPORT_VALIDATE"
	JobTypeCommit   = "IMPORT_COMMIT"

	RoleAdmin = "ADMIN"
)

// ImportBatch is one uploaded file and its processing lifecycle.
type ImportBatch struct {
	ID                  int64
	Code                string
	Profile             string
	OriginalFilename    string
	FileSHA256          string
	FilePath            string
	Status              string
	TotalRows           int
	ValidRows           int
	InvalidRows         int
	CommittedRows       int
	ProgressPercentage  int
	ValidateJobID       sql.NullInt64
	CommitJobID         sql.NullInt64
	ErrorMessage        sql.NullString
	UploadedByUserID    int64
	UploadedByName      sql.NullString
	CommittedByUserID   sql.NullInt64
	CommittedByName     sql.NullString
	UploadedAt          time.Time
	ValidatedAt         sql.NullTime
	CommittedAt         sql.NullTime
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ImportRow is one parsed Excel data row within a batch.
type ImportRow struct {
	ID               int64
	BatchID          int64
	RowIndex         int
	RawPayload       json.RawMessage
	Status           string
	ValidationErrors sql.NullString // json.RawMessage can't Scan a NULL driver value; VALID rows have no errors
	OwnerID          sql.NullInt64
	OutletID         sql.NullInt64
	LeadID           sql.NullInt64
	CommitError      sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ActorRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ImportBatchResponse struct {
	ID                 int64      `json:"id"`
	Code               string     `json:"code"`
	Profile            string     `json:"profile"`
	OriginalFilename   string     `json:"original_filename"`
	Status             string     `json:"status"`
	TotalRows          int        `json:"total_rows"`
	ValidRows          int        `json:"valid_rows"`
	InvalidRows        int        `json:"invalid_rows"`
	CommittedRows      int        `json:"committed_rows"`
	ProgressPercentage int        `json:"progress_percentage"`
	ErrorMessage       *string    `json:"error_message,omitempty"`
	UploadedBy         ActorRef   `json:"uploaded_by"`
	CommittedBy        *ActorRef  `json:"committed_by,omitempty"`
	UploadedAt         time.Time  `json:"uploaded_at"`
	ValidatedAt        *time.Time `json:"validated_at,omitempty"`
	CommittedAt        *time.Time `json:"committed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func NewImportBatchResponse(b ImportBatch) ImportBatchResponse {
	resp := ImportBatchResponse{
		ID:                 b.ID,
		Code:               b.Code,
		Profile:            b.Profile,
		OriginalFilename:   b.OriginalFilename,
		Status:             b.Status,
		TotalRows:          b.TotalRows,
		ValidRows:          b.ValidRows,
		InvalidRows:        b.InvalidRows,
		CommittedRows:      b.CommittedRows,
		ProgressPercentage: b.ProgressPercentage,
		UploadedBy:         ActorRef{ID: b.UploadedByUserID, Name: b.UploadedByName.String},
		UploadedAt:         b.UploadedAt,
		CreatedAt:          b.CreatedAt,
		UpdatedAt:          b.UpdatedAt,
	}
	if b.ErrorMessage.Valid {
		resp.ErrorMessage = &b.ErrorMessage.String
	}
	if b.CommittedByUserID.Valid {
		resp.CommittedBy = &ActorRef{ID: b.CommittedByUserID.Int64, Name: b.CommittedByName.String}
	}
	if b.ValidatedAt.Valid {
		resp.ValidatedAt = &b.ValidatedAt.Time
	}
	if b.CommittedAt.Valid {
		resp.CommittedAt = &b.CommittedAt.Time
	}
	return resp
}

type ImportBatchListResponse struct {
	Items []ImportBatchResponse `json:"items"`
	Meta  ListMeta              `json:"meta"`
}

type ListMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type ImportRowResponse struct {
	ID               int64           `json:"id"`
	BatchID          int64           `json:"batch_id"`
	RowIndex         int             `json:"row_index"`
	RawPayload       json.RawMessage `json:"raw_payload"`
	Status           string          `json:"status"`
	ValidationErrors json.RawMessage `json:"validation_errors,omitempty"`
	OwnerID          *int64          `json:"owner_id,omitempty"`
	OutletID         *int64          `json:"outlet_id,omitempty"`
	LeadID           *int64          `json:"lead_id,omitempty"`
	CommitError      *string         `json:"commit_error,omitempty"`
}

func NewImportRowResponse(r ImportRow) ImportRowResponse {
	resp := ImportRowResponse{
		ID:         r.ID,
		BatchID:    r.BatchID,
		RowIndex:   r.RowIndex,
		RawPayload: r.RawPayload,
		Status:     r.Status,
	}
	if r.ValidationErrors.Valid {
		resp.ValidationErrors = json.RawMessage(r.ValidationErrors.String)
	}
	if r.OwnerID.Valid {
		resp.OwnerID = &r.OwnerID.Int64
	}
	if r.OutletID.Valid {
		resp.OutletID = &r.OutletID.Int64
	}
	if r.LeadID.Valid {
		resp.LeadID = &r.LeadID.Int64
	}
	if r.CommitError.Valid {
		resp.CommitError = &r.CommitError.String
	}
	return resp
}

type ImportRowListResponse struct {
	Items []ImportRowResponse `json:"items"`
	Meta  ListMeta            `json:"meta"`
}

type ListRowsParams struct {
	Status string
	Page   int
	Limit  int
}

type ListBatchesParams struct {
	Status  string
	Profile string
	Page    int
	Limit   int
}

// ValidateJobPayload / CommitJobPayload are the job_queue payloads for the two job types this
// package registers.
type ValidateJobPayload struct {
	BatchID int64 `json:"batch_id"`
}

type CommitJobPayload struct {
	BatchID           int64 `json:"batch_id"`
	TriggeredByUserID int64 `json:"triggered_by_user_id"`
}
