package importing

import (
	"fmt"
	"regexp"
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
	{Name: ProfileNewSubscribe, MarkerHeaders: []string{"NOMINAL AKTIVASI", "PAKET MEMBERSHIP", "KODE OWNER"}},
	{Name: ProfileMonthlyActive, MarkerHeaders: []string{"KODE OWNER", "NAMA BRAND/OUTLET", "WILAYAH"}},
	{Name: ProfileBonusMitra, MarkerHeaders: []string{"MITRA", "KATEGORY MITRA", "STATUS PENCAIRAN KOMISI MITRA"}},
	{Name: ProfileSalesCallChat, MarkerHeaders: []string{"SCOR", "VALIDITAS", "FINALISASI (CLOSING)"}},
	{Name: ProfileSalesTarget, MarkerHeaders: []string{"TARGET USER", "TARGET OMSET", "BULAN"}},
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
	RowCode     string `json:"row_code,omitempty"`
	OutletName  string `json:"outlet_name"`
	OutletPhone string `json:"outlet_phone,omitempty"`
	City        string `json:"city,omitempty"`
	District    string `json:"district,omitempty"`
	SubDistrict string `json:"sub_district,omitempty"`
	Province    string `json:"province,omitempty"`
	Address     string `json:"address,omitempty"`
	DateOfWork  string `json:"date_of_work,omitempty"`
}

func parseOwnerOutletRow(row []string, idx headerIndex) (ownerOutletRow, []string) {
	var errs []string
	address := cellValue(row, idx, "Alamat Lengkap")
	if address == "" {
		address = joinNonEmpty(
			prefixIfPresent("Kel. ", cellValue(row, idx, "Kelurahan")),
			prefixIfPresent("Kec. ", cellValue(row, idx, "Kecamatan")),
		)
	}
	r := ownerOutletRow{
		OwnerCode:   cellValue(row, idx, "Kode Owner"),
		OwnerName:   cellValue(row, idx, "Nama Owner"),
		OwnerEmail:  cellValue(row, idx, "Email Owner"),
		BrandName:   cellValue(row, idx, "Nama Project/BRAND"),
		RowCode:     cellValue(row, idx, "Kode Baris"),
		OutletName:  cellValue(row, idx, "Nama Outlet"),
		City:        cellValue(row, idx, "Kota"),
		District:    cellValue(row, idx, "Kecamatan"),
		SubDistrict: cellValue(row, idx, "Kelurahan"),
		Province:    cellValue(row, idx, "Provinsi"),
		Address:     address,
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

/* ---------- Shared Sprint 15 helpers ---------- */

// noDateSentinel is the literal string the real workbooks use for an absent date, instead of a
// blank cell (confirmed against "02. New & Subscribe 2026" — e.g. "Date Top UP Struk" = "No Date"
// when no physical receipt was recorded).
const noDateSentinel = "no date"

// normalizeOptionalExcelDate treats both "" and the "No Date" sentinel as absent, returning ""
// with no error — used for informational-only date columns where a parse failure/absence must
// never fail the row (mirrors the existing date_of_work handling in parseOwnerOutletRow).
func normalizeOptionalExcelDate(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.ToLower(trimmed) == noDateSentinel {
		return ""
	}
	d, err := normalizeExcelDate(trimmed)
	if err != nil {
		return ""
	}
	return d
}

// normalizeMoneyString strips thousands-separator commas/spaces from a spreadsheet numeric cell
// (e.g. "78,000" -> "78000") and validates the result parses as a number. Money is kept as a
// string throughout (matching internal/wallet, internal/subscription, internal/kpi — no domain
// package uses float64 for currency), so no further conversion happens here.
// dotThousandsPattern matches a period followed by exactly 3 digits at the end of the string
// (optionally repeated, e.g. "2.078.000") — Indonesian-locale thousands grouping, as opposed to a
// genuine decimal point (which would have 1-2 digits after it, or not sit at the very end when
// repeated). Confirmed necessary against the real "Call & Chat-Lidya" sheet, which mixes both
// comma-grouped ("118,000") and period-grouped ("168.000", "2.078.000") cells for the same column
// — almost certainly an artifact of different machines' regional Excel settings when the sheet was
// filled in over time, not a deliberate distinction.
var dotThousandsPattern = regexp.MustCompile(`^\d{1,3}(\.\d{3})+$`)
var firstDigitsPattern = regexp.MustCompile(`\d+`)

// normalizeMoneyString strips thousands-separator punctuation from a spreadsheet numeric cell and
// validates the result parses as a number. Money is kept as a string throughout (matching
// internal/wallet, internal/subscription, internal/kpi — no domain package uses float64 for
// currency), so no further conversion happens here.
func normalizeMoneyString(raw string) (string, error) {
	trimmed := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	if trimmed == "" {
		return "0", nil
	}

	var cleaned string
	switch {
	case strings.Contains(trimmed, ","):
		// Comma present: comma is the thousands separator (US-style), any period is a genuine
		// decimal point.
		cleaned = strings.ReplaceAll(trimmed, ",", "")
	case dotThousandsPattern.MatchString(trimmed):
		// No comma, but periods sit in the classic 3-digit thousands-grouping shape: Indonesian
		// locale, strip them all.
		cleaned = strings.ReplaceAll(trimmed, ".", "")
	default:
		// No comma, and periods (if any) don't match the grouping shape — treat as a genuine
		// decimal (or no punctuation at all).
		cleaned = trimmed
	}

	if _, err := strconv.ParseFloat(cleaned, 64); err != nil {
		return "", fmt.Errorf("nominal tidak valid: %q", raw)
	}
	return cleaned, nil
}

// splitProjectOutlet splits the "Project/Outlet" and "Nama Brand/Outlet"-style combined columns
// (real format confirmed in both "02. New & Subscribe" and "05. Monthly Active") on the first "/".
// A value with no slash is treated as both project and outlet name (single-outlet owners omit the
// separator in some real rows).
func splitProjectOutlet(raw string) (project, outlet string) {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "/"); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+1:])
	}
	return raw, raw
}

