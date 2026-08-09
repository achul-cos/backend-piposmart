package seeder

import (
	"database/sql"
	"testing"
	"time"

	"backend_crm_piposmart/internal/platform/factory"
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

func TestSplitProjectOutlet(t *testing.T) {
	cases := []struct {
		raw       string
		wantBrand string
		wantName  string
	}{
		{"D'java Laundry/Cabang Ambon", "D'java Laundry", "Cabang Ambon"},
		{"Solo Laundry", "", "Solo Laundry"},
		{"", "", ""},
		{"  Brand / Outlet  ", "Brand", "Outlet"},
		{"OnlySlash/", "OnlySlash", "OnlySlash"}, // blank outlet half falls back to the brand half
	}
	for _, tc := range cases {
		brand, name := splitProjectOutlet(tc.raw)
		if brand != tc.wantBrand || name != tc.wantName {
			t.Errorf("splitProjectOutlet(%q) = (%q, %q), want (%q, %q)", tc.raw, brand, name, tc.wantBrand, tc.wantName)
		}
	}
}

func TestNormalizePhoneForMatch(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0896891123", "896891123"},
		{"62896891123", "896891123"},
		{"896891123", "896891123"},
		{"0896-891-123", "896891123"},
		{"+62 896 891 123", "896891123"},
	}
	for _, tc := range cases {
		if got := normalizePhoneForMatch(tc.in); got != tc.want {
			t.Errorf("normalizePhoneForMatch(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDeriveFallbackOwnerCode(t *testing.T) {
	cases := []struct {
		name  string
		phone string
		idx   int
		want  string
	}{
		{"Azizah", "0896891123", 5, "OWN-0896891123"},
		{"D'java Laundry Cabang Ambon", "", 5, "OWN-D'JAVALAUN"},
		{"", "", 5, "OWN-NS-00005"},
	}
	for _, tc := range cases {
		if got := deriveFallbackOwnerCode(tc.name, tc.phone, tc.idx); got != tc.want {
			t.Errorf("deriveFallbackOwnerCode(%q, %q, %d) = %q, want %q", tc.name, tc.phone, tc.idx, got, tc.want)
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

func TestShouldSeedTransferForTopup(t *testing.T) {
	cases := []struct {
		name  string
		topup factory.WalletTopup
		want  bool
	}{
		{
			name: "manual transfer topup",
			topup: factory.WalletTopup{
				PaymentMethod:  "TF_BRI",
				PaymentChannel: "TF/BRI",
			},
			want: true,
		},
		{
			name: "midtrans topup still gets transfer record for imported proof",
			topup: factory.WalletTopup{
				PaymentMethod:  "MIDTRANS",
				PaymentChannel: "Midtrans",
			},
			want: true,
		},
		{
			name: "saldo aplikasi synthetic topup skips transfer",
			topup: factory.WalletTopup{
				PaymentMethod:  "SALDO_APLIKASI",
				PaymentChannel: "Saldo Aplikasi",
			},
			want: false,
		},
	}

	for _, tc := range cases {
		if got := shouldSeedTransferForTopup(tc.topup); got != tc.want {
			t.Errorf("%s: shouldSeedTransferForTopup() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestBuildMatchedTransferForSubscribeTopup(t *testing.T) {
	paidAt := time.Date(2026, 7, 14, 9, 0, 0, 0, time.UTC)
	transferAt := time.Date(2026, 7, 13, 16, 30, 0, 0, time.UTC)
	topup := factory.WalletTopup{
		Amount:            "78000",
		PaidAt:            paidAt,
		TransferDateAt:    transferAt,
		Note:              "Import dari Excel New & Subscribe",
		PaymentMethod:     "TF_BRI",
		PaymentChannel:    "TF/BRI",
		ExternalReference: "EXCEL-TF-2026-77",
	}

	got := buildMatchedTransferForSubscribeTopup("2026-77", topup, 9123)

	if got.Amount != topup.Amount {
		t.Fatalf("Amount = %q, want %q", got.Amount, topup.Amount)
	}
	if !got.TransferDate.Equal(transferAt) {
		t.Fatalf("TransferDate = %v, want %v", got.TransferDate, transferAt)
	}
	if got.MatchStatus != "MATCHED" {
		t.Fatalf("MatchStatus = %q, want MATCHED", got.MatchStatus)
	}
	if got.ExternalReference != "EXCEL-TRF-2026-77" {
		t.Fatalf("ExternalReference = %q, want %q", got.ExternalReference, "EXCEL-TRF-2026-77")
	}
	wantMatched := sql.NullInt64{Int64: 9123, Valid: true}
	if got.MatchedWalletPaymentID != wantMatched {
		t.Fatalf("MatchedWalletPaymentID = %+v, want %+v", got.MatchedWalletPaymentID, wantMatched)
	}
	if got.Note == "" {
		t.Fatal("Note should not be blank")
	}
}

func TestBuildMatchedTransferForSubscribeTopupFallsBackToPaidAt(t *testing.T) {
	paidAt := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	topup := factory.WalletTopup{
		Amount: "120000",
		PaidAt: paidAt,
	}

	got := buildMatchedTransferForSubscribeTopup("2026-15", topup, 77)
	if !got.TransferDate.Equal(paidAt) {
		t.Fatalf("TransferDate = %v, want %v", got.TransferDate, paidAt)
	}
	if got.Note == "" {
		t.Fatal("Note should fall back to default text")
	}
}
