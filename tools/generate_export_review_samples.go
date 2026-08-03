package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/database"
	"backend_crm_piposmart/internal/platform/logging"
	"backend_crm_piposmart/internal/reporting"

	_ "github.com/go-sql-driver/mysql"
)

type reportConfig struct {
	ReportKey string
	FileBase  string
	WritePDF  bool
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	logger, err := logging.New(cfg.Log)
	if err != nil {
		panic(fmt.Errorf("init logging: %w", err))
	}
	slog.SetDefault(logger)

	ctx := context.Background()
	conn, err := database.Open(ctx, cfg.Database, logger)
	if err != nil {
		panic(fmt.Errorf("connect db: %w", err))
	}
	defer conn.Close()

	repo := reporting.NewRepository(conn.SQLDB())

	outDir := filepath.Join("storage", "exports", "review_samples")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(fmt.Errorf("create output directory: %w", err))
	}

	reports := []reportConfig{
		{
			ReportKey: reporting.ReportAdminOwnerOutlet,
			FileBase:  "admin_owner_outlet_review_sample",
			WritePDF:  true,
		},
		{
			ReportKey: reporting.ReportAdminNewSubscribe,
			FileBase:  "admin_new_subscribe_review_sample",
			WritePDF:  false,
		},
		{
			ReportKey: reporting.ReportAdminNasabahProvinsi,
			FileBase:  "admin_nasabah_baru_provinsi_review_sample",
			WritePDF:  false,
		},
	}

	actor := reporting.Actor{
		RoleCode: reporting.RoleAdmin,
	}

	params := reporting.ListReportsParams{
		DateFrom: "2026-07-01",
		DateTo:   "2026-08-31",
		All:      true,
	}

	for _, rpt := range reports {
		data, err := repo.ListReport(ctx, actor, rpt.ReportKey, params)
		if err != nil {
			panic(fmt.Errorf("list report %s: %w", rpt.ReportKey, err))
		}

		// Generate CSV
		csvContent, err := reporting.BuildCSV(data.Columns, data.Items)
		if err != nil {
			panic(fmt.Errorf("build CSV for %s: %w", rpt.ReportKey, err))
		}
		csvPath := filepath.Join(outDir, rpt.FileBase+".csv")
		if err := os.WriteFile(csvPath, csvContent, 0o644); err != nil {
			panic(fmt.Errorf("write CSV to %s: %w", csvPath, err))
		}
		fmt.Printf("Generated CSV: %s (%d rows)\n", csvPath, len(data.Items))

		// Generate XLSX
		xlsxContent, err := reporting.BuildXLSX(rpt.ReportKey, "Report", data.Columns, data.Items, data.Insight)
		if err != nil {
			panic(fmt.Errorf("build XLSX for %s: %w", rpt.ReportKey, err))
		}
		xlsxPath := filepath.Join(outDir, rpt.FileBase+".xlsx")
		if err := os.WriteFile(xlsxPath, xlsxContent, 0o644); err != nil {
			panic(fmt.Errorf("write XLSX to %s: %w", xlsxPath, err))
		}
		fmt.Printf("Generated XLSX: %s (%d rows)\n", xlsxPath, len(data.Items))

		// Generate PDF if needed
		if rpt.WritePDF {
			pdfContent, err := reporting.BuildPDF(rpt.ReportKey, data.Columns, data.Items, data.Insight)
			if err != nil {
				panic(fmt.Errorf("build PDF for %s: %w", rpt.ReportKey, err))
			}
			pdfPath := filepath.Join(outDir, rpt.FileBase+".pdf")
			if err := os.WriteFile(pdfPath, pdfContent, 0o644); err != nil {
				panic(fmt.Errorf("write PDF to %s: %w", pdfPath, err))
			}
			fmt.Printf("Generated PDF: %s (%d rows)\n", pdfPath, len(data.Items))
		}
	}

	fmt.Println("Review sample exports successfully generated with real data in", outDir)
}