func cellAt(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}

/* ---------- Profile NEW_SUBSCRIBE ---------- */

// newSubscribeRow is one wallet top-up + subscription activation/renewal event from
// "02. New & Subscribe 2026". Unlike OWNER_OUTLET/NON_REGISTER, the referenced owner must already
// exist (Sprint 15 profiles are transactional, not onboarding) — commitNewSubscribeRow looks it up
// by OwnerCode and marks the row UNMATCHED (reconciliation candidate) rather than auto-creating.
type newSubscribeRow struct {
	Kode             string `json:"kode"`
	OwnerCode        string `json:"owner_code"`
	OwnerName        string `json:"owner_name,omitempty"`
	OwnerOutletPhone string `json:"owner_outlet_phone,omitempty"`
	ProjectName      string `json:"project_name,omitempty"`
	OutletName       string `json:"outlet_name,omitempty"`
	City             string `json:"city,omitempty"`
	Province         string `json:"province,omitempty"`
	NominalAktivasi  string `json:"nominal_aktivasi"`
	TanggalAktivasi  string `json:"tanggal_aktivasi,omitempty"`
	BalanceTopUp     string `json:"balance_top_up"`
	Diskon           string `json:"diskon,omitempty"`
	FeeMidtrans      string `json:"fee_midtrans,omitempty"`
	Settlement       string `json:"settlement,omitempty"`
	MetodePembayaran string `json:"metode_pembayaran,omitempty"`
	DateOfRenewal    string `json:"date_of_renewal,omitempty"`
	ExpiredDate      string `json:"expired_date,omitempty"`
	TenorMonths      int    `json:"tenor_months"`
	PaketMembership  string `json:"paket_membership"`
	Status           string `json:"status,omitempty"`
}

