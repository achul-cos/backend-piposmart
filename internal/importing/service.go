package importing

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/jobqueue"
)

// Service applies role gating, upload/commit orchestration around the importing Repository.
type Service struct {
	repo    *Repository
	jobs    *jobqueue.Repository
	storage config.StorageConfig
}

// NewService creates a new importing Service.
func NewService(repo *Repository, jobs *jobqueue.Repository, storage config.StorageConfig) *Service {
	return &Service{repo: repo, jobs: jobs, storage: storage}
}

func isAdmin(actor identity.User) bool {
	return actor.RoleCode == RoleAdmin
}

// profilesRequiringSheetName cannot rely on auto-detection: their workbooks (PBGC-style
// per-sales-rep exports) contain several structurally-identical sheets — different sales reps,
// legacy/duplicate copies — so the admin must say which sheet to use. The same two profiles also
// never carry a sales-rep column, only a sheet-name suffix (e.g. "Call & Chat-Lidya"), so they
// additionally require an explicit target_sales_user_id.
var profilesRequiringSheetName = map[string]bool{
	ProfileSalesCallChat: true,
	ProfileSalesTarget:   true,
}

func isKnownProfile(name string) bool {
	for _, p := range knownProfiles {
		if p.Name == name {
			return true
		}
	}
	return false
}

// Upload validates and stores an uploaded workbook, then enqueues async validation. Re-uploading
// a byte-identical file returns the existing batch instead of creating a duplicate.
func (s *Service) Upload(ctx context.Context, actor identity.User, file multipart.File, header *multipart.FileHeader, declaredProfile, sheetName string, targetSalesUserID *int64) (*ImportBatchResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	// Canonicalize whitespace once at the boundary — both the dedup key and ValidateHandler's
	// sheet lookup compare against this trimmed form, so a batch is never keyed on accidental
	// leading/trailing spaces the admin typed (or a browser/curl trimmed) differently between
	// two uploads of the same file.
	sheetName = strings.TrimSpace(sheetName)
	if !strings.EqualFold(filepath.Ext(header.Filename), ".xlsx") {
		return nil, ErrInvalidFileType
	}

	maxBytes := s.storage.MaxUploadSizeMB * 1024 * 1024
	if header.Size > maxBytes {
		return nil, ErrFileTooLarge
	}

	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("importing: read upload: %w", err)
	}
	if len(content) == 0 {
		return nil, ErrEmptyFile
	}
	if int64(len(content)) > maxBytes {
		return nil, ErrFileTooLarge
	}

	if declaredProfile != "" {
		if !isKnownProfile(declaredProfile) {
			return nil, ErrUnknownProfile
		}
		if profilesRequiringSheetName[declaredProfile] {
			if strings.TrimSpace(sheetName) == "" {
				return nil, ErrSheetNameRequired
			}
			if targetSalesUserID == nil || *targetSalesUserID < 1 {
				return nil, ErrTargetSalesUserRequired
			}
		}
	} else if strings.TrimSpace(sheetName) != "" {
		// sheet_name only makes sense alongside an explicit profile — without a declared profile
		// there is nothing to verify that sheet against.
		return nil, ErrSheetNameNeedsProfile
	}

	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])

	profileForRow := declaredProfile
	if profileForRow == "" {
		profileForRow = ProfilePendingDetection
	}

	existing, err := s.repo.FindBatchBySHA256AndProfile(ctx, hash, profileForRow, sheetName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		resp := NewImportBatchResponse(*existing)
		return &resp, nil
	}

	uploadDir := filepath.Join(s.storage.UploadDirectory, "imports")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("importing: create upload directory: %w", err)
	}
	// Filename is derived from the hash, never from user input — prevents path traversal.
	filePath := filepath.Join(uploadDir, hash+".xlsx")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		return nil, fmt.Errorf("importing: store upload: %w", err)
	}

	contextKey := fmt.Sprintf("%s|%s|", profileForRow, strings.TrimSpace(sheetName))
	if targetSalesUserID != nil {
		contextKey = fmt.Sprintf("%s%d", contextKey, *targetSalesUserID)
	}
	contextSum := sha256.Sum256([]byte(contextKey))
	contextHash := hex.EncodeToString(contextSum[:])
	code := fmt.Sprintf("IMPORT-%s-%s-%s", time.Now().Format("20060102"), hash[:8], contextHash[:6])

	batchID, err := s.repo.CreateBatch(ctx, code, profileForRow, sheetName, targetSalesUserID, header.Filename, hash, filePath, content, actor.ID)
	if err != nil {
		return nil, err
	}

	jobID, err := s.jobs.Enqueue(ctx, JobTypeValidate, ValidateJobPayload{BatchID: batchID}, &actor.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetValidateJobID(ctx, batchID, jobID); err != nil {
		return nil, err
	}

	batch, err := s.repo.GetBatchByID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	resp := NewImportBatchResponse(*batch)
	return &resp, nil
}

