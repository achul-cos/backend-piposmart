package reporting

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

func sanitizeSpreadsheetCell(value any) any {
	if value == nil {
		return ""
	}
	text := fmt.Sprint(value)
	if text == "" || text == "<nil>" {
		return ""
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

func BuildCSV(columns []ReportColumn, items []map[string]any) ([]byte, error) {
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

func BuildXLSX(reportKey, sheetName string, columns []ReportColumn, items []map[string]any, insight map[string]any) ([]byte, error) {
	title := "Report Owner & Outlet"
	if reportKey == ReportAdminOwner {
		title = "Report Data Owner"
	}

	dateFrom := ""
	dateTo := ""
	if insight != nil {
		if df, ok := insight["date_from"].(string); ok {
			dateFrom = df
		}
		if dt, ok := insight["date_to"].(string); ok {
			dateTo = dt
		}
	}

	if reportKey == ReportAdminOwnerOutlet || reportKey == ReportAdminOwner {
		return BuildAdminOwnerOutletXLSX(title, sheetName, columns, items, dateFrom, dateTo)
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

const adminOwnerOutletShareExportMax = 10

func adminOwnerOutletShareColumns() []ReportColumn {
	columns := make([]ReportColumn, 0, adminOwnerOutletShareExportMax*3)
	for index := 1; index <= adminOwnerOutletShareExportMax; index++ {
		columns = append(columns,
			ReportColumn{Key: fmt.Sprintf("tanggal_dibagikan_%d", index), Label: fmt.Sprintf("Tanggal Dibagikan %d", index), Type: "string"},
			ReportColumn{Key: fmt.Sprintf("share_%d", index), Label: fmt.Sprintf("Share %d", index), Type: "string"},
			ReportColumn{Key: fmt.Sprintf("kategori_nasabah_%d", index), Label: fmt.Sprintf("Kategori Nasabah %d", index), Type: "string"},
		)
	}
	return columns
}

// GetAdminOwnerOutletColumns - Kolom lengkap untuk Data Owner-Outlet
func GetAdminOwnerOutletColumns() []ReportColumn {
	columns := []ReportColumn{
		{Key: "no", Label: "No", Type: "number"},
		{Key: "date_of_work", Label: "Date of Work", Type: "string"},
		{Key: "nama_penginput", Label: "Nama Penginput", Type: "string"},
		{Key: "kategori_akun", Label: "Kategori Akun", Type: "string"},
		{Key: "kode_baris", Label: "Kode Baris", Type: "string"},
		{Key: "owner_code", Label: "Kode Owner", Type: "string"},
		{Key: "owner_name", Label: "Nama Owner", Type: "string"},
		{Key: "owner_email", Label: "Email Owner", Type: "string"},
		{Key: "owner_phone", Label: "No Hp Owner", Type: "string"},
		{Key: "outlet_phone", Label: "No. Hp Outlet", Type: "string"},
		{Key: "create_date_project", Label: "Create Date", Type: "string"},
		{Key: "brand_name", Label: "Nama Project/BRAND", Type: "string"},
		{Key: "outlet_name", Label: "Nama Outlet", Type: "string"},
		{Key: "kelurahan", Label: "Kelurahan", Type: "string"},
		{Key: "kecamatan", Label: "Kecamatan", Type: "string"},
		{Key: "kota", Label: "Kota", Type: "string"},
		{Key: "provinsi", Label: "Provinsi", Type: "string"},
		{Key: "alamat_lengkap", Label: "Alamat Lengkap", Type: "string"},
		{Key: "status_terbaru", Label: "STATUS TERBARU", Type: "string"},
		{Key: "akuisisi", Label: "Akuisisi", Type: "string"},
		{Key: "pic", Label: "PIC", Type: "string"},
	}
	columns = append(columns, adminOwnerOutletShareColumns()...)
	columns = append(columns, ReportColumn{Key: "jumlah_outlet", Label: "Jumlah Outlet", Type: "number"})
	return columns
}

// GetAdminOwnerColumns - Kolom khusus Data Owner (Penginput, Kategori Akun, Hp Outlet, Bulan, & Outlet Name DAHUS/DIHAPUS)
func GetAdminOwnerColumns() []ReportColumn {
	return []ReportColumn{
		{Key: "no", Label: "No", Type: "number"},
		{Key: "date_of_work", Label: "Date of Work", Type: "string"},
		{Key: "kode_baris", Label: "Kode Baris", Type: "string"},
		{Key: "owner_code", Label: "Kode Owner", Type: "string"},
		{Key: "owner_name", Label: "Nama Owner", Type: "string"},
		{Key: "owner_email", Label: "Email Owner", Type: "string"},
		{Key: "owner_phone", Label: "No Hp Owner", Type: "string"},
		{Key: "create_date_project", Label: "Create Date Project", Type: "string"},
		{Key: "brand_name", Label: "Nama Project/BRAND", Type: "string"},
		{Key: "kelurahan", Label: "Kelurahan", Type: "string"},
		{Key: "kecamatan", Label: "Kecamatan", Type: "string"},
		{Key: "kota", Label: "Kota", Type: "string"},
		{Key: "provinsi", Label: "Provinsi", Type: "string"},
		{Key: "alamat_lengkap", Label: "Alamat Lengkap", Type: "string"},
		{Key: "jumlah_outlet", Label: "Jumlah Outlet", Type: "number"},
		{Key: "saldo_owner", Label: "Saldo Owner", Type: "currency"},
	}
}

func BuildAdminOwnerOutletXLSX(reportTitle, sheetName string, columns []ReportColumn, items []map[string]any, dateFrom, dateTo string) ([]byte, error) {
	file := excelize.NewFile()
	defaultSheet := file.GetSheetName(0)
	file.SetSheetName(defaultSheet, sheetName)

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
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "EAEAEA", Style: 1},
			{Type: "right", Color: "EAEAEA", Style: 1},
			{Type: "top", Color: "EAEAEA", Style: 1},
			{Type: "bottom", Color: "EAEAEA", Style: 1},
		},
	})
	numFmtStr := `"Rp "#,##0_-`
	currencyStyle, _ := file.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "EAEAEA", Style: 1},
			{Type: "right", Color: "EAEAEA", Style: 1},
			{Type: "top", Color: "EAEAEA", Style: 1},
			{Type: "bottom", Color: "EAEAEA", Style: 1},
		},
		CustomNumFmt: &numFmtStr,
	})
	centerStyle, _ := file.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "top"},
		Border: []excelize.Border{
			{Type: "left", Color: "EAEAEA", Style: 1},
			{Type: "right", Color: "EAEAEA", Style: 1},
			{Type: "top", Color: "EAEAEA", Style: 1},
			{Type: "bottom", Color: "EAEAEA", Style: 1},
		},
	})

	headerBgColor := "E8C2B9"

	headerBgStyle, _ := file.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerBgColor}},
	})
	titleStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 18, Color: "000000"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerBgColor}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	metaLabelStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: false, Size: 11, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerBgColor}},
	})
	metaValueStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "000000"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerBgColor}},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})

	for r := 1; r <= 6; r++ {
		_ = file.SetRowHeight(sheetName, r, 24)
	}

	_ = file.SetColWidth(sheetName, "A", "A", 12)
	_ = file.SetColWidth(sheetName, "B", "B", 12)
	_ = file.SetColWidth(sheetName, "C", "C", 12)
	_ = file.SetColWidth(sheetName, "D", "D", 12)
	_ = file.SetColWidth(sheetName, "E", "E", 16)
	_ = file.SetColWidth(sheetName, "F", "F", 24)

	lastDataCol, _ := excelize.ColumnNumberToName(len(columns))
	_ = file.SetCellStyle(sheetName, "A1", lastDataCol+"6", headerBgStyle)

	_ = file.MergeCell(sheetName, "A1", "D5")
	if logoPath := findPiposmartLogoPath(); logoPath != "" {
		_ = file.AddPicture(sheetName, "A1", logoPath, &excelize.GraphicOptions{
			AutoFit:     false,
			OffsetX:     6,
			OffsetY:     6,
			Positioning: "oneCell",
			ScaleX:      0.55,
			ScaleY:      0.55,
		})
	}

	_ = file.MergeCell(sheetName, "E1", "H2")
	_ = file.SetCellValue(sheetName, "E1", reportTitle)
	_ = file.SetCellStyle(sheetName, "E1", "H2", titleStyle)

	exportDateText := time.Now().In(time.Local).Format("02/01/2006")
	if dateFrom != "" && dateTo != "" {
		exportDateText = fmt.Sprintf("%s - %s", dateFrom, dateTo)
	} else if dateFrom != "" {
		exportDateText = fmt.Sprintf("Mulai %s", dateFrom)
	} else if dateTo != "" {
		exportDateText = fmt.Sprintf("s/d %s", dateTo)
	}

	_ = file.SetCellValue(sheetName, "E3", "Diekspor:")
	_ = file.SetCellStyle(sheetName, "E3", "E3", metaLabelStyle)
	_ = file.SetCellValue(sheetName, "F3", exportDateText)
	_ = file.SetCellStyle(sheetName, "F3", "F3", metaValueStyle)

	_ = file.SetCellValue(sheetName, "E4", "Total Baris:")
	_ = file.SetCellStyle(sheetName, "E4", "E4", metaLabelStyle)
	_ = file.SetCellValue(sheetName, "F4", len(items))
	_ = file.SetCellStyle(sheetName, "F4", "F4", metaValueStyle)

	headerRow := 7
	for idx, column := range columns {
		cell, _ := excelize.CoordinatesToCellName(idx+1, headerRow)
		_ = file.SetCellValue(sheetName, cell, column.Label)
		_ = file.SetCellStyle(sheetName, cell, cell, headerStyle)
		colName, _ := excelize.ColumnNumberToName(idx + 1)
		_ = file.SetColWidth(sheetName, colName, colName, adminOwnerOutletColumnWidth(column.Key))
	}
	_ = file.SetRowHeight(sheetName, headerRow, 34)

	firstCell, _ := excelize.CoordinatesToCellName(1, headerRow)
	lastCell, _ := excelize.CoordinatesToCellName(len(columns), headerRow)
	_ = file.AutoFilter(sheetName, firstCell+":"+lastCell, []excelize.AutoFilterOptions{})

	for rowIndex, item := range items {
		item["no"] = rowIndex + 1
		for colIndex, column := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+headerRow+1)
			val := item[column.Key]

			styleToUse := bodyStyle
			if column.Type == "currency" {
				styleToUse = currencyStyle
			} else if column.Key == "date_of_work" || column.Key == "create_date_project" {
				styleToUse = centerStyle
			}

			if column.Type == "currency" || column.Type == "number" {
				if valFloat, err := strconv.ParseFloat(fmt.Sprint(val), 64); err == nil {
					_ = file.SetCellValue(sheetName, cell, valFloat)
				} else {
					_ = file.SetCellValue(sheetName, cell, sanitizeSpreadsheetCell(val))
				}
			} else {
				_ = file.SetCellValue(sheetName, cell, sanitizeSpreadsheetCell(val))
			}
			_ = file.SetCellStyle(sheetName, cell, cell, styleToUse)
		}
	}

	_ = file.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      7,
		YSplit:      headerRow,
		TopLeftCell: fmt.Sprintf("H%d", headerRow+1),
		ActivePane:  "bottomRight",
	})

	lastColumn, _ := excelize.ColumnNumberToName(len(columns))
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
	case "no":
		return 5
	case "date_of_work", "create_date_project":
		return 13
	case "nama_penginput":
		return 16
	case "owner_name", "brand_name", "outlet_name":
		return 18
	case "kategori_akun", "kode_baris":
		return 13
	case "owner_code":
		return 14
	case "owner_email":
		return 22
	case "owner_phone", "outlet_phone":
		return 14
	case "kelurahan", "kecamatan":
		return 14
	case "kota", "provinsi":
		return 13
	case "alamat_lengkap":
		return 26
	case "jumlah_outlet":
		return 10
	case "saldo_owner":
		return 16
	default:
		return 14
	}
}

func BuildPDF(title string, columns []ReportColumn, items []map[string]any, insight map[string]any) ([]byte, error) {
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
