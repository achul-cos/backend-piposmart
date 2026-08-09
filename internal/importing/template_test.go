package importing

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestBuildOwnerImportTemplateXLSXHeaders(t *testing.T) {
	data, err := buildOwnerImportTemplateXLSX()
	if err != nil {
		t.Fatalf("buildOwnerImportTemplateXLSX() error = %v", err)
	}

	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("excelize.OpenReader() error = %v", err)
	}

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		t.Fatalf("GetRows() error = %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("template has no rows")
	}
	if len(rows[0]) != len(ownerImportTemplateHeaders) {
		t.Fatalf("len(header row) = %d, want %d", len(rows[0]), len(ownerImportTemplateHeaders))
	}
	for i, want := range ownerImportTemplateHeaders {
		if rows[0][i] != want {
			t.Fatalf("header[%d] = %q, want %q", i, rows[0][i], want)
		}
	}
}
