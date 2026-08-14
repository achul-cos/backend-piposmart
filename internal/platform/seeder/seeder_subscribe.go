package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"backend_crm_piposmart/internal/platform/factory"
)

type subscribeRow struct {
	Kode            string // "Kode" column — stable per-transaction id, used for the dedup/idempotency key
	Year            int    // from filename, used as a date-fallback anchor
	SheetMonth      int    // 1-12 from the sheet name when it's a plain month sheet, 0 if unknown
	OwnerCode       string // blank for 2021/2022 archives — those files have no "Kode Owner" column at all
	OwnerName       string
	OwnerPhone      string
	ProjectOutlet   string // raw "Brand/Outlet Name" cell, split downstream by splitProjectOutlet
	City            string
	Province        string
	DateWork        string
	DateTopUp       string
	DateStruk       string
	DateAktivasi    string
	DateRenewal     string
	DateExpired     string
	BalanceTf       string
	BalanceSystem   string
	NominalAktivasi string
	FeeMidtrans     string
	Settlement      string
	MetodeBayar     string
	Paket           string
	Tenor           string
}

// archiveFiles are the yearly "New & Subscribe" archives (2021-2026). The single-year sample
// "02. New & Subscribe 2026 (Copy).xlsx" is deliberately NOT read here — it's a subset of the same
// 2026 data already covered by the 2026 archive below; reading both would double-count rows.
// 2021/2022 predate the "Kode Owner" column entirely (see readNewAndSubscribeExcel's header
// handling) — owners/outlets for those rows are related by phone match or created on the fly.
var archiveFiles = []struct {
	name string
	year int
}{
	{"Salinan New & Subscribe 2021.xlsx", 2021},
	{"salinan subscribe 2022.xlsx", 2022},
	{"Salinan New & Subscribe 2023.xlsx", 2023},
	{"Salinan 1. New & Subscribe 2024.xlsx", 2024},
	{"Salinan 1. New & Subscribe 2025.xlsx", 2025},
	{"Salinan 1. New & Subscribe 2026.xlsx", 2026},
}

var reMonthSheetName = regexp.MustCompile(`^\d{2}$`)

// sheetMonthFromName returns 1-12 when the sheet name is a plain zero-padded month ("01".."12"),
// else 0. Every archive file also carries legacy/duplicate sheets ("06 old", "Copy of 06", "MEI",
// "01 (2026)", "Saldo", "Daily Report", "Selisih Midtrans", ...) that share the same header
// signature as real month sheets and would otherwise get parsed as duplicate data.
func sheetMonthFromName(name string) int {
	name = strings.TrimSpace(name)
	if !reMonthSheetName.MatchString(name) {
		return 0
	}
	n, err := strconv.Atoi(name)
	if err != nil || n < 1 || n > 12 {
		return 0
	}
	return n
}

// sheetAllowed decides whether a sheet should be parsed at all: either it's a plain month sheet,
// or the workbook only has one sheet total (2023's archive is a single unsplit "New & Subscribe"
// sheet, so there's no ambiguity to guard against).
func sheetAllowed(totalSheets int, name string) bool {
	if totalSheets == 1 {
		return true
	}
	return sheetMonthFromName(name) != 0
}

