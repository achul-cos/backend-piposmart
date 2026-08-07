package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/factory"

	"github.com/xuri/excelize/v2"
)

const noPICIdentity = "Tanpa PIC"

// realShareEvent is one "Share N" lead hand-off event from the real Owner & Outlet Excel export —
// a real historical reassignment of the lead to a (possibly different) sales person, with the
// outcome category recorded at that point in time.
type realShareEvent struct {
	Date     time.Time
	RawName  string
	Category string
}

type realOwnerExcelRow struct {
	OwnerCode   string
	OwnerName   string
	OwnerPhone  string
	OwnerEmail  string
	BrandName   string
	OutletName  string
	OutletPhone string
	City        string
	Province    string
	Address     string
	DateOfWork  time.Time

	// Organizational columns — only populated by the "01. Owner & Outlet" file/sheets, which are
	// the only source that carries this data (see docs/sprint-16f).
	NamaPenginput string
	PIC           string
	StatusTerbaru string
	Akuisisi      string
	Shares        []realShareEvent
}

func parseMonthIndonesian(mStr string) time.Month {
	mStr = strings.ToLower(strings.TrimSpace(mStr))
	switch {
	case strings.HasPrefix(mStr, "jan"):
		return time.January
	case strings.HasPrefix(mStr, "feb"):
		return time.February
	case strings.HasPrefix(mStr, "mar"):
		return time.March
	case strings.HasPrefix(mStr, "apr"):
		return time.April
	case strings.HasPrefix(mStr, "mei"), strings.HasPrefix(mStr, "may"):
		return time.May
	case strings.HasPrefix(mStr, "jun"):
		return time.June
	case strings.HasPrefix(mStr, "jul"):
		return time.July
	case strings.HasPrefix(mStr, "agust"), strings.HasPrefix(mStr, "agus"), strings.HasPrefix(mStr, "aug"):
		return time.August
	case strings.HasPrefix(mStr, "sep"):
		return time.September
	case strings.HasPrefix(mStr, "okt"), strings.HasPrefix(mStr, "oct"):
		return time.October
	case strings.HasPrefix(mStr, "nov"):
		return time.November
	case strings.HasPrefix(mStr, "des"), strings.HasPrefix(mStr, "dec"):
		return time.December
	default:
		return time.January
	}
}

func getCol(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

// findHeaderIndex resolves a column by header name (exact, trimmed, case-insensitive match).
// The "01. Owner & Outlet" file's two sheets ("2025-2026" and "2021-2024") have a duplicate
// "Nama Pengisi" column in the older sheet that shifts every column after it by one position —
// columns after "STATUS TERBARU" (Akuisisi, PIC, Share N, ...) must be resolved by header name
// per-sheet rather than hardcoded index, or the older sheet's data silently misaligns.
func findHeaderIndex(header []string, name string) int {
	target := strings.ToUpper(strings.TrimSpace(name))
	for i, h := range header {
		if strings.ToUpper(strings.TrimSpace(h)) == target {
			return i
		}
	}
	return -1
}

// parseShareDate parses the "Tanggal Dibagikan N" columns, which use a distinct "02 January
// 2006" English-month layout (e.g. " 03 January 2025") not covered by parseDateRobust.
func parseShareDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "No Date") {
		return time.Time{}
	}
	if t, err := time.Parse("02 January 2006", raw); err == nil {
		return t
	}
	return parseDateRobust(raw)
}

