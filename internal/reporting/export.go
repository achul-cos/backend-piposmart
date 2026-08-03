package reporting

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

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