func readNewAndSubscribeExcel() ([]subscribeRow, error) {
	var results []subscribeRow

	searchDirs := []string{
		filepath.Join("asset", "data_admin", "arsip_new_subscribe"),
		filepath.Join("..", "asset", "data_admin", "arsip_new_subscribe"),
		filepath.Join("backend", "asset", "data_admin", "arsip_new_subscribe"),
	}

	for _, archive := range archiveFiles {
		var f *excelize.File
		for _, dir := range searchDirs {
			opened, err := excelize.OpenFile(filepath.Join(dir, archive.name))
			if err == nil {
				f = opened
				break
			}
		}
		if f == nil {
			continue // this year's archive not found in any search location — skip, not fatal
		}

		sheetList := f.GetSheetList()
		for _, sheet := range sheetList {
			if !sheetAllowed(len(sheetList), sheet) {
				continue
			}
			sheetMonth := sheetMonthFromName(sheet)

			rows, err := f.GetRows(sheet)
			if err != nil || len(rows) <= 3 {
				continue
			}

			// Find header
			headerRowIdx := -1
			colKode, colDateWork, colOwnerCode := -1, -1, -1
			colOwnerName, colOwnerPhone, colProjectOutlet, colCity, colProvince := -1, -1, -1, -1, -1
			colTopUpSystem, colTopUpStruk, colAktivasi := -1, -1, -1
			colNominalAktivasi, colBalanceSystem, colBalanceTf := -1, -1, -1
			colFeeMidtrans, colSettlement, colMetode := -1, -1, -1
			colRenewal, colExpired, colTenor, colPaket := -1, -1, -1, -1

			for rIdx := 0; rIdx < len(rows) && rIdx < 10; rIdx++ {
				row := rows[rIdx]
				for cIdx, cell := range row {
					u := strings.ToUpper(strings.TrimSpace(cell))
					switch u {
					case "KODE":
						colKode = cIdx
					case "DATE OF WORK":
						colDateWork = cIdx
					case "KODE OWNER":
						colOwnerCode = cIdx
					case "NAMA OWNER":
						colOwnerName = cIdx
					case "PROJECT/OUTLET":
						colProjectOutlet = cIdx
					case "KOTA":
						colCity = cIdx
					case "PROVINSI":
						colProvince = cIdx
					case "DATE TOP UP  SYSTEM", "DATE TOP UP":
						colTopUpSystem = cIdx
					case "DATE TOP UP STRUK":
						colTopUpStruk = cIdx
					case "NOMINAL AKTIVASI":
						colNominalAktivasi = cIdx
					case "TANGGAL AKTIVASI":
						colAktivasi = cIdx
					case "BALANCE TOP UP SYSTEM":
						colBalanceSystem = cIdx
					case "BALANCE BUKTI TF":
						colBalanceTf = cIdx
					case "FEE MIDTRANS":
						colFeeMidtrans = cIdx
					case "METODE PEMBAYARAN":
						colMetode = cIdx
					case "DATE OF RENEWAL ON MEMBER":
						colRenewal = cIdx
					case "EXPIRED DATE":
						colExpired = cIdx
					case "TENOR":
						colTenor = cIdx
					case "PAKET MEMBERSHIP":
						colPaket = cIdx
					default:
						// "Settlement " / "Settlement Midtrans" — variant header text across years.
						if colSettlement == -1 && strings.Contains(u, "SETTLEMENT") {
							colSettlement = cIdx
						}
						// "No. Hp" (2021) / "No. Hp Owner/Outlet" (2022-2026) — variant naming.
						if colOwnerPhone == -1 && strings.Contains(u, "HP") {
							colOwnerPhone = cIdx
						}
					}
				}
				// The 2022 archive's "Nama Owner" header cell is blank even though the column
				// (immediately after Kode) holds real owner-name data in every row — only apply
				// this positional fallback for files that also lack "Kode Owner" entirely, so it
				// can never misfire on the 2023-2026 files (which always resolve colOwnerCode).
				if colOwnerName == -1 && colOwnerCode == -1 && colKode != -1 && colKode+1 < len(row) {
					colOwnerName = colKode + 1
				}
				if colKode != -1 && (colOwnerCode != -1 || colOwnerName != -1) {
					headerRowIdx = rIdx
					break
				}
			}

			if headerRowIdx == -1 {
				continue
			}

			for rIdx := headerRowIdx + 1; rIdx < len(rows); rIdx++ {
				row := rows[rIdx]
				if len(row) == 0 {
					continue
				}

				code := getCol(row, colOwnerCode)
				ownerName := getCol(row, colOwnerName)
				if code == "" && ownerName == "" {
					// Neither identifier present — a blank spacer row, a stub row with only a
					// leftover "Kode" value (seen in the 2022 archive's tail), or a trailing
					// total/summary row (seen at the end of both the 2021 and 2022 archives).
					continue
				}

				results = append(results, subscribeRow{
					Kode:            getCol(row, colKode),
					Year:            archive.year,
					SheetMonth:      sheetMonth,
					OwnerCode:       code,
					OwnerName:       ownerName,
					OwnerPhone:      getCol(row, colOwnerPhone),
					ProjectOutlet:   getCol(row, colProjectOutlet),
					City:            getCol(row, colCity),
					Province:        getCol(row, colProvince),
					DateWork:        getCol(row, colDateWork),
					DateTopUp:       getCol(row, colTopUpSystem),
					DateStruk:       getCol(row, colTopUpStruk),
					DateAktivasi:    getCol(row, colAktivasi),
					DateRenewal:     getCol(row, colRenewal),
					DateExpired:     getCol(row, colExpired),
					BalanceTf:       getCol(row, colBalanceTf),
					BalanceSystem:   getCol(row, colBalanceSystem),
					NominalAktivasi: getCol(row, colNominalAktivasi),
					FeeMidtrans:     getCol(row, colFeeMidtrans),
					Settlement:      getCol(row, colSettlement),
					MetodeBayar:     getCol(row, colMetode),
					Paket:           getCol(row, colPaket),
					Tenor:           getCol(row, colTenor),
				})
			}
		}
		f.Close()
	}
	return results, nil
}

