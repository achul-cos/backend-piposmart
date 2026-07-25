package importing

import "testing"

// Real header rows from c:/piposmart/data_admin/ (see Sprint 14 plan) — used verbatim as fixtures
// so profile detection is tested against the actual workbook shapes, not invented ones.
var ownerOutletHeaderRow = []string{
	"No", "Date of Work", "Nama Penginput", "Kategori Akun", "Kode Baris", "Kode Owner",
	"Nama Owner ", "Email Owner", "No Hp Owner", "No. Hp Outlet", "Create Date Project", "Bulan",
	"Nama Project/BRAND", "Nama Outlet", "Kelurahan ", "Kecamatan", "Kota", "Provinsi",
	"Alamat Lengkap", "Nama Pengisi", "Check", "STATUS TERBARU", "Akuisisi", "PIC",
	"Tanggal Berlangganan", "Booking", "Mitra",
}

var nonRegisterTitleRow = []string{"DATA NON REGISTER BULAN JULI"}
var nonRegisterHeaderRow = []string{
	"Date Of Work", "Tanggal FU (Sales)", "PIC Follow up", "Kode OTP", "Nomor Telepon",
	"Created Date Kode OTP", "Status Nomor Telepon", "Status FU (Sales)", "Status Akun",
	"Tanggal Follow UP", "Remarks", "Status FU OTP (ADM)",
}

func TestDetectProfile_OwnerOutlet(t *testing.T) {
	rows := [][]string{ownerOutletHeaderRow, {"1", "46204", "Wilma", "Akun Baru"}}
	profile, headerRow, headers, err := detectProfile(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile != ProfileOwnerOutlet {
		t.Fatalf("expected %s, got %s", ProfileOwnerOutlet, profile)
	}
	if headerRow != 0 {
		t.Fatalf("expected header row 0, got %d", headerRow)
	}
	if _, ok := headers["KODE OWNER"]; !ok {
		t.Fatal("expected KODE OWNER in header index")
	}
}

func TestDetectProfile_NonRegister_HeaderNotOnFirstRow(t *testing.T) {
	// Mirrors the real workbook: row 1 is a merged title, real headers are row 2.
	rows := [][]string{nonRegisterTitleRow, nonRegisterHeaderRow, {"46204", "46204", "Intern", "0264", "6282125558700"}}
	profile, headerRow, _, err := detectProfile(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile != ProfileNonRegister {
		t.Fatalf("expected %s, got %s", ProfileNonRegister, profile)
	}
	if headerRow != 1 {
		t.Fatalf("expected header row 1 (0-indexed), got %d", headerRow)
	}
}

func TestDetectProfile_Ambiguous(t *testing.T) {
	if _, _, _, err := detectProfile([][]string{{"no data here"}}); err != ErrProfileRequired {
		t.Fatalf("expected ErrProfileRequired, got %v", err)
	}
}

func TestVerifyProfile_Mismatch(t *testing.T) {
	rows := [][]string{ownerOutletHeaderRow}
	if _, _, err := verifyProfile(rows, ProfileNonRegister); err != ErrProfileHeaderMismatch {
		t.Fatalf("expected ErrProfileHeaderMismatch, got %v", err)
	}
}

func TestVerifyProfile_UnknownProfile(t *testing.T) {
	if _, _, err := verifyProfile([][]string{ownerOutletHeaderRow}, "NOT_A_PROFILE"); err != ErrUnknownProfile {
		t.Fatalf("expected ErrUnknownProfile, got %v", err)
	}
}

// TestOTPColumnsNeverIndexed is the concrete guarantee behind "OTP mentah tidak tersimpan": OTP
// header names must never appear in the header index that drives row parsing, so no code path
// can ever read an OTP cell value into raw_payload.
func TestOTPColumnsNeverIndexed(t *testing.T) {
	idx := buildHeaderIndex(nonRegisterHeaderRow)
	for _, otpHeader := range []string{"KODE OTP", "CREATED DATE KODE OTP"} {
		if _, ok := idx[otpHeader]; ok {
			t.Fatalf("OTP header %q must never be indexed", otpHeader)
		}
	}
	if _, ok := idx["NOMOR TELEPON"]; !ok {
		t.Fatal("expected NOMOR TELEPON to still be indexed (non-OTP column)")
	}
}

func TestParseOwnerOutletRow_Valid(t *testing.T) {
	idx := buildHeaderIndex(ownerOutletHeaderRow)
	row := make([]string, len(ownerOutletHeaderRow))
	row[idx["KODE OWNER"]] = "19126"
	row[idx["NAMA OWNER"]] = "arkhan"
	row[idx["EMAIL OWNER"]] = "adychandra333@gmail.com"
	row[idx["NO HP OWNER"]] = "082387091945"
	row[idx["NO. HP OUTLET"]] = "082387091945"
	row[idx["NAMA PROJECT/BRAND"]] = "dlaundry"
	row[idx["NAMA OUTLET"]] = "Suka Mulia"
	row[idx["KOTA"]] = "Kota Pekanbaru"
	row[idx["PROVINSI"]] = "Riau"
	row[idx["ALAMAT LENGKAP"]] = "jln Diponegoro VII no 1a"

	parsed, errs := parseOwnerOutletRow(row, idx)
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}
	if parsed.OwnerCode != "19126" || parsed.OwnerName != "arkhan" {
		t.Fatalf("unexpected parsed row: %+v", parsed)
	}
	if parsed.OwnerPhone == "" {
		t.Fatal("expected owner phone to be normalized, got empty")
	}
}

func TestParseOwnerOutletRow_MissingRequiredFields(t *testing.T) {
	idx := buildHeaderIndex(ownerOutletHeaderRow)
	row := make([]string, len(ownerOutletHeaderRow))
	// Everything left blank.
	_, errs := parseOwnerOutletRow(row, idx)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for a fully blank row")
	}
}