func (s *Service) ListBatches(ctx context.Context, actor identity.User, params ListBatchesParams) (*ImportBatchListResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	page, limit := params.Page, params.Limit
	if params.All {
		page = 1
		limit = 0
	} else {
		if page < 1 {
			page = 1
		}
		if limit < 1 {
			limit = 20
		}
		if limit > 10000 {
			limit = 10000
		}
	}
	params.Page, params.Limit = page, limit

	items, total, err := s.repo.ListBatches(ctx, params)
	if err != nil {
		return nil, err
	}
	responses := make([]ImportBatchResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewImportBatchResponse(item))
	}
	return &ImportBatchListResponse{Items: responses, Meta: ListMeta{Page: page, Limit: resolveReturnedLimit(params.All, limit, len(items), total), Total: total}}, nil
}

func (s *Service) GetBatch(ctx context.Context, actor identity.User, id int64) (*ImportBatchResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	batch, err := s.repo.GetBatchByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := NewImportBatchResponse(*batch)
	return &resp, nil
}

func (s *Service) ListRows(ctx context.Context, actor identity.User, batchID int64, params ListRowsParams) (*ImportRowListResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	if _, err := s.repo.GetBatchByID(ctx, batchID); err != nil {
		return nil, err
	}
	page, limit := params.Page, params.Limit
	if params.All {
		page = 1
		limit = 0
	} else {
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 200 {
			limit = 50
		}
	}
	params.Page, params.Limit = page, limit

	items, total, err := s.repo.ListRows(ctx, batchID, params)
	if err != nil {
		return nil, err
	}
	responses := make([]ImportRowResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewImportRowResponse(item))
	}
	return &ImportRowListResponse{Items: responses, Meta: ListMeta{Page: page, Limit: resolveReturnedLimit(params.All, limit, len(items), total), Total: total}}, nil
}

// RelinkRow manually resolves a reconciliation candidate (RowStatusUnmatched) — admin supplies
// the owner/outlet/lead ID the row's own code/name couldn't resolve at commit time, moving it
// back to VALID for the next batch commit to pick up. Deliberately does not re-trigger commit
// itself (the batch-level commit endpoint already exists and handles the retry job wiring).
func (s *Service) RelinkRow(ctx context.Context, actor identity.User, batchID, rowID int64, req RelinkRowRequest) (*ImportRowResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	if req.OwnerID == nil && req.OutletID == nil && req.LeadID == nil {
		return nil, ErrRelinkEntityRequired
	}
	row, err := s.repo.FindRowByID(ctx, batchID, rowID)
	if err != nil {
		return nil, err
	}
	if row.Status != RowStatusUnmatched {
		return nil, ErrRowNotUnmatched
	}
	if err := s.repo.RelinkRow(ctx, rowID, req.OwnerID, req.OutletID, req.LeadID); err != nil {
		return nil, err
	}
	relinked, err := s.repo.FindRowByID(ctx, batchID, rowID)
	if err != nil {
		return nil, err
	}
	resp := NewImportRowResponse(relinked)
	return &resp, nil
}

