package importing

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
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

// Upload validates and stores an uploaded workbook, then enqueues async validation. Re-uploading
// a byte-identical file returns the existing batch instead of creating a duplicate.
func (s *Service) Upload(ctx context.Context, actor identity.User, file multipart.File, header *multipart.FileHeader, declaredProfile string) (*ImportBatchResponse, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
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

	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])

	existing, err := s.repo.FindBatchBySHA256(ctx, hash)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		resp := NewImportBatchResponse(*existing)
		return &resp, nil
	}

	if declaredProfile != "" {
		if declaredProfile != ProfileOwnerOutlet && declaredProfile != ProfileNonRegister {
			return nil, ErrUnknownProfile
		}
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

	code := fmt.Sprintf("IMPORT-%s-%s", time.Now().Format("20060102"), hash[:12])
	profileForRow := declaredProfile
	if profileForRow == "" {
		profileForRow = ProfilePendingDetection
	}

	batchID, err := s.repo.CreateBatch(ctx, code, profileForRow, header.Filename, hash, filePath, actor.ID)
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
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
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
	return &ImportBatchListResponse{Items: responses, Meta: ListMeta{Page: page, Limit: limit, Total: total}}, nil
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
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
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
	return &ImportRowListResponse{Items: responses, Meta: ListMeta{Page: page, Limit: limit, Total: total}}, nil
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
	if batch.Status != BatchStatusValidated {
		return nil, ErrInvalidBatchStatus
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
