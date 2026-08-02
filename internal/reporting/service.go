package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/jobqueue"
)

type Service struct {
	repo    *Repository
	jobs    *jobqueue.Repository
	storage config.StorageConfig
}

func NewService(repo *Repository, jobs *jobqueue.Repository, storage config.StorageConfig) *Service {
	return &Service{repo: repo, jobs: jobs, storage: storage}
}

func toActor(user identity.User) Actor {
	return Actor{
		ID:          user.ID,
		RoleCode:    user.RoleCode,
		Permissions: user.Permissions,
	}
}

func (s *Service) Dashboard(ctx context.Context, actor identity.User, params DashboardParams) (*DashboardResponse, error) {
	return s.repo.Dashboard(ctx, toActor(actor), params)
}

func (s *Service) ListReport(ctx context.Context, actor identity.User, reportKey string, params ListReportsParams) (*ReportListResponse, error) {
	return s.repo.ListReport(ctx, toActor(actor), reportKey, params)
}

func (s *Service) CreateExport(ctx context.Context, actor identity.User, req CreateExportRequest) (*ReportExportResponse, error) {
	format := strings.ToUpper(strings.TrimSpace(req.Format))
	if format != ExportFormatCSV && format != ExportFormatXLSX {
		return nil, ErrInvalidFormat
	}
	if _, err := reportColumns(req.ReportKey); err != nil {
		return nil, err
	}

	code := fmt.Sprintf("RPT-%s-%d", time.Now().UTC().Format("20060102150405"), actor.ID)
	exportID, err := s.repo.CreateExport(ctx, code, req.ReportKey, format, req.Filters, actor.ID)
	if err != nil {
		return nil, err
	}
	jobID, err := s.jobs.Enqueue(ctx, JobTypeGenerateExport, GenerateExportJobPayload{ExportID: exportID}, &actor.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetExportJobID(ctx, exportID, jobID); err != nil {
		return nil, err
	}
	item, err := s.repo.GetExportByID(ctx, exportID)
	if err != nil {
		return nil, err
	}
	resp := toExportResponse(*item)
	return &resp, nil
}

func (s *Service) ListExports(ctx context.Context, actor identity.User, page, limit int) (*ReportExportListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.ListExports(ctx, toActor(actor), page, limit)
}

func (s *Service) GetExport(ctx context.Context, actor identity.User, exportID int64) (*ReportExportResponse, error) {
	item, err := s.repo.GetExportByID(ctx, exportID)
	if err != nil {
		return nil, err
	}
	if !canReadAll(toActor(actor)) && (!item.RequestedByUserID.Valid || item.RequestedByUserID.Int64 != actor.ID) {
		return nil, ErrForbidden
	}
	resp := toExportResponse(*item)
	return &resp, nil
}

type DownloadFile struct {
	Path        string
	FileName    string
	ContentType string
	Content     []byte
}

func (s *Service) DownloadExport(ctx context.Context, actor identity.User, exportID int64) (*DownloadFile, error) {
	item, err := s.repo.GetExportFileSource(ctx, exportID)
	if err != nil {
		return nil, err
	}
	if !canReadAll(toActor(actor)) && (!item.RequestedByUserID.Valid || item.RequestedByUserID.Int64 != actor.ID) {
		return nil, ErrForbidden
	}
	if item.Status != ExportStatusCompleted {
		return nil, ErrExportNotReady
	}
	if item.FilePath.Valid && item.FilePath.String != "" {
		if _, err := os.Stat(item.FilePath.String); err == nil {
			return &DownloadFile{
				Path:        item.FilePath.String,
				FileName:    item.FileName.String,
				ContentType: item.MimeType.String,
			}, nil
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(item.FileBlob) == 0 {
		return nil, ErrExportNotReady
	}
	return &DownloadFile{
		FileName:    item.FileName.String,
		ContentType: item.MimeType.String,
		Content:     item.FileBlob,
	}, nil
}

func (s *Service) GenerateExport(ctx context.Context, exportID int64) error {
	item, err := s.repo.GetExportByID(ctx, exportID)
	if err != nil {
		return err
	}
	var filters map[string]string
	if item.FiltersJSON.Valid && item.FiltersJSON.String != "" {
		if err := json.Unmarshal([]byte(item.FiltersJSON.String), &filters); err != nil {
			return err
		}
	}
	params := ListReportsParams{
		DateFrom: filters["date_from"],
		DateTo:   filters["date_to"],
		Status:   filters["status"],
		Query:    filters["q"],
		Province: filters["province"],
		City:     filters["city"],
		All:      true,
	}
	actor := Actor{RoleCode: strings.ToUpper(item.RequestedByRole.String)}
	if item.RequestedByUserID.Valid {
		actor.ID = item.RequestedByUserID.Int64
	}
	if actor.RoleCode == "" {
		actor.RoleCode = RoleAdmin
	}
	report, err := s.repo.ListReport(ctx, actor, item.ReportKey, params)
	if err != nil {
		return err
	}
	exportDir := filepath.Join(s.storage.ExportDirectory, "reports")
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return err
	}
	var content []byte
	var mimeType, extension string
	fileBase := fmt.Sprintf("%s_%d", item.ReportKey, item.ID)
	switch item.Format {
	case ExportFormatCSV:
		content, err = buildCSV(report.Columns, report.Items)
		mimeType = "text/csv"
		extension = ".csv"
	case ExportFormatXLSX:
		content, err = buildXLSX("Report", report.Columns, report.Items)
		mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		extension = ".xlsx"
	default:
		return ErrInvalidFormat
	}
	if err != nil {
		return err
	}
	fileName := fileBase + extension
	filePath := filepath.Join(exportDir, fileName)
	storedPath := ""
	if err := os.WriteFile(filePath, content, 0o644); err == nil {
		storedPath = filePath
	}
	return s.repo.MarkExportCompleted(ctx, s.repo.db, exportID, storedPath, fileName, mimeType, content, int64(len(report.Items)))
}
