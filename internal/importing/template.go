package importing

import (
	"bytes"

	"github.com/xuri/excelize/v2"
)

var ownerImportTemplateHeaders = []string{
	"KODE",
	"NAMA",
	"TELEPON",
	"EMAIL",
	"NAMA_BRAND",
	"PROVINSI",
	"KOTA",
	"ALAMAT",
}

func buildOwnerImportTemplateXLSX() ([]byte, error) {
	file := excelize.NewFile()
	sheet := file.GetSheetName(0)

	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"C92C1E"}},
	})

	for idx, header := range ownerImportTemplateHeaders {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		_ = file.SetCellValue(sheet, cell, header)
		_ = file.SetCellStyle(sheet, cell, cell, headerStyle)
		_ = file.SetColWidth(sheet, cell[:1], cell[:1], 20)
	}

	_ = file.SetRowHeight(sheet, 1, 24)
	_ = file.SetPanes(sheet, &excelize.Panes{
		Freeze: true,
		Split:  false,
		XSplit: 0,
		YSplit: 1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
