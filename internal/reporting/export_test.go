package reporting

import "testing"

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
