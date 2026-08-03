package reporting

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

func sanitizeSpreadsheetCell(value any) any {
	text := fmt.Sprint(value)
	if text == "" {
		return text
	}
	switch text[0] {
	case '=', '+', '-', '@':
		return "'" + text
	}
	if strings.HasPrefix(text, "\t") || strings.HasPrefix(text, "\r") {
		return "'" + text
	}
	return text
}

func buildCSV(columns []ReportColumn, items []map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	headers := make([]string, 0, len(columns))
	for _, column := range columns {
		headers = append(headers, column.Label)
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}
	for _, item := range items {
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			row = append(row, fmt.Sprint(sanitizeSpreadsheetCell(item[column.Key])))
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildXLSX(reportKey, sheetName string, columns []ReportColumn, items []map[string]any, insight map[string]any) ([]byte, error) {
	if reportKey == ReportAdminOwnerOutlet {
		return buildAdminOwnerOutletXLSX(sheetName, columns, items)
	}
	file := excelize.NewFile()
	defaultSheet := file.GetSheetName(0)
	file.SetSheetName(defaultSheet, sheetName)
	for idx, column := range columns {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		_ = file.SetCellValue(sheetName, cell, column.Label)
	}
	for rowIndex, item := range items {
		for colIndex, column := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			_ = file.SetCellValue(sheetName, cell, sanitizeSpreadsheetCell(item[column.Key]))
		}
	}
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildAdminOwnerOutletXLSX(sheetName string, columns []ReportColumn, items []map[string]any) ([]byte, error) {
	file := excelize.NewFile()
	defaultSheet := file.GetSheetName(0)
	file.SetSheetName(defaultSheet, sheetName)

	titleStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "C92C1E"},
	})
	metaLabelStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	metaValueStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10},
	})
	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C92C1E"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})
	bodyStyle, _ := file.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "EAEAEA", Style: 1},
			{Type: "right", Color: "EAEAEA", Style: 1},
			{Type: "top", Color: "EAEAEA", Style: 1},
			{Type: "bottom", Color: "EAEAEA", Style: 1},
		},
	})

	_ = file.SetRowHeight(sheetName, 1, 28)
	for row := 2; row <= 6; row++ {
		_ = file.SetRowHeight(sheetName, row, 20)
	}

	_ = file.MergeCell(sheetName, "C1", "H1")
	_ = file.SetCellValue(sheetName, "C1", "Report Owner & Outlet")
	_ = file.SetCellStyle(sheetName, "C1", "H1", titleStyle)

	_ = file.SetCellValue(sheetName, "C3", "Tanggal Export")
	_ = file.SetCellStyle(sheetName, "C3", "C3", metaLabelStyle)
	_ = file.SetCellValue(sheetName, "D3", time.Now().In(time.Local).Format("02/01/2006"))
	_ = file.SetCellStyle(sheetName, "D3", "D3", metaValueStyle)

	_ = file.SetCellValue(sheetName, "C4", "Total Baris")
	_ = file.SetCellStyle(sheetName, "C4", "C4", metaLabelStyle)
	_ = file.SetCellValue(sheetName, "D4", len(items))
	_ = file.SetCellStyle(sheetName, "D4", "D4", metaValueStyle)

	if logoPath := findPiposmartLogoPath(); logoPath != "" {
		_ = file.AddPicture(sheetName, "A1", logoPath, &excelize.GraphicOptions{
			AutoFit:     false,
			ScaleX:      0.55,
			ScaleY:      0.55,
			Positioning: "oneCell",
		})
	}

	headerRow := 7
	for idx, column := range columns {
		cell, _ := excelize.CoordinatesToCellName(idx+1, headerRow)
		_ = file.SetCellValue(sheetName, cell, column.Label)
		_ = file.SetCellStyle(sheetName, cell, cell, headerStyle)
		colName, _ := excelize.ColumnNumberToName(idx + 1)
		_ = file.SetColWidth(sheetName, colName, colName, adminOwnerOutletColumnWidth(column.Key))
	}
	_ = file.SetRowHeight(sheetName, headerRow, 34)

	for rowIndex, item := range items {
		for colIndex, column := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+headerRow+1)
			_ = file.SetCellValue(sheetName, cell, sanitizeSpreadsheetCell(item[column.Key]))
			_ = file.SetCellStyle(sheetName, cell, cell, bodyStyle)
		}
	}

	lastColumn, _ := excelize.ColumnNumberToName(len(columns))
	_ = file.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      headerRow,
		TopLeftCell: fmt.Sprintf("A%d", headerRow+1),
		ActivePane:  "bottomLeft",
	})
	if len(items) > 0 {
		_ = file.SetCellStyle(sheetName, "A8", fmt.Sprintf("%s%d", lastColumn, len(items)+headerRow), bodyStyle)
	}
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func findPiposmartLogoPath() string {
	candidates := []string{
		filepath.Join("asset", "piposmart-vertical.png"),
		filepath.Join("assets", "piposmart-vertical.png"),
		filepath.Join("backend_crm_piposmart", "asset", "piposmart-vertical.png"),
		filepath.Join("backend_crm_piposmart", "assets", "piposmart-vertical.png"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func adminOwnerOutletColumnWidth(key string) float64 {
	switch key {
	case "owner_year", "outlet_year":
		return 12
	case "owner_month", "outlet_month":
		return 16
	case "owner_date", "outlet_date", "shared_at", "subscription_started_at", "subscription_ended_at":
		return 20
	case "owner_code":
		return 16
	case "owner_email":
		return 26
	case "owner_phone", "outlet_phone":
		return 18
	case "brand_name", "outlet_name", "pic_lead", "subscription_package_name":
		return 24
	case "outlet_kelurahan", "outlet_kecamatan", "outlet_city", "outlet_province", "subscription_status":
		return 18
	case "outlet_address":
		return 34
	case "subscription_tenor":
		return 14
	default:
		return 20
	}
}

func buildPDF(title string, columns []ReportColumn, items []map[string]any, insight map[string]any) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(8, 8, 8)
	pdf.SetAutoPageBreak(true, 8)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 8, strings.ToUpper(strings.ReplaceAll(title, "_", " ")), "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	if insight != nil {
		dateFrom := fmt.Sprint(insight["date_from"])
		dateTo := fmt.Sprint(insight["date_to"])
		count := fmt.Sprint(insight["count"])
		pdf.CellFormat(0, 6, fmt.Sprintf("Periode: %s s/d %s | Total baris: %s", dateFrom, dateTo, count), "", 1, "L", false, 0, "")
	}
	pdf.Ln(2)

	colWidths := estimatePDFColumnWidths(columns)
	pdf.SetFont("Arial", "B", 8)
	for i, column := range columns {
		pdf.CellFormat(colWidths[i], 7, column.Label, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 8)
	for _, item := range items {
		for i, column := range columns {
			value := normalizePDFText(item[column.Key])
			pdf.CellFormat(colWidths[i], 6, value, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func estimatePDFColumnWidths(columns []ReportColumn) []float64 {
	totalWidth := 281.0
	widths := make([]float64, len(columns))
	if len(columns) == 0 {
		return widths
	}
	base := totalWidth / float64(len(columns))
	for i, column := range columns {
		width := base
		switch column.Type {
		case "datetime":
			width = 28
		case "date":
			width = 22
		case "currency":
			width = 24
		case "number":
			width = 18
		default:
			width = base
		}
		if len(column.Label) > 18 {
			width += 8
		}
		widths[i] = width
	}
	sum := 0.0
	for _, width := range widths {
		sum += width
	}
	if sum <= totalWidth {
		return widths
	}
	ratio := totalWidth / sum
	for i := range widths {
		widths[i] = widths[i] * ratio
	}
	return widths
}

func normalizePDFText(value any) string {
	text := fmt.Sprint(value)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	if len(text) > 40 {
		return text[:37] + "..."
	}
	return text
}