var reDigitsOnly = regexp.MustCompile(`^\d+$`)

// cleanAmountString strips thousands separators from a Rupiah figure copied out of Excel. Source
// data has no decimals, so both "," and "." are treated purely as separators, never as a decimal
// point (matches the convention the original single-file importer already used). Admins also use
// placeholder text like "-" for a genuinely blank cell — anything that isn't plain digits after
// stripping separators is treated as blank rather than passed through to a money parser.
func cleanAmountString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, " ", "")
	if !reDigitsOnly.MatchString(s) {
		return ""
	}
	return s
}

// resolveFallbackDate tries each candidate cell in order; when none parse, it anchors on the row's
// known file-year + sheet-month (defaulting to the 1st of that month) instead of wall-clock "now" —
// a 2023 row must never be seeded as if it happened in 2026 just because its date cells were blank.
func resolveFallbackDate(row subscribeRow, candidates ...string) time.Time {
	for _, c := range candidates {
		if t := parseDateRobust(c); !t.IsZero() {
			return t
		}
	}
	month := row.SheetMonth
	if month == 0 {
		month = 1
	}
	return time.Date(row.Year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
}

// hasPurchaseIntent is the reliable signal for "a real package purchase happened on this row".
// Tenor/Paket Membership/Status are not reliable — admins left them at lazy defaults (1/Basic/
// Berlangganan) even on rows that were topup-only.
func hasPurchaseIntent(row subscribeRow) bool {
	return !parseDateRobust(row.DateAktivasi).IsZero() ||
		!parseDateRobust(row.DateRenewal).IsZero() ||
		!parseDateRobust(row.DateExpired).IsZero()
}

// extractUniqueCode returns the trailing amount-mod-1000 digits (e.g. 78012 -> "12"), the TF/BRI
// unique transfer-matching suffix, or "" when the amount is already a round number.
func extractUniqueCode(cleanedAmount string) string {
	n, err := strconv.ParseInt(cleanedAmount, 10, 64)
	if err != nil || n <= 0 {
		return ""
	}
	remainder := n % 1000
	if remainder == 0 {
		return ""
	}
	return strconv.FormatInt(remainder, 10)
}

// inferPaymentMethod resolves TF_BRI / MIDTRANS / SALDO_APLIKASI. It trusts the literal "Metode
// Pembayaran" column first; when that's blank/unrecognized it falls back to the business rule the
// admins themselves used: a round Balance Top Up System figure (no unique-code suffix) means
// Midtrans, a non-round one means TF/BRI, and both balances blank means no topup evidence at all
// (Saldo Aplikasi — an old leftover balance being spent).
func inferPaymentMethod(metode, balanceSystemRaw, balanceTfRaw string) (method string, uniqueCode string) {
	m := strings.ToUpper(strings.TrimSpace(metode))
	switch {
	case strings.Contains(m, "MIDTRANS") || strings.Contains(m, "MDR"):
		return "MIDTRANS", ""
	case strings.Contains(m, "SALDO"):
		return "SALDO_APLIKASI", ""
	case strings.Contains(m, "TF") || strings.Contains(m, "BRI") || strings.Contains(m, "TRANSFER"):
		return "TF_BRI", extractUniqueCode(cleanAmountString(balanceSystemRaw))
	}

	balanceSystem := cleanAmountString(balanceSystemRaw)
	balanceTf := cleanAmountString(balanceTfRaw)
	sysEmpty := balanceSystem == "" || balanceSystem == "0"
	tfEmpty := balanceTf == "" || balanceTf == "0"
	if sysEmpty && tfEmpty {
		return "SALDO_APLIKASI", ""
	}
	if !sysEmpty {
		if n, err := strconv.ParseInt(balanceSystem, 10, 64); err == nil && n%1000 == 0 {
			return "MIDTRANS", ""
		}
		return "TF_BRI", extractUniqueCode(balanceSystem)
	}
	return "TF_BRI", ""
}

// splitProjectOutlet parses the combined "Project/Outlet" cell ("Brand/Outlet Name") into its two
// parts. A cell with no "/" is treated as an outlet name with no separate brand.
func splitProjectOutlet(raw string) (brand string, outletName string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	idx := strings.Index(raw, "/")
	if idx == -1 {
		return "", raw
	}
	brand = strings.TrimSpace(raw[:idx])
	outletName = strings.TrimSpace(raw[idx+1:])
	if outletName == "" {
		outletName = brand
	}
	return brand, outletName
}

// normalizePhoneForMatch strips everything but digits and drops a leading country/trunk prefix
// (62 or 0), so "62896xxxx", "0896xxxx", and "896xxxx" all compare on the same core number —
// needed because owners.phone in this dataset isn't stored in one consistent format.
func normalizePhoneForMatch(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	switch {
	case strings.HasPrefix(digits, "62"):
		digits = digits[2:]
	case strings.HasPrefix(digits, "0"):
		digits = digits[1:]
	}
	return digits
}

// minMatchablePhoneDigits guards against matching on a too-short/placeholder phone fragment.
const minMatchablePhoneDigits = 7

// deriveFallbackOwnerCode mirrors the fallback convention seeder_real.go already uses when a row's
// own owner code is missing/unusable: prefer a phone-derived code, then a name-derived one, then a
// purely positional one — so a brand-new owner created here still gets a stable, traceable code.
func deriveFallbackOwnerCode(name, phone string, rowIndex int) string {
	phone = strings.TrimSpace(phone)
	if phone != "" {
		return "OWN-" + phone
	}
	name = strings.TrimSpace(name)
	if name != "" {
		cleanName := strings.ToUpper(strings.ReplaceAll(name, " ", ""))
		if len(cleanName) > 10 {
			cleanName = cleanName[:10]
		}
		return "OWN-" + cleanName
	}
	return fmt.Sprintf("OWN-NS-%05d", rowIndex)
}

// loadOwnerPhoneIndex preloads a normalized-phone -> owner id map from the current owners table,
// so per-row phone matching (the fallback relation path when a row has no usable owner code, or its
// code doesn't resolve) is an O(1) map lookup instead of a per-row fuzzy SQL query. When two owners
// share a normalized phone (placeholder/duplicate data), the first one encountered wins — a
// best-effort relation heuristic, not a strict guarantee.
func loadOwnerPhoneIndex(ctx context.Context, tx *sql.Tx) (map[string]int64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, phone FROM owners
		WHERE deleted_at IS NULL AND phone IS NOT NULL AND phone <> '' AND phone <> '-'`)
	if err != nil {
		return nil, fmt.Errorf("load owner phone index: %w", err)
	}
	defer rows.Close()

	index := make(map[string]int64)
	for rows.Next() {
		var id int64
		var phone string
		if err := rows.Scan(&id, &phone); err != nil {
			return nil, err
		}
		key := normalizePhoneForMatch(phone)
		if len(key) < minMatchablePhoneDigits {
			continue
		}
		if _, exists := index[key]; !exists {
			index[key] = id
		}
	}
	return index, rows.Err()
}

// resolveOwnerAndOutlet relates a row to an owner/outlet, prioritizing existing data: first by the
// row's own owner code (when present — always true for 2023-2026, never true for 2021-2022, which
// have no "Kode Owner" column at all), then by a normalized phone match against every owner already
// in the database. Only when neither matches does it create a brand-new owner+outlet (and, for a
// brand-new owner, a minimal customer_leads row under the "Tanpa PIC" fallback sales identity, so
// the closing/order chain below has a lead to attach to instead of landing as a HANGING_ORDER).
func resolveOwnerAndOutlet(
	ctx context.Context,
	tx *sql.Tx,
	fake *factory.Factory,
	row subscribeRow,
	ownerPhoneIndex map[string]int64,
	fallbackSalesEmail string,
	createdAt time.Time,
	rowIndex int,
) (ownerID int64, err error) {
	code := strings.TrimSpace(row.OwnerCode)
	phoneKey := normalizePhoneForMatch(row.OwnerPhone)

	var ownerFound bool
	if code != "" {
		lookupErr := tx.QueryRowContext(ctx, "SELECT id FROM owners WHERE code = ? AND deleted_at IS NULL LIMIT 1", code).Scan(&ownerID)
		if lookupErr == nil {
			ownerFound = true
		} else if lookupErr != sql.ErrNoRows {
			return 0, lookupErr
		}
	}
	if !ownerFound && len(phoneKey) >= minMatchablePhoneDigits {
		if id, ok := ownerPhoneIndex[phoneKey]; ok {
			ownerID = id
			ownerFound = true
		}
	}

	isNewOwner := false
	if !ownerFound {
		isNewOwner = true
		ownerName := strings.TrimSpace(row.OwnerName)
		if ownerName == "" {
			ownerName = "-"
		}
		ownerPhone := strings.TrimSpace(row.OwnerPhone)
		if ownerPhone == "" {
			ownerPhone = "-"
		}
		ownerCode := code
		if ownerCode == "" {
			ownerCode = deriveFallbackOwnerCode(row.OwnerName, row.OwnerPhone, rowIndex)
		}
		brand, _ := splitProjectOutlet(row.ProjectOutlet)
		brandName := brand
		if brandName == "" {
			brandName = ownerName
		}
		newOwnerID, createErr := fake.CreateOwner(ctx, tx, factory.Owner{
			Code:      ownerCode,
			Name:      ownerName,
			Phone:     ownerPhone,
			BrandName: brandName,
			Province:  row.Province,
			City:      row.City,
			CreatedAt: createdAt,
		})
		if createErr != nil {
			return 0, fmt.Errorf("create owner from new & subscribe archive: %w", createErr)
		}
		ownerID = newOwnerID
		if len(phoneKey) >= minMatchablePhoneDigits {
			ownerPhoneIndex[phoneKey] = ownerID
		}
	}

	_, outletName := splitProjectOutlet(row.ProjectOutlet)
	if outletName == "" {
		outletName = "-"
	}

	var outletID int64
	var outletFound bool
	lookupErr := tx.QueryRowContext(ctx,
		"SELECT id FROM outlets WHERE owner_id = ? AND LOWER(name) = LOWER(?) AND deleted_at IS NULL LIMIT 1",
		ownerID, outletName).Scan(&outletID)
	if lookupErr == nil {
		outletFound = true
	} else if lookupErr != sql.ErrNoRows {
		return 0, lookupErr
	}
	if !outletFound {
		var outletCount int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM outlets WHERE owner_id = ?", ownerID).Scan(&outletCount); err != nil {
			return 0, err
		}
		outletCode := fmt.Sprintf("OUT-NS-%d-%02d", ownerID, outletCount+1)
		newOutletID, createErr := fake.CreateOutlet(ctx, tx, ownerID, factory.Outlet{
			Code:     outletCode,
			Name:     outletName,
			Phone:    strings.TrimSpace(row.OwnerPhone),
			Province: row.Province,
			City:     row.City,
		})
		if createErr != nil {
			return 0, fmt.Errorf("create outlet from new & subscribe archive owner=%d: %w", ownerID, createErr)
		}
		outletID = newOutletID
		if _, err := tx.ExecContext(ctx, "UPDATE outlets SET created_at = ? WHERE id = ?", createdAt, outletID); err != nil {
			return 0, err
		}
	}

	if isNewOwner {
		lead := factory.Lead{
			Code:             fmt.Sprintf("LEAD-NS-%d", ownerID),
			SourceType:       "IMPORT",
			SourceReference:  "excel:new-subscribe-archive",
			Stage:            "NEW",
			Status:           "OPEN",
			NextFollowUpAt:   createdAt.AddDate(0, 0, 7),
			ActiveSalesEmail: fallbackSalesEmail,
		}
		if _, err := fake.CreateLead(ctx, tx, ownerID, outletID, lead); err != nil {
			return 0, fmt.Errorf("create lead from new & subscribe archive owner=%d: %w", ownerID, err)
		}
	}

	return ownerID, nil
}

func SeedSubscriptionsFromExcel(ctx context.Context, tx *sql.Tx, fake *factory.Factory, adminID int64, fallbackSalesEmail string) error {
	_ = adminID // kept for call-site compatibility; CreateWalletTopup/CreateSubscriptionOrder resolve their own admin actor internally
	rows, err := readNewAndSubscribeExcel()
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	ownerPhoneIndex, err := loadOwnerPhoneIndex(ctx, tx)
	if err != nil {
		return err
	}

	for i, row := range rows {
		// 1. Parse dates first — a brand-new owner/outlet (see step 2) needs a historically
		// plausible created_at, not "now".
		paidAt := parseDateRobust(row.DateTopUp)
		transferDate := parseDateRobust(row.DateStruk)
		aktivasiDate := parseDateRobust(row.DateAktivasi)
		if aktivasiDate.IsZero() {
			aktivasiDate = resolveFallbackDate(row, row.DateTopUp, row.DateWork)
		}
		if paidAt.IsZero() {
			paidAt = aktivasiDate
		}
		ownerCreatedAt := aktivasiDate
		if !paidAt.IsZero() && paidAt.Before(ownerCreatedAt) {
			ownerCreatedAt = paidAt
		}

		// 2. Relate to an existing owner/outlet when possible (by code, then by phone);
		// only create a new owner/outlet when neither matches (see resolveOwnerAndOutlet doc).
		ownerID, err := resolveOwnerAndOutlet(ctx, tx, fake, row, ownerPhoneIndex, fallbackSalesEmail, ownerCreatedAt, i)
		if err != nil {
			return fmt.Errorf("resolve owner row=%d: %w", i, err)
		}

		// Per-transaction key: year + the row's position in the combined multi-file/multi-year
		// slice. `i` alone already guarantees global uniqueness across the whole run — the row's
		// own "Kode" cell reads as a per-row transaction id in most rows but isn't reliably
		// unique (some rows carry copy-pasted junk there, e.g. an outlet name) so it's not used
		// for the key. Deliberately kept short: CreateWalletTopup/CreateWalletDebit/
		// CreateSubscriptionOrder derive their *_code columns via factory.CompactSeedKey, which
		// keeps only the LAST 40 characters of the idempotency key — a long key here would let a
		// topup and its paired subscription order (same owner, same row, differing only in the
		// "excel-tf-"/"excel-sub-" prefix) collide once that differing prefix falls outside the
		// last 40 characters (this happened with a real row: keys long enough to truncate away
		// the "TF"/"SUB" distinction produced identical DEMO-WTX-... codes for two unrelated rows).
		keySuffix := fmt.Sprintf("%d-%d", row.Year, i)

		// 3. Determine the amount that actually credits the wallet: what the owner really
		// transferred (Balance Bukti TF), not what the app told them to pay (Balance Top Up
		// System, which still includes the untouched unique-code suffix).
		balanceTf := cleanAmountString(row.BalanceTf)
		primaryAmount := balanceTf
		if primaryAmount == "" || primaryAmount == "0" {
			primaryAmount = cleanAmountString(row.BalanceSystem)
		}
		hadTopupThisRow := primaryAmount != "" && primaryAmount != "0"

		method, uniqueCode := inferPaymentMethod(row.MetodeBayar, row.BalanceSystem, row.BalanceTf)

		if hadTopupThisRow {
			topup := factory.WalletTopup{
				Amount:            primaryAmount,
				PaymentChannel:    row.MetodeBayar,
				PaymentMethod:     method,
				FeeAmount:         cleanAmountString(row.FeeMidtrans),
				SettlementAmount:  cleanAmountString(row.Settlement),
				UniqueCode:        uniqueCode,
				ExternalReference: fmt.Sprintf("EXCEL-TF-%s", keySuffix),
				IdempotencyKey:    fmt.Sprintf("excel-tf-%s", strings.ToLower(keySuffix)),
				PaidAt:            paidAt,
				Note:              "Import dari Excel New & Subscribe",
				Status:            "ACCEPTED",
			}
			if !transferDate.IsZero() {
				topup.TransferDateAt = transferDate
			}
			paymentID, err := fake.CreateWalletTopup(ctx, tx, ownerID, topup)
			if err != nil {
				return fmt.Errorf("create topup owner=%s: %w", row.OwnerCode, err)
			}
			if shouldSeedTransferForTopup(topup) {
				matchedTransfer := buildMatchedTransferForSubscribeTopup(keySuffix, topup, paymentID)
				if _, err := fake.CreateTransfer(ctx, tx, ownerID, matchedTransfer); err != nil {
					return fmt.Errorf("create matched transfer owner=%s: %w", row.OwnerCode, err)
				}
			}
		}

		// 4. Topup-only row (or nothing at all this row) — no closing/order, regardless of what
		// Tenor/Paket Membership/Status say (see hasPurchaseIntent doc comment).
		if !hasPurchaseIntent(row) {
			continue
		}

		tenor, _ := strconv.Atoi(row.Tenor)
		if tenor == 0 {
			tenor = 1
		}
		paketStr := strings.ToUpper(strings.TrimSpace(row.Paket))
		if paketStr == "" {
			paketStr = "BASIC"
		}
		if paketStr == "BISNIS" {
			paketStr = "BUSINESS"
		}
		planCode := fmt.Sprintf("%s_%02d_MONTHS", paketStr, tenor)

		note := "Import dari Excel New & Subscribe"
		// AllowBalanceShortfall stays true for every order (matching the original importer's
		// "Biarkan minus kalau balance kurang" behavior). Rows are read per-sheet/per-file and
		// merged across years, so replay order isn't guaranteed to exactly match the real
		// chronological order money moved for owners with multiple topup/purchase events — a
		// legitimate row can still land short even outside the Saldo Aplikasi case below. The
		// synthetic backfill immediately below exists to minimize how often that actually
		// happens, not to be the only thing preventing it; a real shortfall still gets recorded
		// on balance_shortfall_amount for later reconciliation instead of hard-failing the import.
		allowShortfall := true

		// 5. "Saldo Aplikasi": purchase happened but no topup evidence anywhere in this row —
		// the money came from an old leftover balance whose original topup predates the migrated
		// data range. Backfill a synthetic topup so the order below doesn't need the shortfall
		// escape hatch at all.
		if !hadTopupThisRow {
			backfillAmount := cleanAmountString(row.NominalAktivasi)
			if backfillAmount == "" || backfillAmount == "0" {
				var planPrice string
				lookupErr := tx.QueryRowContext(ctx,
					"SELECT CAST(price AS CHAR) FROM subscription_plans WHERE code = ? AND deleted_at IS NULL LIMIT 1",
					planCode).Scan(&planPrice)
				if lookupErr == nil {
					if cleaned := cleanAmountString(planPrice); cleaned != "" && cleaned != "0" {
						backfillAmount = cleaned
					}
				} else if lookupErr != sql.ErrNoRows {
					return fmt.Errorf("lookup plan price owner=%s plan=%s: %w", row.OwnerCode, planCode, lookupErr)
				}
			}

			if backfillAmount == "" || backfillAmount == "0" {
				// Amount can't be determined — no synthetic topup to inject, so this one relies
				// entirely on the shortfall escape hatch above. Say so explicitly in the note.
				note += " (approksimasi: saldo lama tidak dapat ditentukan)"
			} else {
				backfill := factory.WalletTopup{
					Amount:              backfillAmount,
					PaymentChannel:      "SALDO APLIKASI",
					PaymentMethod:       "SALDO_APLIKASI",
					IsSyntheticBackfill: true,
					ExternalReference:   fmt.Sprintf("EXCEL-BACKFILL-%s", keySuffix),
					IdempotencyKey:      fmt.Sprintf("excel-backfill-%s", strings.ToLower(keySuffix)),
					PaidAt:              aktivasiDate.AddDate(0, 0, -1),
					Status:              "ACCEPTED",
					Note:                "Saldo lama sebelum rentang data migrasi (2023-2026); sumber topup asli di luar cakupan.",
				}
				if _, err := fake.CreateWalletTopup(ctx, tx, ownerID, backfill); err != nil {
					return fmt.Errorf("create synthetic backfill topup owner=%s: %w", row.OwnerCode, err)
				}
			}
		}

		// 6. Create Subscription — via a Closing first, so the order chains owner→lead→closing→
		// subscription instead of landing directly as a "HANGING_ORDER" reconciliation issue.
		var closingID sql.NullInt64
		var leadID int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM customer_leads WHERE owner_id = ? LIMIT 1", ownerID).Scan(&leadID); err == nil {
			newClosingID, cErr := fake.CreateClosing(ctx, tx, leadID, factory.Closing{
				PlanCode: planCode,
				Status:   "CONFIRMED",
				Note:     note,
				ClosedAt: aktivasiDate,
			})
			if cErr != nil {
				if strings.Contains(cErr.Error(), "tidak ditemukan") || strings.Contains(cErr.Error(), "no rows in result set") {
					continue
				}
				return fmt.Errorf("create closing owner=%s: %w", row.OwnerCode, cErr)
			}
			closingID = sql.NullInt64{Int64: newClosingID, Valid: true}
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("lookup lead owner=%s: %w", row.OwnerCode, err)
		}

		order := factory.SubscriptionOrder{
			PlanCode:              planCode,
			ClosingID:             closingID,
			ExternalReference:     fmt.Sprintf("EXCEL-SUB-%s", keySuffix),
			IdempotencyKey:        fmt.Sprintf("excel-sub-%s", strings.ToLower(keySuffix)),
			PurchasedAt:           paidAt,
			SubscriptionStartDate: aktivasiDate,
			Note:                  note,
			AllowBalanceShortfall: allowShortfall,
		}
		if _, err := fake.CreateSubscriptionOrder(ctx, tx, ownerID, order); err != nil {
			if strings.Contains(err.Error(), "tidak ditemukan") || strings.Contains(err.Error(), "no rows in result set") {
				continue
			}
			return fmt.Errorf("create subscription owner=%s: %w", row.OwnerCode, err)
		}
	}
	return nil
}

func parseDateRobust(d string) time.Time {
	d = strings.TrimSpace(d)
	if d == "" || strings.EqualFold(d, "No Date") {
		return time.Time{}
	}
	if strings.Contains(d, " ") {
		parts := strings.Split(d, " ")
		if strings.Contains(parts[0], "-") || strings.Contains(parts[0], "/") {
			d = parts[0]
		}
	}
	// format dd/mm/yy
	if t, err := time.Parse("02/01/06", d); err == nil {
		return t
	}
	if t, err := time.Parse("02/01/2006", d); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", d); err == nil {
		return t
	}
	return time.Time{}
}

func shouldSeedTransferForTopup(topup factory.WalletTopup) bool {
	method := strings.ToUpper(strings.TrimSpace(topup.PaymentMethod))
	channel := strings.ToUpper(strings.TrimSpace(topup.PaymentChannel))
	return method != "SALDO_APLIKASI" && channel != "SALDO APLIKASI"
}

func buildMatchedTransferForSubscribeTopup(keySuffix string, topup factory.WalletTopup, paymentID int64) factory.Transfer {
	transferDate := topup.PaidAt
	if !topup.TransferDateAt.IsZero() {
		transferDate = topup.TransferDateAt
	}

	note := strings.TrimSpace(topup.Note)
	if note == "" {
		note = "Import dari Excel New & Subscribe"
	}

	return factory.Transfer{
		Amount:                 topup.Amount,
		TransferDate:           transferDate,
		Note:                   note + " | Auto-paired transfer untuk top up impor.",
		MatchStatus:            "MATCHED",
		MatchedWalletPaymentID: sql.NullInt64{Int64: paymentID, Valid: true},
		ExternalReference:      fmt.Sprintf("EXCEL-TRF-%s", keySuffix),
	}
}