func TestParseNonRegisterRow_InvalidPhone(t *testing.T) {
	idx := buildHeaderIndex(nonRegisterHeaderRow)
	row := make([]string, len(nonRegisterHeaderRow))
	row[idx["NOMOR TELEPON"]] = "Tidak Tersedia" // real dirty value seen in the source workbook
	row[idx["REMARKS"]] = "nasabah tidak merespon"

	_, errs := parseNonRegisterRow(row, idx)
	if len(errs) == 0 {
		t.Fatal("expected a validation error for an unparseable phone number")
	}
}

func TestParseNonRegisterRow_Valid(t *testing.T) {
	idx := buildHeaderIndex(nonRegisterHeaderRow)
	row := make([]string, len(nonRegisterHeaderRow))
	row[idx["NOMOR TELEPON"]] = "6282125558700"
	row[idx["REMARKS"]] = "greeting"

	parsed, errs := parseNonRegisterRow(row, idx)
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}
	if parsed.Phone == "" {
		t.Fatal("expected normalized phone")
	}
}

func TestNormalizeExcelDate(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"excel serial number", "46204", "2026-07-01", false},
		{"iso date passthrough", "2026-07-01", "2026-07-01", false},
		{"empty is not an error", "", "", false},
		{"garbage", "not a date", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeExcelDate(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("normalizeExcelDate(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestIsBlankRow(t *testing.T) {
	if !isBlankRow([]string{"", " ", "\t"}) {
		t.Fatal("expected row of blanks to be blank")
	}
	if isBlankRow([]string{"", "x", ""}) {
		t.Fatal("expected row with content to not be blank")
	}
}

func TestJoinNonEmpty(t *testing.T) {
	got := joinNonEmpty("Jl. Merdeka", "", "Kel. Sail", "  ")
	want := "Jl. Merdeka, Kel. Sail"
	if got != want {
		t.Fatalf("joinNonEmpty = %q, want %q", got, want)
	}
}