func parseNewSubscribeRow(row []string, idx headerIndex) (newSubscribeRow, []string) {
	var errs []string
	project, outlet := splitProjectOutlet(cellValue(row, idx, "Project/Outlet"))
	r := newSubscribeRow{
		Kode:             cellValue(row, idx, "Kode"),
		OwnerCode:        cellValue(row, idx, "Kode Owner"),
		OwnerName:        cellValue(row, idx, "Nama Owner"),
		ProjectName:      project,
		OutletName:       outlet,
		City:             cellValue(row, idx, "Kota"),
		Province:         cellValue(row, idx, "Provinsi"),
		TanggalAktivasi:  normalizeOptionalExcelDate(cellValue(row, idx, "Tanggal Aktivasi")),
		MetodePembayaran: cellValue(row, idx, "Metode Pembayaran"),
		DateOfRenewal:    normalizeOptionalExcelDate(cellValue(row, idx, "Date of Renewal On Member")),
		ExpiredDate:      normalizeOptionalExcelDate(cellValue(row, idx, "Expired Date")),
		PaketMembership:  cellValue(row, idx, "Paket Membership"),
		Status:           cellValue(row, idx, "Status"),
	}

	if r.Kode == "" {
		errs = append(errs, "kode wajib diisi")
	}
	if r.OwnerCode == "" {
		errs = append(errs, "kode_owner wajib diisi")
	}
	if r.PaketMembership == "" {
		errs = append(errs, "paket_membership wajib diisi")
	}

	if raw := cellValue(row, idx, "No. Hp Owner/Outlet"); raw != "" {
		phone, err := customer.NormalizePhone(raw)
		if err != nil {
			errs = append(errs, fmt.Sprintf("no_hp_owner_outlet tidak valid: %s", raw))
		} else {
			r.OwnerOutletPhone = phone
		}
	}

	if raw := cellValue(row, idx, "Tenor"); raw != "" {
		tenor, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || tenor < 1 {
			errs = append(errs, fmt.Sprintf("tenor tidak valid: %s", raw))
		} else {
			r.TenorMonths = tenor
		}
	} else {
		errs = append(errs, "tenor wajib diisi")
	}

	if nominal, err := normalizeMoneyString(cellValue(row, idx, "Nominal Aktivasi")); err != nil {
		errs = append(errs, err.Error())
	} else {
		r.NominalAktivasi = nominal
	}
	if balance, err := normalizeMoneyString(cellValue(row, idx, "Balance Top Up System")); err != nil {
		errs = append(errs, err.Error())
	} else {
		r.BalanceTopUp = balance
	}
	// Diskon/Fee/Settlement are informational cross-checks on the top-up amount, not required —
	// an unparsable value is dropped (left "0"), not fatal to the row.
	if diskon, err := normalizeMoneyString(cellValue(row, idx, "Diskon Langganan")); err == nil {
		r.Diskon = diskon
	}
	if fee, err := normalizeMoneyString(cellValue(row, idx, "Fee Midtrans")); err == nil {
		r.FeeMidtrans = fee
	}
	if settlement, err := normalizeMoneyString(cellValue(row, idx, "Settlement")); err == nil {
		r.Settlement = settlement
	}

	return r, errs
}

/* ---------- Profile MONTHLY_ACTIVE ---------- */

type monthColumn struct {
	Col   int
	Year  int
	Month int
}

type monthlyActiveEntry struct {
	PeriodYear  int    `json:"period_year"`
	PeriodMonth int    `json:"period_month"`
	RawCode     string `json:"raw_code"`
	Category    string `json:"category"`
	PackageCode string `json:"package_code,omitempty"`
	TenorMonths *int   `json:"tenor_months,omitempty"`
}

type monthlyActiveRow struct {
	OwnerCode   string               `json:"owner_code"`
	OwnerName   string               `json:"owner_name,omitempty"`
	ProjectName string               `json:"project_name,omitempty"`
	OutletName  string               `json:"outlet_name"`
	City        string               `json:"city,omitempty"`
	Region      string               `json:"region,omitempty"`
	Activities  []monthlyActiveEntry `json:"activities"`
}