func seedDemoReal(ctx context.Context, tx *sql.Tx, options Options) error {
	fake := factory.New(options.Seed, options.AsOf)
	rng := rand.New(rand.NewSource(options.Seed))

	// 1. Setup Team (Admin) — kept as the fallback/system admin used by other seed paths
	// (e.g. seedDemoReal's own bootstrap), separate from the real "Nama Penginput" admin
	// accounts created below.
	adminUser := fake.BuildUser("ADMIN", 1)
	adminUser.Email = "admin@piposmart.id"
	adminUser.Name = "Initial Admin"
	adminID, err := fake.CreateUser(ctx, tx, adminUser)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	// 2. Read All Authentic Real Owner Records from Excel files
	excelRows := readAllRealOwnerExcelFiles()

	if len(excelRows) == 0 {
		return fmt.Errorf("tidak ada data Excel yang ditemukan di asset/data_admin atau asset/data_sales")
	}

	// 3. Pre-pass: extract real admin ("Nama Penginput") and sales (PIC/Share) identities from
	// the whole dataset up front, so the main loop only does O(1) map lookups instead of a
	// CreateUser call per row (~11.8K rows). See docs/sprint-16f for the data shape this reads.
	adminIDByName, salesEmailByKey, supervisorID, err := seedRealIdentities(ctx, tx, fake, excelRows)
	if err != nil {
		return fmt.Errorf("seed real identities: %w", err)
	}

	// Untuk preset real: hanya iterasi data Excel yang ada, tidak boleh duplikat
	targetCount := len(excelRows)

	_ = rng

	timeline := generateGrowthTimeline(options.From, options.To, targetCount, options.Variation, rng)
	progress := newProgressBar(targetCount)

	seenOwners := make(map[string]int64)
	seenOutlets := make(map[string]bool)
	ownerOutletCount := make(map[string]int) // counter outlet per ownerCode

	for idx, defaultCreatedAt := range timeline {
		if idx >= len(excelRows) {
			break // jangan pernah melebihi data real
		}
		ownerIndex := idx + 1
		row := excelRows[idx]

		createdAt := defaultCreatedAt
		if !row.DateOfWork.IsZero() {
			createdAt = clampToAsOf(row.DateOfWork, options.AsOf)
		}

		var enteredBy sql.NullInt64
		if id, ok := adminIDByName[strings.TrimSpace(row.NamaPenginput)]; ok {
			enteredBy = sql.NullInt64{Int64: id, Valid: true}
		}

		ownerCode := row.OwnerCode
		if ownerCode == "" || len(ownerCode) > 20 || strings.Contains(ownerCode, " ") {
			if row.OwnerPhone != "" {
				ownerCode = "OWN-" + strings.TrimSpace(row.OwnerPhone)
			} else if row.OwnerName != "" {
				cleanName := strings.ToUpper(strings.ReplaceAll(row.OwnerName, " ", ""))
				if len(cleanName) > 10 {
					cleanName = cleanName[:10]
				}
				ownerCode = "OWN-" + cleanName
			} else {
				ownerCode = fmt.Sprintf("OWN-%05d", ownerIndex)
			}
		}

		var ownerID int64
		isNewOwner := false
		if existingID, exists := seenOwners[ownerCode]; exists {
			ownerID = existingID
		} else {
			isNewOwner = true
			ownerName := strings.TrimSpace(row.OwnerName)
			if ownerName == "" {
				ownerName = strings.TrimSpace(row.BrandName)
			}
			if ownerName == "" {
				ownerName = "-"
			}

			ownerPhone := row.OwnerPhone
			if len(ownerPhone) > 25 {
				ownerPhone = strings.TrimSpace(ownerPhone[:25])
			}
			if ownerPhone == "" {
				ownerPhone = "-"
			}

			brandName := row.BrandName
			if brandName == "" {
				brandName = ownerName
			}

			city := row.City
			province := row.Province

			owner := factory.Owner{
				Code:            ownerCode,
				Name:            ownerName,
				Phone:           ownerPhone,
				Email:           row.OwnerEmail,
				BrandName:       brandName,
				Province:        province,
				City:            city,
				Address:         row.Address,
				CreatedAt:       createdAt,
				EnteredByUserID: enteredBy,
			}

			newOwnerID, err := fake.CreateOwner(ctx, tx, owner)
			if err != nil {
				return fmt.Errorf("create real owner %d: %w", ownerIndex, err)
			}
			ownerID = newOwnerID
			seenOwners[ownerCode] = ownerID
		}

		outletName := row.OutletName
		if outletName == "" {
			outletName = strings.TrimSpace(row.BrandName)
			if outletName == "" {
				outletName = "-"
			}
		}

		outletKey := ownerCode + "|" + strings.ToLower(strings.TrimSpace(outletName))
		if seenOutlets[outletKey] {
			progress.update(ownerIndex)
			continue
		}
		seenOutlets[outletKey] = true

		outletPhone := row.OutletPhone
		if len(outletPhone) > 25 {
			outletPhone = strings.TrimSpace(outletPhone[:25])
		}

		city := row.City
		province := row.Province

		ownerOutletCount[ownerCode]++
		ownerOutletIdx := ownerOutletCount[ownerCode]
		outletCode := fmt.Sprintf("OUT-%05d-%02d", ownerIndex, ownerOutletIdx)
		if len(ownerCode) <= 20 {
			outletCode = fmt.Sprintf("OUT-%s-%02d", ownerCode, ownerOutletIdx)
		}
		outlet := factory.Outlet{
			Code:            outletCode,
			Name:            outletName,
			Phone:           outletPhone,
			Province:        province,
			City:            city,
			Address:         row.Address,
			EnteredByUserID: enteredBy,
		}
		outletID, err := fake.CreateOutlet(ctx, tx, ownerID, outlet)
		if err != nil {
			return fmt.Errorf("create real outlet owner=%d: %w", ownerIndex, err)
		}
		// Ensure outlet created_at is historical
		_, _ = tx.ExecContext(ctx, "UPDATE outlets SET created_at = ? WHERE id = ?", createdAt, outletID)

		// customer_leads is unique per owner, not per outlet — only build the lead + its
		// assignment/interaction history off the FIRST outlet row seen for a given owner.
		if isNewOwner {
			if err := seedRealLeadForOwner(ctx, tx, fake, ownerID, outletID, ownerCode, row, createdAt, enteredBy, supervisorID, salesEmailByKey, options.AsOf); err != nil {
				return fmt.Errorf("create real lead owner=%d: %w", ownerIndex, err)
			}
		}

		progress.update(ownerIndex)
	}

	progress.finish()

	// 4. Chain subscription/closing data from the "02. New & Subscribe" file onto the owner→lead
	// data just seeded above (needs the leads created in the loop, so runs after it).
	if err := SeedSubscriptionsFromExcel(ctx, tx, fake, adminID); err != nil {
		return fmt.Errorf("seed subscriptions from excel: %w", err)
	}

	return nil
}

