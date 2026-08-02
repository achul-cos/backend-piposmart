package reporting

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"backend_crm_piposmart/internal/platform/jobqueue"
)

func GenerateExportHandler(repo *Repository, service *Service) jobqueue.Handler {
	return func(ctx context.Context, tx *sql.Tx, jobID int64, payload json.RawMessage) error {
		var req GenerateExportJobPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return fmt.Errorf("reporting: parse export payload: %w", err)
		}
		if err := repo.MarkExportProcessing(ctx, repo.db, req.ExportID); err != nil {
			return err
		}
		if err := service.GenerateExport(ctx, req.ExportID); err != nil {
			_ = repo.MarkExportFailed(ctx, repo.db, req.ExportID, err)
			return err
		}
		_ = jobID
		return nil
	}
}
