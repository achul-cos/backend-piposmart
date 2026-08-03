package reporting

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

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

func buildXLSX(sheetName string, columns []ReportColumn, items []map[string]any) ([]byte, error) {
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
