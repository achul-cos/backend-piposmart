package reporting

import (
	"fmt"
	"testing"
)

func TestSanitizeSpreadsheetCell(t *testing.T) {
	cases := map[string]string{
		"=SUM(A1:A2)": "'=SUM(A1:A2)",
		"+cmd":        "'+cmd",
		"-danger":     "'-danger",
		"@formula":    "'@formula",
		"normal":      "normal",
	}

	for input, want := range cases {
		got := sanitizeSpreadsheetCell(input)
		if got != want {
			t.Fatalf("sanitizeSpreadsheetCell(%q) = %v, want %q", input, got, want)
		}
	}
}

func TestGetAdminOwnerOutletColumnsIncludesShareColumnsUntilFive(t *testing.T) {
	columns := GetAdminOwnerOutletColumns()
	labels := make(map[string]bool, len(columns))
	for _, column := range columns {
		labels[column.Label] = true
	}

	for index := 1; index <= adminOwnerOutletShareExportMax; index++ {
		for _, label := range []string{
			fmt.Sprintf("Tanggal Dibagikan %d", index),
			fmt.Sprintf("Share %d", index),
		} {
			if !labels[label] {
				t.Fatalf("missing export column label %q", label)
			}
		}
	}
	// "Kategori Nasabah" repeats for each share group (no number suffix, matching reference Excel)
	if !labels["Kategori Nasabah"] {
		t.Fatal("missing export column label \"Kategori Nasabah\"")
	}
}
