package seeder

import (
	"testing"
	"time"
)

func TestSheetMonthFromName(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{"01", 1},
		{"12", 12},
		{"00", 0},
		{"13", 0},
		{"06 old", 0},
		{"Copy of 06", 0},
		{"MEI", 0},
		{"01 (2026)", 0},
		{"Saldo", 0},
		{"Daily Report", 0},
		{"Selisih Midtrans", 0},
		{" 07 ", 7},
	}
	for _, tc := range cases {
		if got := sheetMonthFromName(tc.name); got != tc.want {
			t.Errorf("sheetMonthFromName(%q) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSheetAllowed(t *testing.T) {
	cases := []struct {
		totalSheets int
		name        string
		want        bool
	}{
		{1, "New & Subscribe", true}, // 2023's single-sheet archive
		{13, "07", true},
		{13, "06 old", false},
		{13, "Copy of 06", false},
		{18, "01 (2026)", false},
		{15, "Saldo", false},
		{15, "Daily Report", false},
	}
	for _, tc := range cases {
		if got := sheetAllowed(tc.totalSheets, tc.name); got != tc.want {
			t.Errorf("sheetAllowed(%d, %q) = %v, want %v", tc.totalSheets, tc.name, got, tc.want)
		}
	}
}

func TestResolveFallbackDate(t *testing.T) {
	row := subscribeRow{Year: 2023, SheetMonth: 7}

	// A parseable candidate wins over the fallback.
	got := resolveFallbackDate(row, "", "15/07/23")
	want := time.Date(2023, 7, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("resolveFallbackDate with candidate = %v, want %v", got, want)
	}

	// All candidates blank -> anchor on the row's known file-year + sheet-month, never "now".
	got = resolveFallbackDate(row, "", "No Date")
	want = time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("resolveFallbackDate fallback = %v, want %v", got, want)
	}

	// Unknown sheet month (e.g. 2023's single unsplit sheet) defaults to January of that year.
	rowNoMonth := subscribeRow{Year: 2023, SheetMonth: 0}
	got = resolveFallbackDate(rowNoMonth)
	want = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("resolveFallbackDate with unknown month = %v, want %v", got, want)
	}
}

func TestHasPurchaseIntent(t *testing.T) {
	cases := []struct {
		name string
		row  subscribeRow
		want bool
	}{
		{"all blank", subscribeRow{}, false},
		{"activation date present", subscribeRow{DateAktivasi: "01/07/24"}, true},
		{"renewal date present", subscribeRow{DateRenewal: "01/07/24"}, true},
		{"expired date present", subscribeRow{DateExpired: "01/07/24"}, true},
		{"only 'No Date' text", subscribeRow{DateAktivasi: "No Date", DateRenewal: "No Date", DateExpired: "No Date"}, false},
		// The trap: Tenor/Paket/Status having lazy-default values must NOT influence this check.
		{"lazy defaults but no dates", subscribeRow{Tenor: "1", Paket: "Basic"}, false},
	}
	for _, tc := range cases {
		if got := hasPurchaseIntent(tc.row); got != tc.want {
			t.Errorf("%s: hasPurchaseIntent() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestExtractUniqueCode(t *testing.T) {
	cases := []struct {
		amount string
		want   string
	}{
		{"78012", "12"},
		{"78000", ""},
		{"0", ""},
		{"", ""},
		{"128123", "123"},
		{"not-a-number", ""},
	}
	for _, tc := range cases {
		if got := extractUniqueCode(tc.amount); got != tc.want {
			t.Errorf("extractUniqueCode(%q) = %q, want %q", tc.amount, got, tc.want)
		}
	}
}

func TestInferPaymentMethod(t *testing.T) {
	cases := []struct {
		name              string
		metode            string
		balanceSystem     string
		balanceTf         string
		wantMethod        string
		wantUniqueCode    string
	}{
		{"explicit midtrans label", "Midtrans", "78000", "78000", "MIDTRANS", ""},
		{"explicit TF/BRI label", "TF/BRI", "78012", "78000", "TF_BRI", "12"},
		{"explicit saldo aplikasi label", "Saldo Aplikasi", "", "", "SALDO_APLIKASI", ""},
		{"blank label, round balance -> midtrans", "", "78000", "78000", "MIDTRANS", ""},
		{"blank label, suffixed balance -> tf/bri", "", "78012", "60000", "TF_BRI", "12"},
		{"blank label, both balances blank -> saldo aplikasi", "", "", "", "SALDO_APLIKASI", ""},
		{"blank label, both balances zero -> saldo aplikasi", "", "0", "0", "SALDO_APLIKASI", ""},
	}
	for _, tc := range cases {
		method, uniqueCode := inferPaymentMethod(tc.metode, tc.balanceSystem, tc.balanceTf)
		if method != tc.wantMethod || uniqueCode != tc.wantUniqueCode {
			t.Errorf("%s: inferPaymentMethod(%q, %q, %q) = (%q, %q), want (%q, %q)",
				tc.name, tc.metode, tc.balanceSystem, tc.balanceTf, method, uniqueCode, tc.wantMethod, tc.wantUniqueCode)
		}
	}
}

func TestCleanAmountString(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"78,012", "78012"},
		{"78.012", "78012"},
		{" 78012 ", "78012"},
		{"", ""},
		{"0", "0"},
		{"-", ""},
		{"No Date", ""},
		{"Rp 78.012", ""},
	}
	for _, tc := range cases {
		if got := cleanAmountString(tc.in); got != tc.want {
			t.Errorf("cleanAmountString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