func monthNumberFromLabel(raw string) int {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "jan", "januari":
		return 1
	case "feb", "februari":
		return 2
	case "mar", "maret":
		return 3
	case "apr", "april":
		return 4
	case "mei", "may":
		return 5
	case "jun", "juni":
		return 6
	case "jul", "juli":
		return 7
	case "agu", "agustus", "agt":
		return 8
	case "sep", "sept", "september":
		return 9
	case "okt", "oct", "oktober":
		return 10
	case "nov", "november":
		return 11
	case "des", "desember", "dec":
		return 12
	default:
		return 0
	}
}

// inferMonthlyActiveColumns derives the sequential month/year mapping for the "Monthly active"
// sheet. The real workbook stores a 64-column rolling timeline (starting at Dec of the year before
// the first 4-digit year marker) with occasional corrupted header cells; once the first
// year+month anchor is found, the safest interpretation is simply "one column = next month".
func inferMonthlyActiveColumns(rows [][]string, headerRowIdx int, idx headerIndex) ([]monthColumn, error) {
	headerRow := rows[headerRowIdx]
	outletCol, ok := idx["NAMA OUTLET"]
	if !ok {
		return nil, fmt.Errorf("header NAMA OUTLET tidak ditemukan")
	}

	firstMonthCol := -1
	for col := outletCol + 1; col < len(headerRow); col++ {
		if monthNumberFromLabel(headerRow[col]) > 0 {
			firstMonthCol = col
			break
		}
	}
	if firstMonthCol == -1 {
		return nil, fmt.Errorf("kolom bulan monthly active tidak ditemukan")
	}

	anchorCol, anchorYear, anchorMonth := -1, 0, 0
outer:
	for rowIdx := 0; rowIdx < headerRowIdx && rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		for col := firstMonthCol; col < len(headerRow); col++ {
			yearRaw := cellAt(row, col)
			if len(yearRaw) != 4 {
				continue
			}
			year, err := strconv.Atoi(yearRaw)
			if err != nil || year < 2000 || year > 2100 {
				continue
			}
			month := monthNumberFromLabel(headerRow[col])
			if month == 0 {
				continue
			}
			anchorCol, anchorYear, anchorMonth = col, year, month
			break outer
		}
	}
	if anchorCol == -1 {
		return nil, fmt.Errorf("tahun anchor monthly active tidak ditemukan")
	}

	start := time.Date(anchorYear, time.Month(anchorMonth), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(anchorCol - firstMonthCol), 0)
	cols := make([]monthColumn, 0, len(headerRow)-firstMonthCol)
	for col := firstMonthCol; col < len(headerRow); col++ {
		period := start.AddDate(0, col-firstMonthCol, 0)
		cols = append(cols, monthColumn{
			Col:   col,
			Year:  period.Year(),
			Month: int(period.Month()),
		})
	}
	return cols, nil
}

func parseMonthlyActiveCode(raw string) (category, packageCode string, tenorMonths *int, ok bool) {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if raw == "" {
		return "", "", nil, false
	}
	if _, err := strconv.ParseFloat(raw, 64); err == nil {
		return "", "", nil, false
	}

	switch raw {
	case "X":
		return "NOT_ACTIVATED", "", nil, true
	case "CLOSE":
		return "UNSUBSCRIBE", "", nil, true
	}

	parts := strings.SplitN(raw, "-", 3)
	if len(parts) < 2 {
		return "", "", nil, false
	}

	switch parts[0] {
	case "N":
		category = "NEW"
	case "S":
		category = "SUBSCRIBE"
	case "F":
		category = "FOLLOWING"
	case "U":
		category = "UNSUBSCRIBE"
	default:
		return "", "", nil, false
	}

	switch {
	case strings.Contains(raw, "-BC"):
		packageCode = "BC"
	case strings.Contains(raw, "-BS"):
		packageCode = "BS"
	case strings.Contains(raw, "-PR"):
		packageCode = "PR"
	default:
		return "", "", nil, false
	}

	if digits := firstDigitsPattern.FindString(raw); digits != "" {
		if parsed, err := strconv.Atoi(digits); err == nil && parsed > 0 {
			tenorMonths = &parsed
		}
	} else if strings.Contains(raw, "-F") {
		// Real workbook shorthand like "N-BS-F" omits the "(12)" suffix for annual plans.
		parsed := 12
		tenorMonths = &parsed
	}

	return category, packageCode, tenorMonths, true
}

