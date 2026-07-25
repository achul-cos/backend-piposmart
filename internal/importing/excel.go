package importing

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/customer"

	"github.com/xuri/excelize/v2"
)

// maxHeaderScanRows bounds how many leading rows we search for a header row. The real "Non
// Register" workbook has a merged title in row 1 and real headers in row 2 — scanning a small
// window handles that without hardcoding a row number per profile.
const maxHeaderScanRows = 5

type profileSpec struct {
	Name          string
	MarkerHeaders []string // normalized (trimmed+upper); ALL must be present in a row for a match
}

// knownProfiles' marker headers are taken directly from the real workbooks at
// c:/piposmart/data_admin/ (01. Owner & Outlet, 04. Data Belum Registrasi) — see Sprint 14 plan.
var knownProfiles = []profileSpec{
	{Name: ProfileOwnerOutlet, MarkerHeaders: []string{"KODE OWNER", "NAMA OWNER", "NAMA OUTLET"}},
	{Name: ProfileNonRegister, MarkerHeaders: []string{"NOMOR TELEPON", "KODE OTP", "STATUS AKUN"}},
}

// otpHeaderDenylist columns are never read into any row struct, regardless of profile — this is
// the concrete mechanism behind "OTP mentah tidak tersimpan": the raw OTP value never enters Go
// memory as parsed data, let alone raw_payload/JSON.
var otpHeaderDenylist = map[string]bool{
	"KODE OTP":              true,
	"CREATED DATE KODE OTP": true,
	"OTP":                   true,
}

func normalizeHeader(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// headerIndex maps a normalized header name to its 0-based column index for one row.
type headerIndex map[string]int

// buildFullHeaderIndex indexes every header, including OTP-named ones. Used ONLY for profile
// marker matching (the presence of a "Kode OTP" column is itself the signal that distinguishes
// the Non-Register workbook) — never handed to row parsing.
func buildFullHeaderIndex(row []string) headerIndex {
	idx := make(headerIndex, len(row))
	for i, cell := range row {
		h := normalizeHeader(cell)
		if h == "" {
			continue
		}
		if _, exists := idx[h]; !exists {
			idx[h] = i
		}
	}
	return idx
}

// buildHeaderIndex is the safe index used for actual value extraction (cellValue): OTP-named
// columns are removed here so no parse function can ever retrieve one, even by mistake — this is
// the enforcement point behind "OTP mentah tidak tersimpan", independent of buildFullHeaderIndex
// above which exists purely for detecting the profile.
func buildHeaderIndex(row []string) headerIndex {
	idx := make(headerIndex, len(row))
	for h, col := range buildFullHeaderIndex(row) {
		if otpHeaderDenylist[h] {
			continue
		}
		idx[h] = col
	}
	return idx
}

func (h headerIndex) hasAll(markers []string) bool {
	for _, m := range markers {
		if _, ok := h[m]; !ok {
			return false
		}
	}
	return true
}

// findHeaderRow scans the first maxHeaderScanRows rows for one containing all requiredMarkers
// (matched against the full, unfiltered header set), returning the safe (OTP-stripped) index for
// the matched row.
func findHeaderRow(rows [][]string, requiredMarkers []string) (rowIdx int, headers headerIndex, found bool) {
	limit := maxHeaderScanRows
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		full := buildFullHeaderIndex(rows[i])
		if full.hasAll(requiredMarkers) {
			return i, buildHeaderIndex(rows[i]), true
		}
	}
	return 0, nil, false
}

// detectProfile tries every known profile's markers, succeeding only if exactly one matches.
func detectProfile(rows [][]string) (profile string, headerRow int, headers headerIndex, err error) {
	type candidate struct {
		profile string
		row     int
		headers headerIndex
	}
	var matches []candidate
	for _, p := range knownProfiles {
		if row, idx, ok := findHeaderRow(rows, p.MarkerHeaders); ok {
			matches = append(matches, candidate{profile: p.Name, row: row, headers: idx})
		}
	}
	if len(matches) != 1 {
		return "", 0, nil, ErrProfileRequired
	}
	return matches[0].profile, matches[0].row, matches[0].headers, nil
}

// verifyProfile checks a user-declared profile's markers are present, without trying others.
func verifyProfile(rows [][]string, profile string) (headerRow int, headers headerIndex, err error) {
	for _, p := range knownProfiles {
		if p.Name == profile {
			row, idx, ok := findHeaderRow(rows, p.MarkerHeaders)
			if !ok {
				return 0, nil, ErrProfileHeaderMismatch
			}
			return row, idx, nil
		}
	}
	return 0, nil, ErrUnknownProfile
}

func cellValue(row []string, idx headerIndex, header string) string {
	col, ok := idx[normalizeHeader(header)]
	if !ok || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}