func readAllRealOwnerExcelFiles() []realOwnerExcelRow {
	var results []realOwnerExcelRow

	searchDirs := []string{
		filepath.Join("asset", "data_admin"),
		filepath.Join("..", "asset", "data_admin"),
		filepath.Join("backend", "asset", "data_admin"),
	}

	validFiles := []string{
		"01. Owner & Outlet 2021 - 2024 & 2025 - 2026.xlsx",
	}

	var files []string
	for _, dir := range searchDirs {
		for _, v := range validFiles {
			files = append(files, filepath.Join(dir, v))
		}
	}

	for _, fn := range files {
		f, err := excelize.OpenFile(fn)
		if err != nil {
			continue
		}

		for _, sheet := range f.GetSheetList() {
			rows, err := f.GetRows(sheet)
			if err != nil || len(rows) <= 1 {
				continue
			}

			// 1. File "01. Owner & Outlet" — mapping kolom spesifik sesuai header aslinya
			// Row[0] = header: col0=No, col1=Date of Work, col2=Nama Penginput, col3=Kategori Akun,
			// col4=Kode Baris, col5=Kode Owner, col6=Nama Owner, col7=Email Owner,
			// col8=No Hp Owner, col9=No. Hp Outlet, col11=Create Date, col12=Bulan,
			// col13=Nama Project/BRAND, col14=Nama Outlet, col17=Kota, col18=Provinsi, col19=Alamat Lengkap
			if strings.Contains(fn, "01. Owner") {
				// Mapping kolom EKSAK dari header sheet "Owner & Outlet 2025-2026":
				// [0]=No, [1]=Date of Work, [2]=Nama Penginput, [3]=Kategori Akun,
				// [4]=Kode Baris, [5]=Kode Owner, [6]=Nama Owner, [7]=Email Owner,
				// [8]=No Hp Owner, [9]=No. Hp Outlet, [10]=Create Date Project,
				// [11]=Bulan, [12]=Nama Project/BRAND, [13]=Nama Outlet,
				// [14]=Kelurahan, [15]=Kecamatan, [16]=Kota, [17]=Provinsi, [18]=Alamat Lengkap
				const (
					colOwnerCode   = 5
					colOwnerName   = 6
					colOwnerEmail  = 7
					colOwnerPhone  = 8
					colOutletPhone = 9
					colCreateDate  = 10 // "Create Date Project"
					colBrandName   = 12 // "Nama Project/BRAND"
					colOutletName  = 13 // "Nama Outlet"
					colKota        = 16 // "Kota"
					colProvinsi    = 17 // "Provinsi"
					colAlamat      = 18 // "Alamat Lengkap"
					colNamaPeng    = 2  // "Nama Penginput" — before the sheet-layout divergence, safe hardcoded
					colStatus      = 21 // "STATUS TERBARU" — also before the divergence
				)

				// Columns after STATUS TERBARU (Akuisisi, PIC, Share N) shift by one position on
				// the older "2021-2024" sheet (it has an extra duplicate "Nama Pengisi" column) —
				// resolve those by header name per-sheet instead of a fixed index.
				header := rows[0]
				colAkuisisi := findHeaderIndex(header, "Akuisisi")
				colPIC := findHeaderIndex(header, "PIC")
				type shareCols struct{ tanggal, share, kategori int }
				var shareColIdx [5]shareCols
				for n := 0; n < 5; n++ {
					tCol := findHeaderIndex(header, fmt.Sprintf("Tanggal Dibagikan %d", n+1))
					sCol := findHeaderIndex(header, fmt.Sprintf("Share %d", n+1))
					kCol := -1
					if sCol != -1 {
						kCol = sCol + 1 // "Kategori Nasabah" always immediately follows "Share N"
					}
					shareColIdx[n] = shareCols{tanggal: tCol, share: sCol, kategori: kCol}
				}

				for rIdx := 1; rIdx < len(rows); rIdx++ {
					row := rows[rIdx]
					if len(row) < 14 {
						continue
					}

					// Hapus pengecekan `#ref`, `#value`, `#n/a` sesuai instruksi agar total pas 11.792

					code := getCol(row, colOwnerCode)
					name := getCol(row, colOwnerName)
					brand := getCol(row, colBrandName)
					outlet := getCol(row, colOutletName)

					// Hapus pengecekan if code=="" && name=="" && brand=="" agar masuk semua baris

					ownerPhone := getCol(row, colOwnerPhone)
					outletPhone := getCol(row, colOutletPhone)
					phone := ownerPhone
					if phone == "" {
						phone = outletPhone
					}

					rawDate := getCol(row, colCreateDate)
					var dt time.Time
					for _, layout := range []string{"02/01/2026", "02/01/06", "02/01/2006", "2006-01-02", "01/02/2006"} {
						if t, err := time.Parse(layout, rawDate); err == nil {
							dt = t
							break
						}
					}
					if dt.IsZero() {
						dt = parseDateRobust(rawDate)
					}

					var shares []realShareEvent
					for n := 0; n < 5; n++ {
						sc := shareColIdx[n]
						shareName := getCol(row, sc.share)
						if shareName == "" {
							continue
						}
						shares = append(shares, realShareEvent{
							Date:     parseShareDate(getCol(row, sc.tanggal)),
							RawName:  shareName,
							Category: getCol(row, sc.kategori),
						})
					}

					results = append(results, realOwnerExcelRow{
						OwnerCode:     code,
						OwnerName:     name,
						BrandName:     brand,
						OutletName:    outlet,
						OwnerPhone:    ownerPhone,
						NamaPenginput: getCol(row, colNamaPeng),
						PIC:           getCol(row, colPIC),
						StatusTerbaru: getCol(row, colStatus),
						Akuisisi:      getCol(row, colAkuisisi),
						Shares:        shares,
						OwnerEmail:    getCol(row, colOwnerEmail),
						OutletPhone:   outletPhone,
						City:          getCol(row, colKota),
						Province:      getCol(row, colProvinsi),
						Address:       getCol(row, colAlamat),
						DateOfWork:    dt,
					})
				}
				continue
			}

			if len(rows) <= 3 {
				continue
			}

			// 2. Province sheets pattern in 03. Nasabah Baru
			if strings.Contains(fn, "03. Nasabah Baru") && sheet != "TOTAL NASABAH BARU" && sheet != "Kode Wilayah" {
				for rIdx := 4; rIdx < len(rows); rIdx++ {
					row := rows[rIdx]
					if len(row) < 5 {
						continue
					}
					tahun := getCol(row, 1)
					bulanStr := getCol(row, 2)
					code := getCol(row, 3)
					name := getCol(row, 4)
					phone := getCol(row, 5)
					email := getCol(row, 6)
					brand := getCol(row, 7)
					city := getCol(row, 8)
					addr := getCol(row, 9)

					if name == "" && brand == "" {
						continue
					}

					key := code
					if key == "" {
						key = phone
					}
					if key == "" {
						key = name
					}

					var dt time.Time
					if yr, err := strconv.Atoi(tahun); err == nil && yr >= 2020 {
						month := parseMonthIndonesian(bulanStr)
						dt = time.Date(yr, month, 15, 0, 0, 0, 0, time.UTC)
					}

					results = append(results, realOwnerExcelRow{
						OwnerCode:  code,
						OwnerName:  name,
						OwnerPhone: phone,
						OwnerEmail: email,
						BrandName:  brand,
						OutletName: brand,
						City:       city,
						Province:   sheet,
						Address:    addr,
						DateOfWork: dt,
					})
				}
				continue
			}

			// 3. Standard Header matching for all other sheets
			headerRowIdx := -1
			var colCode, colName, colBrand, colOutlet, colPhone, colEmail, colCity, colProv, colAddr, colDate int = -1, -1, -1, -1, -1, -1, -1, -1, -1, -1

			for rIdx := 0; rIdx < len(rows) && rIdx < 10; rIdx++ {
				row := rows[rIdx]
				for cIdx, cell := range row {
					u := strings.ToUpper(strings.TrimSpace(cell))
					switch {
					case u == "KODE OWNER" || u == "KODE NASABAH" || u == "KODE":
						colCode = cIdx
					case u == "NAMA OWNER" || u == "NAMA NASABAH" || u == "NAMA":
						colName = cIdx
					case strings.Contains(u, "BRAND") || strings.Contains(u, "PROJECT") || u == "NAMA BRAND / OUTLET":
						colBrand = cIdx
					case u == "NAMA OUTLET" || u == "OUTLET":
						colOutlet = cIdx
					case strings.Contains(u, "NO HP") || strings.Contains(u, "NO. HP") || u == "HP" || u == "TELEPON":
						if colPhone == -1 {
							colPhone = cIdx
						}
					case strings.Contains(u, "EMAIL"):
						colEmail = cIdx
					case strings.Contains(u, "KOTA") || strings.Contains(u, "KABUPATEN"):
						colCity = cIdx
					case strings.Contains(u, "PROVINSI"):
						colProv = cIdx
					case strings.Contains(u, "ALAMAT"):
						colAddr = cIdx
					case strings.Contains(u, "DATE") || strings.Contains(u, "TANGGAL") || u == "CREATED":
						if colDate == -1 {
							colDate = cIdx
						}
					}
				}
				if colName != -1 || colBrand != -1 || colCode != -1 {
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

				// Skip baris yang ada error formula di kolom mana saja
				hasError := false
				for _, cell := range row {
					if strings.Contains(cell, "#REF!") || strings.Contains(cell, "#VALUE!") || strings.Contains(cell, "#N/A") {
						hasError = true
						break
					}
				}
				if hasError {
					continue
				}

				code := getCol(row, colCode)
				name := getCol(row, colName)
				brand := getCol(row, colBrand)
				outlet := getCol(row, colOutlet)
				phone := getCol(row, colPhone)
				email := getCol(row, colEmail)
				city := getCol(row, colCity)
				prov := getCol(row, colProv)
				addr := getCol(row, colAddr)
				rawDate := getCol(row, colDate)

				if code == "" && name == "" && brand == "" {
					continue
				}

				if strings.Contains(strings.ToUpper(getCol(row, 0)), "TOTAL") {
					continue
				}

				key := code
				if key == "" {
					key = phone
				}
				if key == "" {
					key = name
				}

				var dt time.Time
				if rawDate != "" {
					if t, err := time.Parse("2006-01-02", rawDate); err == nil {
						dt = t
					} else if t, err := time.Parse("02/01/06", rawDate); err == nil {
						dt = t
					} else if t, err := time.Parse("02/01/2006", rawDate); err == nil {
						dt = t
					}
				}

				results = append(results, realOwnerExcelRow{
					OwnerCode:  code,
					OwnerName:  name,
					OwnerPhone: phone,
					OwnerEmail: email,
					BrandName:  brand,
					OutletName: outlet,
					City:       city,
					Province:   prov,
					Address:    addr,
					DateOfWork: dt,
				})
			}
		}
		f.Close()
	}

	return results
}

// realAssignmentEvent is one point in a lead's chronologically-replayed real assignment history —
// either a "Share N" hand-off or, appended last, the row's current "PIC" (unless it already
// matches the last Share event, in which case it would be a no-op reassignment and is skipped).
type realAssignmentEvent struct {
	Date     time.Time
	Key      string
	Name     string
	Category string
}

// seedRealLeadForOwner creates the customer_leads row for a real-seeded owner (called once, off
// the first outlet row seen for that owner, since customer_leads is unique per owner) and replays
// its real "PIC"/"Share 1..5" assignment history as lead_assignments + customer_interactions —
// data-driven from the Excel export, unlike the synthetic 5-stage generator this replaces.
func seedRealLeadForOwner(
	ctx context.Context,
	tx *sql.Tx,
	fake *factory.Factory,
	ownerID, outletID int64,
	ownerCode string,
	row realOwnerExcelRow,
	createdAt time.Time,
	enteredBy sql.NullInt64,
	supervisorID int64,
	salesEmailByKey map[string]string,
	asOf time.Time,
) error {
	var events []realAssignmentEvent
	for _, sh := range row.Shares {
		if sh.RawName == "" {
			continue
		}
		d := sh.Date
		if d.IsZero() {
			d = createdAt
		}
		events = append(events, realAssignmentEvent{
			Date:     d,
			Key:      salesIdentityKey(sh.RawName),
			Name:     normalizeSalesName(sh.RawName),
			Category: sh.Category,
		})
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Date.Before(events[j].Date) })

	if row.PIC != "" {
		picKey := salesIdentityKey(row.PIC)
		lastKey, lastDate := "", createdAt
		if len(events) > 0 {
			lastKey = events[len(events)-1].Key
			lastDate = events[len(events)-1].Date
		}
		if picKey != lastKey {
			events = append(events, realAssignmentEvent{Date: lastDate, Key: picKey, Name: normalizeSalesName(row.PIC)})
		}
	}

	if len(events) == 0 {
		events = append(events, realAssignmentEvent{
			Date: createdAt,
			Key:  salesIdentityKey(noPICIdentity),
			Name: normalizeSalesName(noPICIdentity),
		})
	}

	firstEmail, ok := salesEmailByKey[events[0].Key]
	if !ok {
		firstEmail = salesEmailByKey[salesIdentityKey(noPICIdentity)]
	}

	stage, status := "NEW", "OPEN"
	lastCategory := events[len(events)-1].Category
	if strings.Contains(strings.ToUpper(row.Akuisisi), "GAGAL") ||
		strings.Contains(strings.ToUpper(row.StatusTerbaru), "GAGAL") ||
		strings.EqualFold(strings.TrimSpace(lastCategory), "Tidak Tertarik") {
		stage, status = "INVALID", "INVALID"
	}

	lead := factory.Lead{
		Code:             fmt.Sprintf("LEAD-%s", ownerCode),
		SourceType:       "IMPORT",
		SourceReference:  "excel:01-owner-outlet",
		Stage:            stage,
		Status:           status,
		NextFollowUpAt:   clampToAsOf(createdAt.AddDate(0, 0, 7), asOf),
		ActiveSalesEmail: firstEmail,
		EnteredByUserID:  enteredBy,
	}
	leadID, err := fake.CreateLead(ctx, tx, ownerID, outletID, lead)
	if err != nil {
		return fmt.Errorf("create lead %s: %w", lead.Code, err)
	}
	// CreateLead's own bootstrap lead_assignments row always dates itself to the seed run's asOf
	// (today), not the real historical first-assignment date — left as-is it sorts chronologically
	// LAST instead of first. Backdate it to the real first event so history reads correctly.
	if _, err := tx.ExecContext(ctx, `
		UPDATE lead_assignments
		SET action = 'REAL_INITIAL_ASSIGNMENT', started_at = ?
		WHERE lead_id = ? AND action = 'DEMO_ASSIGNED_TO_SALES'`,
		events[0].Date, leadID,
	); err != nil {
		return fmt.Errorf("backdate lead %s initial assignment: %w", lead.Code, err)
	}

	prevSalesID, err := lookupRealUserIDByEmail(ctx, tx, firstEmail)
	if err != nil {
		return err
	}
	if err := createRealShareInteraction(ctx, tx, fake, leadID, events[0]); err != nil {
		return fmt.Errorf("create lead %s interaction 0: %w", lead.Code, err)
	}

	for i := 1; i < len(events); i++ {
		ev := events[i]
		email, ok := salesEmailByKey[ev.Key]
		if !ok {
			continue
		}
		toSalesID, err := lookupRealUserIDByEmail(ctx, tx, email)
		if err != nil {
			return err
		}
		reason := fmt.Sprintf("Share ke %s", ev.Name)
		if ev.Category != "" {
			reason += " — " + ev.Category
		}
		if err := fake.ReassignLead(ctx, tx, leadID, ownerID, prevSalesID, toSalesID, supervisorID,
			"REAL_SHARE_REASSIGNED", reason, remarkScoreForCategory(ev.Category), ev.Date); err != nil {
			return fmt.Errorf("create lead %s reassignment %d: %w", lead.Code, i, err)
		}
		if err := createRealShareInteraction(ctx, tx, fake, leadID, ev); err != nil {
			return fmt.Errorf("create lead %s interaction %d: %w", lead.Code, i, err)
		}
		prevSalesID = toSalesID
	}

	return nil
}