func parseMonthlyActiveRow(row []string, idx headerIndex, months []monthColumn) (monthlyActiveRow, []string) {
	var errs []string
	projectName, outletFallback := splitProjectOutlet(cellValue(row, idx, "Nama Brand/Outlet"))
	outletName := cellValue(row, idx, "Nama Outlet")
	if outletName == "" || strings.EqualFold(outletName, "#VALUE!") {
		outletName = outletFallback
	}

	r := monthlyActiveRow{
		OwnerCode:   cellValue(row, idx, "Kode Owner"),
		OwnerName:   cellValue(row, idx, "Nama Owner"),
		ProjectName: projectName,
		OutletName:  outletName,
		City:        cellValue(row, idx, "Kota"),
		Region:      cellValue(row, idx, "Wilayah"),
		Activities:  make([]monthlyActiveEntry, 0, len(months)),
	}

	if r.OwnerCode == "" {
		errs = append(errs, "kode_owner wajib diisi")
	}
	if r.OutletName == "" {
		errs = append(errs, "nama_outlet wajib diisi")
	}

	for _, month := range months {
		rawCode := cellAt(row, month.Col)
		if rawCode == "" {
			continue
		}
		if _, err := strconv.ParseFloat(rawCode, 64); err == nil {
			continue
		}
		if rawCode == "-" {
			continue
		}
		category, packageCode, tenorMonths, ok := parseMonthlyActiveCode(rawCode)
		if !ok {
			errs = append(errs, fmt.Sprintf("kode monthly active tidak dikenali untuk %04d-%02d: %s", month.Year, month.Month, rawCode))
			continue
		}
		r.Activities = append(r.Activities, monthlyActiveEntry{
			PeriodYear:  month.Year,
			PeriodMonth: month.Month,
			RawCode:     strings.ToUpper(rawCode),
			Category:    category,
			PackageCode: packageCode,
			TenorMonths: tenorMonths,
		})
	}

	if len(r.Activities) == 0 {
		errs = append(errs, "tidak ada kode monthly active yang dapat diproses")
	}

	return r, errs
}

/* ---------- Profile BONUS_MITRA ---------- */

type bonusMitraRow struct {
	PartnerName        string `json:"partner_name"`
	PartnerOwnerCode   string `json:"partner_owner_code,omitempty"`
	PartnerOwnerName   string `json:"partner_owner_name,omitempty"`
	PartnerBrandName   string `json:"partner_brand_name,omitempty"`
	PartnerCategory    string `json:"partner_category,omitempty"`
	ReferredOwnerCode  string `json:"referred_owner_code"`
	ReferredOwnerName  string `json:"referred_owner_name,omitempty"`
	ReferredProject    string `json:"referred_project_name,omitempty"`
	ReferredOutletName string `json:"referred_outlet_name,omitempty"`
	PackageName        string `json:"package_name,omitempty"`
	SalesPICName       string `json:"sales_pic_name,omitempty"`
	TopUpDate          string `json:"top_up_date,omitempty"`
	RenewalDate        string `json:"renewal_date,omitempty"`
	PayoutDate1        string `json:"payout_date_1,omitempty"`
	PayoutDate2        string `json:"payout_date_2,omitempty"`
	SubscriptionStatus string `json:"subscription_status,omitempty"`
	PayoutStatus       string `json:"payout_status,omitempty"`
	CMOName            string `json:"cmo_name,omitempty"`
	UnpaidAmount       string `json:"unpaid_amount"`
	Stage1Amount       string `json:"stage1_amount"`
	Stage2Amount       string `json:"stage2_amount"`
	PaidAmount         string `json:"paid_amount"`
	TotalAmount        string `json:"total_amount"`
}