// normalizeExcelDate converts either an Excel serial-number string (the raw form Excel stores,
// e.g. "46204") or an already-formatted date string into ISO 8601 (YYYY-MM-DD). Satisfies Sprint
// 14's "Normalisasi Excel serial date" DoD item.
func normalizeExcelDate(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if serial, err := strconv.ParseFloat(raw, 64); err == nil {
		t, convErr := excelize.ExcelDateToTime(serial, false)
		if convErr != nil {
			return "", fmt.Errorf("invalid excel serial date %q: %w", raw, convErr)
		}
		return t.Format("2006-01-02"), nil
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "2/1/2006", "02/01/06", "2/1/06", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("unrecognized date format: %q", raw)
}

func joinNonEmpty(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, strings.TrimSpace(p))
		}
	}
	return strings.Join(kept, ", ")
}

/* ---------- Profile OWNER_OUTLET ---------- */

type ownerOutletRow struct {
	OwnerCode   string `json:"owner_code"`
	OwnerName   string `json:"owner_name"`
	OwnerEmail  string `json:"owner_email,omitempty"`
	OwnerPhone  string `json:"owner_phone"`
	BrandName   string `json:"brand_name,omitempty"`
	OutletName  string `json:"outlet_name"`
	OutletPhone string `json:"outlet_phone,omitempty"`
	City        string `json:"city,omitempty"`
	Province    string `json:"province,omitempty"`
	Address     string `json:"address,omitempty"`
	DateOfWork  string `json:"date_of_work,omitempty"`
}

func parseOwnerOutletRow(row []string, idx headerIndex) (ownerOutletRow, []string) {
	var errs []string
	r := ownerOutletRow{
		OwnerCode:  cellValue(row, idx, "Kode Owner"),
		OwnerName:  cellValue(row, idx, "Nama Owner"),
		OwnerEmail: cellValue(row, idx, "Email Owner"),
		BrandName:  cellValue(row, idx, "Nama Project/BRAND"),
		OutletName: cellValue(row, idx, "Nama Outlet"),
		City:       cellValue(row, idx, "Kota"),
		Province:   cellValue(row, idx, "Provinsi"),
		Address: joinNonEmpty(
			cellValue(row, idx, "Alamat Lengkap"),
			prefixIfPresent("Kel. ", cellValue(row, idx, "Kelurahan")),
			prefixIfPresent("Kec. ", cellValue(row, idx, "Kecamatan")),
		),
	}

	if r.OwnerCode == "" {
		errs = append(errs, "kode_owner wajib diisi")
	}
	if r.OwnerName == "" {
		errs = append(errs, "nama_owner wajib diisi")
	}
	if r.OutletName == "" {
		errs = append(errs, "nama_outlet wajib diisi")
	}

	if raw := cellValue(row, idx, "No Hp Owner"); raw != "" {
		phone, err := customer.NormalizePhone(raw)
		if err != nil {
			errs = append(errs, fmt.Sprintf("no_hp_owner tidak valid: %s", raw))
		} else {
			r.OwnerPhone = phone
		}
	} else {
		errs = append(errs, "no_hp_owner wajib diisi")
	}

	if raw := cellValue(row, idx, "No. Hp Outlet"); raw != "" {
		phone, err := customer.NormalizePhone(raw)
		if err != nil {
			errs = append(errs, fmt.Sprintf("no_hp_outlet tidak valid: %s", raw))
		} else {
			r.OutletPhone = phone
		}
	}

	// date_of_work is informational only (never drives owner/outlet creation) — a parse failure
	// is not fatal to the row, unlike the required/phone fields above.
	if raw := cellValue(row, idx, "Date of Work"); raw != "" {
		if d, err := normalizeExcelDate(raw); err == nil {
			r.DateOfWork = d
		}
	}

	return r, errs
}

func prefixIfPresent(prefix, value string) string {
	if value == "" {
		return ""
	}
	return prefix + value
}

/* ---------- Profile NON_REGISTER ---------- */

type nonRegisterRow struct {
	Phone      string `json:"phone"`
	Remarks    string `json:"remarks,omitempty"`
	StatusAkun string `json:"status_akun,omitempty"`
	DateOfWork string `json:"date_of_work,omitempty"`
}

func parseNonRegisterRow(row []string, idx headerIndex) (nonRegisterRow, []string) {
	var errs []string
	r := nonRegisterRow{
		Remarks:    cellValue(row, idx, "Remarks"),
		StatusAkun: cellValue(row, idx, "Status Akun"),
	}

	raw := cellValue(row, idx, "Nomor Telepon")
	if raw == "" {
		errs = append(errs, "nomor_telepon wajib diisi")
	} else {
		phone, err := customer.NormalizePhone(raw)
		if err != nil {
			errs = append(errs, fmt.Sprintf("nomor_telepon tidak valid: %s", raw))
		} else {
			r.Phone = phone
		}
	}

	// Informational only, non-fatal — see the identical comment in parseOwnerOutletRow.
	if raw := cellValue(row, idx, "Date Of Work"); raw != "" {
		if d, err := normalizeExcelDate(raw); err == nil {
			r.DateOfWork = d
		}
	}

	return r, errs
}

/* ---------- File reading ---------- */

// readRows opens an xlsx file and returns all rows of its first sheet.
func readRows(path string) ([][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("workbook has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read sheet %s: %w", sheets[0], err)
	}
	return rows, nil
}