// GetSummary aggregates batch counts per status (GET /imports/summary) — so admin can see how
// many batches need attention without paging through GET /imports?status=... one status at a time.
func (s *Service) GetSummary(ctx context.Context, actor identity.User) (*BatchSummaryResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	counts, err := s.repo.GetBatchStatusCounts(ctx)
	if err != nil {
		return nil, err
	}
	var total, needsAttention int64
	for status, count := range counts {
		total += count
		if status == BatchStatusValidationFailed || status == BatchStatusCommitFailed {
			needsAttention += count
		}
	}
	return &BatchSummaryResponse{Total: total, CountsByStatus: counts, NeedsAttention: needsAttention}, nil
}

func resolveReturnedLimit(all bool, limit int, itemCount int, total int64) int {
	if !all {
		return limit
	}
	if total == 0 {
		return 0
	}
	return itemCount
}

// ExportRejectedRows builds a CSV of every INVALID row plus its validation errors.
func (s *Service) ExportRejectedRows(ctx context.Context, actor identity.User, batchID int64) ([]byte, string, error) {
	if !isAdmin(actor) {
		return nil, "", ErrForbidden
	}
	batch, err := s.repo.GetBatchByID(ctx, batchID)
	if err != nil {
		return nil, "", err
	}
	rows, err := s.repo.ListRowsByStatus(ctx, batchID, RowStatusInvalid)
	if err != nil {
		return nil, "", err
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"row_index", "raw_payload", "validation_errors"})
	for _, row := range rows {
		var errs []string
		if row.ValidationErrors.Valid {
			_ = json.Unmarshal([]byte(row.ValidationErrors.String), &errs)
		}
		_ = w.Write([]string{
			fmt.Sprintf("%d", row.RowIndex),
			string(row.RawPayload),
			strings.Join(errs, "; "),
		})
	}
	w.Flush()

	filename := fmt.Sprintf("%s-rejected-rows.csv", batch.Code)
	return []byte(buf.String()), filename, w.Error()
}

func (s *Service) TriggerCommit(ctx context.Context, actor identity.User, batchID int64) (*ImportBatchResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	batch, err := s.repo.GetBatchByID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if canReuseCommitResult(batch.Status) {
		resp := NewImportBatchResponse(*batch)
		return &resp, nil
	}
	if err := validateCommitStatus(batch.Status); err != nil {
		return nil, err
	}

	jobID, err := s.jobs.Enqueue(ctx, JobTypeCommit, CommitJobPayload{BatchID: batchID, TriggeredByUserID: actor.ID}, &actor.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetCommitJobID(ctx, batchID, jobID); err != nil {
		return nil, err
	}
	if err := s.repo.SetBatchCommitting(ctx, batchID); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetBatchByID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	resp := NewImportBatchResponse(*updated)
	return &resp, nil
}

type OriginalFilePayload struct {
	Path             string
	OriginalFilename string
	Content          []byte
	ContentType      string
}

func (s *Service) GetOriginalFile(ctx context.Context, actor identity.User, batchID int64) (*OriginalFilePayload, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	batch, err := s.repo.GetBatchFileSource(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(batch.FilePath) != "" {
		if _, err := os.Stat(batch.FilePath); err == nil {
			return &OriginalFilePayload{
				Path:             batch.FilePath,
				OriginalFilename: batch.OriginalFilename,
				ContentType:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			}, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("importing: stat original file: %w", err)
		}
	}
	if len(batch.FileBlob) == 0 {
		return nil, ErrFileUnavailable
	}
	return &OriginalFilePayload{
		OriginalFilename: batch.OriginalFilename,
		Content:          batch.FileBlob,
		ContentType:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}, nil
}