func parseBonusMitraRow(row []string, _ headerIndex) (bonusMitraRow, []string) {
	var errs []string
	referredProject, referredOutlet := splitProjectOutlet(cellAt(row, 7))
	r := bonusMitraRow{
		PartnerName:        cellAt(row, 1),
		PartnerOwnerCode:   cellAt(row, 2),
		PartnerOwnerName:   cellAt(row, 3),
		PartnerBrandName:   cellAt(row, 4),
		PartnerCategory:    cellAt(row, 9),
		ReferredOwnerCode:  cellAt(row, 5),
		ReferredOwnerName:  cellAt(row, 6),
		ReferredProject:    referredProject,
		ReferredOutletName: referredOutlet,
		PackageName:        cellAt(row, 10),
		SalesPICName:       cellAt(row, 11),
		TopUpDate:          normalizeOptionalExcelDate(cellAt(row, 13)),
		RenewalDate:        normalizeOptionalExcelDate(cellAt(row, 14)),
		PayoutDate1:        normalizeOptionalExcelDate(cellAt(row, 16)),
		PayoutDate2:        normalizeOptionalExcelDate(cellAt(row, 17)),
		SubscriptionStatus: cellAt(row, 15),
		PayoutStatus:       cellAt(row, 18),
		CMOName:            cellAt(row, 19),
	}

	if r.PartnerName == "" {
		errs = append(errs, "partner_name wajib diisi")
	}
	if r.ReferredOwnerCode == "" {
		errs = append(errs, "referred_owner_code wajib diisi")
	}

	var err error
	if r.UnpaidAmount, err = normalizeMoneyString(cellAt(row, 65)); err != nil {
		errs = append(errs, "unpaid_amount: "+err.Error())
	}
	if r.Stage1Amount, err = normalizeMoneyString(cellAt(row, 66)); err != nil {
		errs = append(errs, "stage1_amount: "+err.Error())
	}
	if r.Stage2Amount, err = normalizeMoneyString(cellAt(row, 67)); err != nil {
		errs = append(errs, "stage2_amount: "+err.Error())
	}
	if r.PaidAmount, err = normalizeMoneyString(cellAt(row, 68)); err != nil {
		errs = append(errs, "paid_amount: "+err.Error())
	}
	if r.TotalAmount, err = normalizeMoneyString(cellAt(row, 69)); err != nil {
		errs = append(errs, "total_amount: "+err.Error())
	}

	return r, errs
}

/* ---------- Profile SALES_TARGET ---------- */

// indonesianMonths maps the "Bulan" column's Indonesian month names (as used in both the
// "TARGET" sheet and "02. New & Subscribe") to their 1-12 number. Not exported — this profile is
// the only one that needs a month *name* rather than a date/serial.
var indonesianMonths = map[string]int{
	"januari": 1, "februari": 2, "maret": 3, "april": 4, "mei": 5, "juni": 6,
	"juli": 7, "agustus": 8, "september": 9, "oktober": 10, "november": 11, "desember": 12,
}

// yearInTitleRows scans the merged title rows above the header (e.g. "Tahun 2026") for a 4-digit
// year. The "TARGET" sheet never repeats the year per data row — only "Bulan" (month name) — so
// the year has to come from the sheet's title, not the row itself.
func yearInTitleRows(rows [][]string, headerRowIdx int) (int, bool) {
	for i := 0; i < headerRowIdx && i < len(rows); i++ {
		for _, cell := range rows[i] {
			for _, word := range strings.Fields(cell) {
				if len(word) == 4 {
					if year, err := strconv.Atoi(word); err == nil && year >= 2000 && year <= 2100 {
						return year, true
					}
				}
			}
		}
	}
	return 0, false
}