func createRealShareInteraction(ctx context.Context, tx *sql.Tx, fake *factory.Factory, leadID int64, ev realAssignmentEvent) error {
	at := ev.Date
	if at.IsZero() {
		at = time.Now().UTC()
	}
	note := fmt.Sprintf("Share ke %s", ev.Name)
	if ev.Category != "" {
		note += " — " + ev.Category
	}
	// Append the date so repeated identical share events (same sales + same category, which does
	// occur in the real data) don't collide on CreateInteraction's note-based dedup delete.
	note += " (" + at.Format("2006-01-02") + ")"
	_, err := fake.CreateInteraction(ctx, tx, leadID, factory.Interaction{
		Type:          "NOTE",
		InteractionAt: at,
		RemarkScore:   remarkScoreForCategory(ev.Category),
		Note:          note,
		// CreateInteraction always writes FollowUpAt as-is (no NULL-guard for a zero time.Time) —
		// every other caller in this package sets a real date for the same reason; match that.
		FollowUpAt: at.AddDate(0, 0, 3),
	})
	return err
}

func remarkScoreForCategory(category string) int {
	if strings.EqualFold(strings.TrimSpace(category), "Tidak Tertarik") {
		return 0 // INVALID
	}
	return 1 // POSSIBLE — neutral default; only 4 remark_reasons buckets exist and Excel
	// category text is too free-form to map precisely beyond "explicitly not interested".
}

func lookupRealUserIDByEmail(ctx context.Context, tx *sql.Tx, email string) (int64, error) {
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup users.email=%s: %w", email, err)
	}
	return id, nil
}