// salesTargetRow is one month's target-setting from the "TARGET" sheet. Only Target
// User/Target Omset are imported — "Realisasi..." columns are Sprint 13's KPI worker's own
// live-computed output, not something to import as a conflicting second source of truth.
type salesTargetRow struct {
	PeriodYear  int    `json:"period_year"`
	PeriodMonth int    `json:"period_month"`
	TargetUser  string `json:"target_user"`
	TargetOmset string `json:"target_omset"`
}

func parseSalesTargetRow(row []string, idx headerIndex, year int) (salesTargetRow, []string) {
	var errs []string
	r := salesTargetRow{PeriodYear: year}

	if year == 0 {
		errs = append(errs, "tahun tidak ditemukan di judul sheet")
	}

	bulan := strings.ToLower(strings.TrimSpace(cellValue(row, idx, "Bulan")))
	if month, ok := indonesianMonths[bulan]; ok {
		r.PeriodMonth = month
	} else {
		errs = append(errs, fmt.Sprintf("bulan tidak dikenali: %q", bulan))
	}

	if targetUser, err := normalizeMoneyString(cellValue(row, idx, "Target User")); err != nil {
		errs = append(errs, err.Error())
	} else {
		r.TargetUser = targetUser
	}
	if targetOmset, err := normalizeMoneyString(cellValue(row, idx, "Target Omset")); err != nil {
		errs = append(errs, err.Error())
	} else {
		r.TargetOmset = targetOmset
	}

	return r, errs
}

/* ---------- Profile SALES_CALL_CHAT ---------- */

// notYetSentinel is the "Finalisasi (Closing)" column's value for a lead that hasn't subscribed —
// a real package name (Basic/Business/Pro) in that column, paired with Scor 3, is what actually
// signals a closing (confirmed against real rows in "Call & Chat-Lidya": every Scor-3 row carries
// a real package name there, every lower-score row carries this sentinel or is blank).
const notYetSentinel = "not yet"

// salesCallChatRow is one lead interaction (and, when Scor is 3 with a real package name in
// Finalisasi, an accompanying Closing) from the "Call & Chat-Lidya" sheet.
type salesCallChatRow struct {
	OwnerCode       string `json:"owner_code"`
	OwnerName       string `json:"owner_name,omitempty"`
	OwnerPhone      string `json:"owner_phone,omitempty"`
	TanggalFU       string `json:"tanggal_fu,omitempty"`
	Scor            int    `json:"scor"`
	CallStatus      string `json:"call_status,omitempty"`
	ChatStatus      string `json:"chat_status,omitempty"`
	Validitas       string `json:"validitas,omitempty"`
	Remaks          string `json:"remaks,omitempty"`
	FinalisasiPaket string `json:"finalisasi_paket,omitempty"`
	Nominal         string `json:"nominal,omitempty"`
	IsClosing       bool   `json:"is_closing"`
}

func parseSalesCallChatRow(row []string, idx headerIndex) (salesCallChatRow, []string) {
	var errs []string
	r := salesCallChatRow{
		OwnerCode:  cellValue(row, idx, "Kode Owner"),
		OwnerName:  cellValue(row, idx, "Nama Owner"),
		TanggalFU:  normalizeOptionalExcelDate(cellValue(row, idx, "Tanggal FU")),
		CallStatus: cellValue(row, idx, "Call"),
		ChatStatus: cellValue(row, idx, "Chat"),
		Validitas:  cellValue(row, idx, "Validitas"),
		Remaks:     cellValue(row, idx, "Remaks"),
	}

	if r.OwnerCode == "" {
		errs = append(errs, "kode_owner wajib diisi")
	}

	if raw := cellValue(row, idx, "No. HP Owner"); raw != "" {
		phone, err := customer.NormalizePhone(raw)
		if err != nil {
			errs = append(errs, fmt.Sprintf("no_hp_owner tidak valid: %s", raw))
		} else {
			r.OwnerPhone = phone
		}
	}

	scorRaw := strings.TrimSpace(cellValue(row, idx, "Scor"))
	if scorRaw == "" {
		errs = append(errs, "scor wajib diisi")
	} else if scor, err := strconv.Atoi(scorRaw); err != nil || scor < 0 || scor > 3 {
		errs = append(errs, fmt.Sprintf("scor tidak valid: %s", scorRaw))
	} else {
		r.Scor = scor
	}

	finalisasi := strings.TrimSpace(cellValue(row, idx, "Finalisasi (Closing)"))
	r.FinalisasiPaket = finalisasi
	r.IsClosing = finalisasi != "" && strings.ToLower(finalisasi) != notYetSentinel

	if r.IsClosing {
		if nominal, err := normalizeMoneyString(cellValue(row, idx, "Nominal")); err != nil {
			errs = append(errs, err.Error())
		} else {
			r.Nominal = nominal
		}
	}

	return r, errs
}

/* ---------- File reading ---------- */

// readRows opens an xlsx file and returns all rows of its first sheet. Kept for the Sprint 14
// profiles (OWNER_OUTLET/NON_REGISTER), whose workbooks are always single-sheet.
func readRows(path string) ([][]string, error) {
	sheets, err := readAllSheets(path)
	if err != nil {
		return nil, err
	}
	return sheets[0].Rows, nil
}

// namedSheet pairs a workbook sheet's name with its rows — Sprint 15's profiles live on
// non-first sheets of multi-sheet workbooks (e.g. "Mitra - Referral" is the 2nd sheet of
// "06. Data Bonus Mitra"; "Call & Chat-Lidya"/"TARGET" are sheets within the same PBGC sales
// file, imported as two separate batches per the confirmed "one batch = one profile" model).
type namedSheet struct {
	Name string
	Rows [][]string
}

// readAllSheets opens an xlsx file and returns every sheet's rows, in workbook order.
func readAllSheets(path string) ([]namedSheet, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer f.Close()

	sheetNames := f.GetSheetList()
	if len(sheetNames) == 0 {
		return nil, fmt.Errorf("workbook has no sheets")
	}
	sheets := make([]namedSheet, 0, len(sheetNames))
	for _, name := range sheetNames {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("read sheet %s: %w", name, err)
		}
		sheets = append(sheets, namedSheet{Name: name, Rows: rows})
	}
	return sheets, nil
}

// detectProfileAcrossSheets tries detectProfile on each sheet in turn, returning the first sheet
// that yields exactly one profile match. Sheets that are pure reference/summary data (e.g.
// "Daftar Mitra", "Komisi Mitra Corporate") are expected to match nothing and are skipped, not
// treated as an error.
func detectProfileAcrossSheets(sheets []namedSheet) (profile string, sheetIdx int, headerRow int, headers headerIndex, err error) {
	for i, sheet := range sheets {
		p, hRow, hIdx, derr := detectProfile(sheet.Rows)
		if derr == nil {
			return p, i, hRow, hIdx, nil
		}
	}
	return "", 0, 0, nil, ErrProfileRequired
}

// verifyProfileAcrossSheets tries verifyProfile on each sheet in turn for a user-declared profile,
// returning the first sheet whose headers match that profile's markers.
func verifyProfileAcrossSheets(sheets []namedSheet, profile string) (sheetIdx int, headerRow int, headers headerIndex, err error) {
	for i, sheet := range sheets {
		hRow, hIdx, verr := verifyProfile(sheet.Rows, profile)
		if verr == nil {
			return i, hRow, hIdx, nil
		}
	}
	return 0, 0, nil, ErrProfileHeaderMismatch
}
